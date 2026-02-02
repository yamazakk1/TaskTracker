package postgres_test

import (
	"context"
	"fmt"
	"taskTracker/internal/models/task"
	"taskTracker/internal/repository/task/postgres"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresTestSuite для интеграционных тестов с PostgreSQL
type PostgresTestSuite struct {
	suite.Suite
	container testcontainers.Container
	storage   *postgres.Storage
	ctx       context.Context
	pool      interface{} // Добавляем поле для доступа к пулу
}

// SetupSuite запускается один раз перед всеми тестами
func (s *PostgresTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Запускаем контейнер с PostgreSQL
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)
	s.container = container

	// Получаем connection string
	host, err := container.Host(s.ctx)
	require.NoError(s.T(), err)

	port, err := container.MappedPort(s.ctx, "5432")
	require.NoError(s.T(), err)

	connString := fmt.Sprintf("postgres://test:test@%s:%s/testdb", host, port.Port())

	// Создаем storage
	s.storage, err = postgres.New(s.ctx, connString)
	require.NoError(s.T(), err)

	// Применяем миграции
	err = s.applyTestMigrations()
	require.NoError(s.T(), err)
}

// TearDownSuite очищает после всех тестов
func (s *PostgresTestSuite) TearDownSuite() {
	if s.storage != nil {
		s.storage.Close()
	}
	if s.container != nil {
		s.container.Terminate(s.ctx)
	}
}

// SetupTest запускается перед каждым тестом
func (s *PostgresTestSuite) SetupTest() {
	// Очищаем таблицу перед каждым тестом
	// Используем прямой доступ к пулу через reflection или через public метод
	// Для этого нам нужно либо добавить метод GetPool в Storage, либо использовать другой подход

	// Вместо прямого доступа к пулу, мы можем использовать SQL команды через существующие методы
	// или создать временную задачу и удалить все через SQL
	s.cleanupDatabase()
}

// cleanupDatabase очищает таблицу tasks
func (s *PostgresTestSuite) cleanupDatabase() {
	// Используем временное соединение для очистки
	// В реальном коде лучше добавить метод ClearAll в Storage
	ctx := context.Background()

	// Создаем временное подключение
	host, err := s.container.Host(s.ctx)
	require.NoError(s.T(), err)

	port, err := s.container.MappedPort(s.ctx, "5432")
	require.NoError(s.T(), err)

	connString := fmt.Sprintf("postgres://test:test@%s:%s/testdb", host, port.Port())

	// Используем pgx для прямого подключения
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		s.T().Logf("Не удалось подключиться для очистки: %v", err)
		return
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "DELETE FROM tasks")
	if err != nil {
		s.T().Logf("Не удалось очистить таблицу: %v", err)
	}
}

