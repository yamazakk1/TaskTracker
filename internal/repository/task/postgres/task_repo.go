package postgres

import (
	"context"
	"fmt"
	"taskTracker/internal/logger"
	"taskTracker/internal/models/task"
	"time"


	"github.com/google/uuid"
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


func (s *Storage) Delete(ctx context.Context, task *task.Task) error{
	start := time.Now()

	query := `UPDATE tasks
				SET deleted_at = NOW(),
				status = $1,
				version = version + 1
			WHERE uuid = $2 AND version = $3 AND deleted_at IS NULL
			RETURNING deleted_at, version`

	err := s.pool.QueryRow(ctx, query, task.Status, task.UUID, task.Version).Scan(&task.DeletedAt, &task.Version)

	if err != nil{
		logger.Error("Repository: Мягкое удаление задачи", err, zap.Duration("ms", time.Since(start)))
		return fmt.Errorf("мягкое удаление: %w", err)
	}

	
	if time.Since(start) > time.Millisecond * 100{
		logger.Warn("Repository: Медленная операция", zap.Duration("ms", time.Since(start)))
	}
	return err 
}

func (s *Storage) Update(ctx context.Context, task *task.Task) error {
	start := time.Now()

	query := `UPDATE tasks
			SET title = $1,
				description = $2,
				status = $3,
				due_time = $4,
				version = version + 1,
				updated_at = NOW()
			WHERE uuid = $5 AND version = $6
			RETURNING updated_at, version`

	err := s.pool.QueryRow(ctx, query,
		task.Title,
		task.Description,
		task.Status,
		task.DueTime,
		task.UUID,
		task.Version,
	).Scan(&task.UpdatedAt, &task.Version)
	
	if err != nil{
		logger.Error("Repository: Не удалось обновить задачу", err)
		return fmt.Errorf("обновление задачи: %w", err)
	}

	if time.Since(start) > time.Millisecond * 100{
		logger.Warn("Repository: Медленная операция", zap.Duration("ms", time.Since(start)))
	}
	return err 
}

func (s *Storage) Create(ctx context.Context, task *task.Task) error {
	start := time.Now()

	task.CreatedAt = time.Now()

	query := `INSERT INTO tasks
				(uuid, title, description, status, due_time, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING created_at`

	err := s.pool.QueryRow(ctx, query,
		task.UUID,
		task.Title,
		task.Description,
		task.Status,
		task.DueTime,
		time.Now(),
	).Scan(&task.CreatedAt)

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
				version 
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

func (s *Storage) GetWIthLimit(ctx context.Context, limit int) ([]*task.Task, error) {
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
				version
				FROM tasks
				LIMIT $1`

	rows, err := s.pool.Query(ctx, query, limit)
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
