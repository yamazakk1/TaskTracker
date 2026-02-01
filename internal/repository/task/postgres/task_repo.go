package postgres

import (
	"context"
	"fmt"
	"taskTracker/internal/logger"
	"taskTracker/internal/models/task"
	repo "taskTracker/internal/repository"
	"time"
	"os"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Storage struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, connString string) (*Storage, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logger.Error("Repository: Ошибка загрузки конфига", err)
		return nil, fmt.Errorf("загрузка конфига: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = time.Minute * 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		logger.Error("Repository: Ошибка создания пула", err)
		return nil, fmt.Errorf("создание пула: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		logger.Error("Repository: Неудачная проверка ping", err)
		return nil, fmt.Errorf("проверка соединения ping: %w", err)
	}

	logger.Info("Repository: Успешное создание подключения к PostgreSQL")
	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() {
	s.pool.Close()
	logger.Info("Repository: Закрытие всех соединений PostgreSQL")
}

func (s *Storage) HealthCheck(ctx context.Context) error {
	err := s.pool.Ping(ctx)
	if err != nil {
		logger.Error("Repository: Неудачная проверка ping", err)
		return fmt.Errorf("проверка соединения ping: %w", err)
	}
	logger.Info("Repository: Соединение стабильно")
	return nil
}

// мягкое удаление задачи
func (s *Storage) DeleteSoft(ctx context.Context, taskToDelete *task.Task) error {
	start := time.Now()

	query := `UPDATE tasks
				SET deleted_at = NOW(),
				flag = $1,
				version = version + 1
			WHERE uuid = $2 AND version = $3 
			RETURNING deleted_at, version`

	err := s.pool.QueryRow(ctx, query, task.FlagDeleted, taskToDelete.UUID, taskToDelete.Version).Scan(&taskToDelete.DeletedAt, &taskToDelete.Version)

	if err != nil {
		if err == pgx.ErrNoRows {
			logger.Warn("Конфликт версий при мягком удалении",
				zap.String("task_id", taskToDelete.UUID.String()),
				zap.Int("expected_version", taskToDelete.Version))
			return repo.ErrVersionConflict
		}

		logger.Error("Repository: Мягкое удаление задачи", err, zap.Duration("ms", time.Since(start)))
		return fmt.Errorf("мягкое удаление: %w", err)
	}

	if time.Since(start) > time.Millisecond*100 {
		logger.Warn("Repository: Медленная операция", zap.Duration("ms", time.Since(start)))
	}
	return err
}

// полное удаление из БД
func (s *Storage) DeleteFull(ctx context.Context, uuid uuid.UUID) error {
	start := time.Now()

	query := `DELETE FROM tasks
				WHERE uuid = $1`

	_, err := s.pool.Exec(ctx, query, uuid)

	if err != nil {
		logger.Error("Repositry: Полное уделание задачи", err, zap.Duration("ms", time.Since(start)))
		return fmt.Errorf("полное удаление: %w", err)
	}

	if time.Since(start) > time.Millisecond*100 {
		logger.Warn("Repository: Медленная операция", zap.Duration("ms", time.Since(start)))
	}

	return nil
}

func (s *Storage) Update(ctx context.Context, taskToUpdate *task.Task) error {
	start := time.Now()

	query := `UPDATE tasks
			SET title = $1,
				description = $2,
				status = $3,
				due_time = $4,
				version = version + 1,
				updated_at = NOW(),
				flag = $5
			WHERE uuid = $6 AND version = $7
			RETURNING updated_at, version`

	err := s.pool.QueryRow(ctx, query,
		taskToUpdate.Title,
		taskToUpdate.Description,
		taskToUpdate.Status,
		taskToUpdate.DueTime,
		taskToUpdate.Flag,
		taskToUpdate.UUID,
		taskToUpdate.Version,
	).Scan(&taskToUpdate.UpdatedAt, &taskToUpdate.Version)

	if err != nil {
		if err == pgx.ErrNoRows {
			logger.Warn("Конфликт версий при обновлении задачи",
				zap.String("task_id", taskToUpdate.UUID.String()),
				zap.Int("expected_version", taskToUpdate.Version))
			return repo.ErrVersionConflict // Нужно добавить эту ошибку
		}
		logger.Error("Repository: Не удалось обновить задачу", err)
		return fmt.Errorf("обновление задачи: %w", err)
	}

	if time.Since(start) > time.Millisecond*100 {
		logger.Warn("Repository: Медленная операция", zap.Duration("ms", time.Since(start)))
	}
	return err
}

func (s *Storage) Create(ctx context.Context, taskToCreate *task.Task) error {
	start := time.Now()

	query := `INSERT INTO tasks
				(uuid, title, description, status, due_time, created_at, flag)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING created_at`

	err := s.pool.QueryRow(ctx, query,
		taskToCreate.UUID,
		taskToCreate.Title,
		taskToCreate.Description,
		taskToCreate.Status,
		taskToCreate.DueTime,
		time.Now(),
		task.FlagActive,
	).Scan(&taskToCreate.CreatedAt)

	if err != nil {
		logger.Error("Repository: Не удалось добавить задачу", err, zap.Duration("ms", time.Since(start)))
		return fmt.Errorf("добавление задачи: %w", err)
	}

	if time.Since(start) > time.Millisecond*50 {
		logger.Warn("Repository: Медленный запрос", zap.Duration("ms", time.Since(start)))
	}
	return nil
}

func (s *Storage) GetByID(ctx context.Context, uuid uuid.UUID) (*task.Task, error) {
	start := time.Now()

	query := `SELECT 
				uuid,
				title,
				description,
				status,
				due_time,
				created_at,
				updated_at,
				deleted_at,
				version,
				flag 
				FROM tasks
				WHERE uuid = $1`

	task := &task.Task{}
	err := s.pool.QueryRow(ctx, query, uuid).Scan(
		&task.UUID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.DueTime,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.DeletedAt,
		&task.Version,
		&task.Flag,
	)

	if err != nil {
		logger.Error("Repository: Не удалось получить задачу", err, zap.Duration("ms", time.Since(start)))
		return nil, fmt.Errorf("получение задачи: %w", err)
	}

	if time.Since(start) > time.Millisecond*100 {
		logger.Warn("Repository: Медленный запрос", zap.Duration("ms", time.Since(start)))
	}

	return task, nil
}

// все задачи с флагами active или archived
func (s *Storage) GetAllWithLimit(ctx context.Context, page, limit int) ([]*task.Task, error) {
	start := time.Now()
	offset := (page - 1) * limit
	query := `SELECT
				uuid,
				title,
				description,
				status,
				due_time,
				created_at,
				updated_at,
				deleted_at,
				version,
				flag
				FROM tasks
				WHERE flag != $1
				LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, query, task.FlagDeleted, limit, offset)
	if err != nil {
		logger.Error("Repository: Не удалось получить задачи", err, zap.Duration("ms", time.Since(start)))
		return nil, fmt.Errorf("получение задач: %w", err)
	}

	defer rows.Close()

	tasks := []*task.Task{}

	for rows.Next() {
		task := &task.Task{}

		err := rows.Scan(
			&task.UUID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.DueTime,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.DeletedAt,
			&task.Version,
			&task.Flag,
		)

		if err != nil {
			logger.Warn("Repository: Ошибка сканирования задачи", zap.Error(err))
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		logger.Error("Repository: Ошибка иерации по строкам", err)
		return nil, fmt.Errorf("итерация по строкам: %w", err)
	}

	if time.Since(start) > time.Millisecond*50+time.Millisecond*10*time.Duration(limit) {
		logger.Warn("Repository: Медленный запрос", zap.Duration("ms", time.Since(start)))
	}

	return tasks, nil

}

// получение задач с определённым статусом
func (s *Storage) GetStatusedWithLimit(ctx context.Context, page, limit int, status task.Status) ([]*task.Task, error) {
	start := time.Now()

	offset := (page - 1) * limit
	query := `SELECT
				uuid,
				title,
				description,
				status,
				due_time,
				created_at,
				updated_at,
				deleted_at,
				version,
				flag
				FROM tasks
				WHERE STATUS = $1
				LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		logger.Error("Repository: Не удалось получить задачи", err, zap.Duration("ms", time.Since(start)))
		return nil, fmt.Errorf("получение задач: %w", err)
	}

	defer rows.Close()

	tasks := []*task.Task{}
	for rows.Next() {
		task := &task.Task{}

		err := rows.Scan(
			&task.UUID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.DueTime,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.DeletedAt,
			&task.Version,
			&task.Flag,
		)
		if err != nil {
			logger.Warn("Repository: Ошибка сканирования задачи", zap.Error(err))
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		logger.Error("Repository: Ошибка итерации по строкам", err)
		return nil, fmt.Errorf("итерация по строкам: %w", err)
	}

	if time.Since(start) > time.Millisecond*50+time.Millisecond*10*time.Duration(limit) {
		logger.Warn("Repository: Медленный запрос", zap.Duration("ms", time.Since(start)))
	}

	return tasks, nil
}

// получение задачи с определённым флагом
func (s *Storage) GetFlaggedWithLimit(ctx context.Context, page, limit int, flag task.Flag) ([]*task.Task, error) {
	start := time.Now()

	offset := (page - 1) * limit
	query := `SELECT
				uuid,
				title,
				description,
				status,
				due_time,
				created_at,
				updated_at,
				deleted_at,
				version,
				flag
				FROM tasks
				WHERE flag = $1
				LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, query, flag, limit, offset)
	if err != nil {
		logger.Error("Repository: Не удалось получить задачи", err, zap.Duration("ms", time.Since(start)))
		return nil, fmt.Errorf("получение задач: %w", err)
	}

	defer rows.Close()

	tasks := []*task.Task{}
	for rows.Next() {
		task := &task.Task{}

		err := rows.Scan(
			&task.UUID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.DueTime,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.DeletedAt,
			&task.Version,
			&task.Flag,
		)
		if err != nil {
			logger.Warn("Repository: Ошибка сканирования задачи", zap.Error(err))
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		logger.Error("Repository: Ошибка итерации по строкам", err)
		return nil, fmt.Errorf("итерация по строкам: %w", err)
	}

	if time.Since(start) > time.Millisecond*50+time.Millisecond*10*time.Duration(limit) {
		logger.Warn("Repository: Медленный запрос", zap.Duration("ms", time.Since(start)))
	}

	return tasks, nil
}

func (s *Storage) GetTasksDueBefore(ctx context.Context, deadline time.Time, limit int) ([]*task.Task, error){
	start := time.Now()

	query := `SELECT * FROM tasks 
              WHERE flag = 'active' 
                AND status NOT IN ('done', 'overdue')
                AND due_time < $1
              LIMIT $2`

	rows, err := s.pool.Query(ctx, query, deadline, limit)
	if err != nil{
		logger.Error("Repository: Не удалось получить задачи", err, zap.Duration("ms", time.Since(start)))
		return nil, fmt.Errorf("получение задач: %w", err)
	}

	tasks := []*task.Task{}
	for rows.Next(){
		task := &task.Task{}

		err := rows.Scan(
			&task.UUID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.DueTime,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.DeletedAt,
			&task.Version,
			&task.Flag,
		)

		if err != nil{
			logger.Warn("Repository: Ошибка сканирования задачи", zap.Error(err))
		}

		tasks = append(tasks, task)
	}
	
	if err := rows.Err(); err != nil{
		logger.Error("Repository: Ошибка итерации по строкам", err)
		return nil, fmt.Errorf("итерация по строкам: %w", err)
	}

	if time.Since(start) > time.Millisecond*50+time.Millisecond*10*time.Duration(limit) {
		logger.Warn("Repository: Медленный запрос", zap.Duration("ms", time.Since(start)))
	}
	
	return tasks, nil
}


func (s *Storage)Migrate(ctx context.Context) error {
	logger.Info("Попытка миграций")
	
	initUp, err := os.ReadFile("internal/migrations/001_init.up.sql")
	if err != nil {
		logger.Error("failed to read 001_init.up.sql", err)
		return err
	}
	
	indexesUp, err := os.ReadFile("internal/migrations/002_indexes.up.sql")
	if err != nil {
		
		logger.Error("failed to read 002_indexes.up.sql", err)
		return err
	}
	
		_, err = s.pool.Exec(ctx,string(initUp))
	if err != nil {
		logger.Error("failed to apply 001_init", err)
		return err 
	}
	
	_, err = s.pool.Exec(ctx,string(indexesUp))
	if err != nil {
		logger.Error("failed to apply 002_indexes", err)
		return err
	}
	
	logger.Info("Христа ради миграции заработали")
	return nil
}


func (s *Storage )Down(ctx context.Context) error {
	logger.Info("Откат миграций")

	indexesDown, err :=	os.ReadFile("internal/migrations/002_indexes.down.sql")
	if err != nil {
		logger.Error("failed to read 002_indexes.down.sql", err)
		return err
	}
	
	initDown, err := os.ReadFile("internal/migrations/001_init.down.sql")
	if err != nil {
		logger.Error("failed to read 001_init.down.sql", err)
		return err
	}
	
	_, err = s.pool.Exec(ctx, string(indexesDown))
	if err != nil {
		logger.Error("failed to rollback 002_indexes", err)
		return  err
	}
	
	_, err = s.pool.Exec(ctx, string(initDown))
	if err != nil {
		logger.Error("failed to rollback 001_init", err)
		return err
	}
	
	logger.Info("Migrations rolled back successfully!")
	return nil
}