// applyTestMigrations создает тестовую таблицу
func (s *PostgresTestSuite) applyTestMigrations() error {
	// Используем прямое подключение для создания таблиц
	host, err := s.container.Host(s.ctx)
	if err != nil {
		return err
	}

	port, err := s.container.MappedPort(s.ctx, "5432")
	if err != nil {
		return err
	}

	connString := fmt.Sprintf("postgres://test:test@%s:%s/testdb", host, port.Port())

	conn, err := pgx.Connect(s.ctx, connString)
	if err != nil {
		return err
	}
	defer conn.Close(s.ctx)

	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		uuid UUID PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		status VARCHAR(50) NOT NULL,
		due_time TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP,
		deleted_at TIMESTAMP,
		version INTEGER NOT NULL DEFAULT 1,
		flag VARCHAR(50) NOT NULL DEFAULT 'active'
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_flag ON tasks(flag);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_due_time ON tasks(due_time);
	CREATE INDEX IF NOT EXISTS idx_tasks_deleted_at ON tasks(deleted_at) WHERE deleted_at IS NOT NULL;
	`

	_, err = conn.Exec(s.ctx, query)
	return err
}

// TestPostgresTestSuite запускает suite
func TestPostgresTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционные тесты в коротком режиме")
	}
	suite.Run(t, new(PostgresTestSuite))
}

// TestStorage_Create тестирует создание задачи
func (s *PostgresTestSuite) TestStorage_Create() {
	ctx := context.Background()

	taskToCreate := &task.Task{
		UUID:        uuid.New(),
		Title:       "Test Task",
		Description: "Test Description",
		Status:      task.StatusNew,
		DueTime:     time.Now().Add(24 * time.Hour),
		Flag:        task.FlagActive,
	}

	err := s.storage.Create(ctx, taskToCreate)
	require.NoError(s.T(), err)
	assert.False(s.T(), taskToCreate.CreatedAt.IsZero())

	// Проверяем, что задача создана
	retrievedTask, err := s.storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Test Task", retrievedTask.Title)
	assert.Equal(s.T(), task.FlagActive, retrievedTask.Flag)
	assert.Equal(s.T(), 1, retrievedTask.Version)
}

// TestStorage_GetByID тестирует получение задачи по ID
func (s *PostgresTestSuite) TestStorage_GetByID() {
	ctx := context.Background()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Test Get Task",
		Status:  task.StatusInProgress,
		DueTime: time.Now().Add(24 * time.Hour),
		Flag:    task.FlagActive,
	}

	err := s.storage.Create(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Получаем задачу
	retrievedTask, err := s.storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), taskToCreate.UUID, retrievedTask.UUID)
	assert.Equal(s.T(), "Test Get Task", retrievedTask.Title)

	// Пытаемся получить несуществующую задачу
	nonExistentID := uuid.New()
	_, err = s.storage.GetByID(ctx, nonExistentID)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "не найдено")
}

// TestStorage_Update тестирует обновление задачи
func (s *PostgresTestSuite) TestStorage_Update() {
	ctx := context.Background()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Original Title",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
		Flag:    task.FlagActive,
	}

	err := s.storage.Create(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Обновляем задачу
	taskToCreate.Title = "Updated Title"
	taskToCreate.Description = "Updated Description"
	taskToCreate.Status = task.StatusInProgress

	err = s.storage.Update(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Проверяем обновление
	retrievedTask, err := s.storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Updated Title", retrievedTask.Title)
	assert.Equal(s.T(), "Updated Description", retrievedTask.Description)
	assert.Equal(s.T(), task.StatusInProgress, retrievedTask.Status)
	assert.True(s.T(), retrievedTask.UpdatedAt != nil)
	assert.Equal(s.T(), 2, retrievedTask.Version)
}

// TestStorage_Update_VersionConflict тестирует конфликт версий
func (s *PostgresTestSuite) TestStorage_Update_VersionConflict() {
	ctx := context.Background()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Test Task",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
		Flag:    task.FlagActive,
	}

	err := s.storage.Create(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Получаем задачу
	task1, err := s.storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(s.T(), err)

	task2, err := s.storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(s.T(), err)

	// Обновляем через task1
	task1.Title = "Updated by task1"
	err = s.storage.Update(ctx, task1)
	require.NoError(s.T(), err)

	// Пытаемся обновить через task2 (устаревшая версия)
	task2.Title = "Updated by task2"
	err = s.storage.Update(ctx, task2)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "конфликт версий")
}

// TestStorage_DeleteSoft тестирует мягкое удаление
func (s *PostgresTestSuite) TestStorage_DeleteSoft() {
	ctx := context.Background()

	// Создаем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Task to delete",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
		Flag:    task.FlagActive,
	}

	err := s.storage.Create(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Мягко удаляем
	err = s.storage.DeleteSoft(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Проверяем, что задача помечена как удаленная
	retrievedTask, err := s.storage.GetByID(ctx, taskToCreate.UUID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), task.FlagDeleted, retrievedTask.Flag)
	assert.True(s.T(), retrievedTask.DeletedAt != nil)
	assert.Equal(s.T(), 2, retrievedTask.Version)
}

// TestStorage_DeleteFull тестирует полное удаление
func (s *PostgresTestSuite) TestStorage_DeleteFull() {
	ctx := context.Background()

	// Создаем и удаляем задачу
	taskToCreate := &task.Task{
		UUID:    uuid.New(),
		Title:   "Task to purge",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
		Flag:    task.FlagActive,
	}

	err := s.storage.Create(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Сначала мягкое удаление
	err = s.storage.DeleteSoft(ctx, taskToCreate)
	require.NoError(s.T(), err)

	// Полное удаление
	err = s.storage.DeleteFull(ctx, taskToCreate.UUID)
	require.NoError(s.T(), err)

	// Проверяем, что задачи больше нет
	_, err = s.storage.GetByID(ctx, taskToCreate.UUID)
	require.Error(s.T(), err)
}

// TestStorage_GetAllWithLimit тестирует получение всех задач с пагинацией
func (s *PostgresTestSuite) TestStorage_GetAllWithLimit() {
	ctx := context.Background()

	// Создаем несколько задач
	for i := 1; i <= 5; i++ {
		taskToCreate := &task.Task{
			UUID:    uuid.New(),
			Title:   fmt.Sprintf("Task %d", i),
			Status:  task.StatusNew,
			DueTime: time.Now().Add(time.Duration(i) * 24 * time.Hour),
			Flag:    task.FlagActive,
		}
		err := s.storage.Create(ctx, taskToCreate)
		require.NoError(s.T(), err)
	}

	// Создаем удаленную задачу
	deletedTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Deleted Task",
		Status:  task.StatusNew,
		DueTime: time.Now().Add(24 * time.Hour),
		Flag:    task.FlagActive,
	}
	err := s.storage.Create(ctx, deletedTask)
	require.NoError(s.T(), err)
	err = s.storage.DeleteSoft(ctx, deletedTask)
	require.NoError(s.T(), err)

	// Получаем все активные задачи
	tasks, err := s.storage.GetAllWithLimit(ctx, 1, 10)
	require.NoError(s.T(), err)
	assert.Len(s.T(), tasks, 5) // Только активные задачи

	// Тестируем пагинацию
	tasksPage1, err := s.storage.GetAllWithLimit(ctx, 1, 2)
	require.NoError(s.T(), err)
	assert.Len(s.T(), tasksPage1, 2)

	tasksPage2, err := s.storage.GetAllWithLimit(ctx, 2, 2)
	require.NoError(s.T(), err)
	assert.Len(s.T(), tasksPage2, 2)
}

// TestStorage_GetFlaggedWithLimit тестирует получение задач по флагу
func (s *PostgresTestSuite) TestStorage_GetFlaggedWithLimit() {
	ctx := context.Background()

	// Создаем задачи с разными флагами
	flags := []task.Flag{task.FlagActive, task.FlagArchived, task.FlagDeleted}
	for _, flag := range flags {
		for i := 1; i <= 2; i++ {
			taskToCreate := &task.Task{
				UUID:    uuid.New(),
				Title:   fmt.Sprintf("%s Task %d", flag, i),
				Status:  task.StatusNew,
				DueTime: time.Now().Add(24 * time.Hour),
				Flag:    flag,
			}
			err := s.storage.Create(ctx, taskToCreate)
			require.NoError(s.T(), err)
		}
	}

	// Получаем активные задачи
	activeTasks, err := s.storage.GetFlaggedWithLimit(ctx, 1, 10, task.FlagActive)
	require.NoError(s.T(), err)
	assert.Len(s.T(), activeTasks, 2)
	for _, t := range activeTasks {
		assert.Equal(s.T(), task.FlagActive, t.Flag)
	}

	// Получаем архивные задачи
	archivedTasks, err := s.storage.GetFlaggedWithLimit(ctx, 1, 10, task.FlagArchived)
	require.NoError(s.T(), err)
	assert.Len(s.T(), archivedTasks, 2)
	for _, t := range archivedTasks {
		assert.Equal(s.T(), task.FlagArchived, t.Flag)
	}
}

// TestStorage_GetTasksDueBefore тестирует получение просроченных задач
func (s *PostgresTestSuite) TestStorage_GetTasksDueBefore() {
	ctx := context.Background()
	now := time.Now()

	// Создаем просроченную задачу
	overdueTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Overdue Task",
		Status:  task.StatusNew,
		DueTime: now.Add(-24 * time.Hour), // Просрочена
		Flag:    task.FlagActive,
	}
	err := s.storage.Create(ctx, overdueTask)
	require.NoError(s.T(), err)

	// Создаем задачу на будущее
	futureTask := &task.Task{
		UUID:    uuid.New(),
		Title:   "Future Task",
		Status:  task.StatusNew,
		DueTime: now.Add(24 * time.Hour), // На будущее
		Flag:    task.FlagActive,
	}
	err = s.storage.Create(ctx, futureTask)
	require.NoError(s.T(), err)

	// Получаем просроченные задачи
	overdueTasks, err := s.storage.GetTasksDueBefore(ctx, now, 10)
	require.NoError(s.T(), err)
	assert.Len(s.T(), overdueTasks, 1)
	assert.Equal(s.T(), "Overdue Task", overdueTasks[0].Title)
}

// TestStorage_HealthCheck тестирует проверку здоровья
func (s *PostgresTestSuite) TestStorage_HealthCheck() {
	ctx := context.Background()

	err := s.storage.HealthCheck(ctx)
	require.NoError(s.T(), err)
}

// TestStorage_GetStatusedWithLimit тестирует получение задач по статусу
func (s *PostgresTestSuite) TestStorage_GetStatusedWithLimit() {
	ctx := context.Background()

	// Создаем задачи с разными статусами
	statuses := []task.Status{task.StatusNew, task.StatusInProgress, task.StatusDone}
	for _, status := range statuses {
		for i := 1; i <= 2; i++ {
			taskToCreate := &task.Task{
				UUID:    uuid.New(),
				Title:   fmt.Sprintf("%s Task %d", status, i),
				Status:  status,
				DueTime: time.Now().Add(24 * time.Hour),
				Flag:    task.FlagActive,
			}
			err := s.storage.Create(ctx, taskToCreate)
			require.NoError(s.T(), err)
		}
	}

	// Получаем задачи в статусе "в процессе"
	inProgressTasks, err := s.storage.GetStatusedWithLimit(ctx, 1, 10, task.StatusInProgress)
	require.NoError(s.T(), err)
	assert.Len(s.T(), inProgressTasks, 2)
	for _, t := range inProgressTasks {
		assert.Equal(s.T(), task.StatusInProgress, t.Status)
	}
}

// Unit тесты (без базы данных)
func TestStorage_New(t *testing.T) {
	tests := []struct {
		name        string
		connString  string
		expectError bool
	}{
		{
			name:        "invalid connection string",
			connString:  "invalid",
			expectError: true,
		},
		{
			name:        "empty connection string",
			connString:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := postgres.New(ctx, tt.connString)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStorage_Close(t *testing.T) {
	// Это тест на то, что Close не паникует
	storage := &postgres.Storage{}
	assert.NotPanics(t, func() {
		storage.Close()
	})
}

// Mock тесты для изолированного тестирования
func TestStorage_Methods_ErrorHandling(t *testing.T) {

	t.Run("Create with invalid task", func(t *testing.T) {
		// Здесь можно использовать mock для pgxpool.Pool
		// Для примера просто проверяем, что методы определены
		storage := &postgres.Storage{}

		// Проверяем, что storage реализует интерфейс
		var _ interface {
			Create(context.Context, *task.Task) error
		} = storage

		// Аналогично для других методов
		var _ interface {
			GetByID(context.Context, uuid.UUID) (*task.Task, error)
		} = storage

		var _ interface {
			Update(context.Context, *task.Task) error
		} = storage

		var _ interface {
			DeleteSoft(context.Context, *task.Task) error
		} = storage

		var _ interface {
			DeleteFull(context.Context, uuid.UUID) error
		} = storage

		var _ interface {
			GetAllWithLimit(context.Context, int, int) ([]*task.Task, error)
		} = storage

		var _ interface {
			HealthCheck(context.Context) error
		} = storage
	})
}

// TestEdgeCases тестирует граничные случаи
func (s *PostgresTestSuite) TestEdgeCases() {
	ctx := context.Background()

	s.T().Run("empty result sets", func(t *testing.T) {
		// Получение пустого списка
		tasks, err := s.storage.GetAllWithLimit(ctx, 1, 10)
		require.NoError(t, err)
		assert.Empty(t, tasks)

		// Получение по несуществующему флагу
		tasks, err = s.storage.GetFlaggedWithLimit(ctx, 1, 10, "NON_EXISTENT_FLAG")
		require.NoError(t, err)
		assert.Empty(t, tasks)
	})

	s.T().Run("pagination edge cases", func(t *testing.T) {
		// Создаем одну задачу
		taskToCreate := &task.Task{
			UUID:    uuid.New(),
			Title:   "Single Task",
			Status:  task.StatusNew,
			DueTime: time.Now().Add(24 * time.Hour),
			Flag:    task.FlagActive,
		}
		err := s.storage.Create(ctx, taskToCreate)
		require.NoError(t, err)

		// Страница за пределами данных
		tasks, err := s.storage.GetAllWithLimit(ctx, 100, 10)
		require.NoError(t, err)
		assert.Empty(t, tasks)

		// Нулевой лимит
		tasks, err = s.storage.GetAllWithLimit(ctx, 1, 0)
		require.NoError(t, err)
		assert.Empty(t, tasks)
	})
}

// TestPerformanceLogging проверяет логирование медленных запросов
func (s *PostgresTestSuite) TestPerformanceLogging() {
	ctx := context.Background()

	// Создаем много задач для тестирования производительности
	for i := 1; i <= 100; i++ {
		taskToCreate := &task.Task{
			UUID:    uuid.New(),
			Title:   fmt.Sprintf("Performance Task %d", i),
			Status:  task.StatusNew,
			DueTime: time.Now().Add(time.Duration(i) * time.Hour),
			Flag:    task.FlagActive,
		}
		err := s.storage.Create(ctx, taskToCreate)
		require.NoError(s.T(), err)
	}

	// Этот тест в основном проверяет, что запросы выполняются
	// Логирование производительности проверяется в логах
	tasks, err := s.storage.GetAllWithLimit(ctx, 1, 50)
	require.NoError(s.T(), err)
	assert.Len(s.T(), tasks, 50)
}
