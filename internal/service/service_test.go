package service_test

import (
	"context"
	"errors"
	"taskTracker/internal/models/task"
	"taskTracker/internal/service"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTaskRepository - мок репозитория
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskRepository) GetAllWithLimit(ctx context.Context, page, limit int) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskRepository) GetTasksDueBefore(ctx context.Context, dueTime time.Time, limit int) ([]*task.Task, error) {
	args := m.Called(ctx, dueTime, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskRepository) GetFlaggedWithLimit(ctx context.Context, page, limit int, flag task.Flag) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit, flag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskRepository) Create(ctx context.Context, t *task.Task) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTaskRepository) Update(ctx context.Context, t *task.Task) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTaskRepository) DeleteSoft(ctx context.Context, t *task.Task) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTaskRepository) DeleteFull(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskRepository) GetStatusedWithLimit(ctx context.Context, page, limit int, flag task.Status) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit, flag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

var _ service.TaskRepository = (*MockTaskRepository)(nil)

// TestTaskService_HealthCheck тестирует HealthCheck
func TestTaskService_HealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockTaskRepository)
		expectError bool
	}{
		{
			name: "success - health check passes",
			setupMock: func(m *MockTaskRepository) {
				m.On("HealthCheck", mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "error - health check fails",
			setupMock: func(m *MockTaskRepository) {
				m.On("HealthCheck", mock.Anything).Return(errors.New("db connection failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			tt.setupMock(mockRepo)

			svc := service.NewTaskService(mockRepo, service.DBType)
			err := svc.HealthCheck(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "проверка здоровья сервиса")
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestTaskService_ArchiveTask тестирует архивацию задачи
func TestTaskService_ArchiveTask(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	now := time.Now()

	tests := []struct {
		name        string
		setupMock   func(*MockTaskRepository)
		expectError bool
		errorType   string
	}{
		{
			name: "success - archive active task",
			setupMock: func(m *MockTaskRepository) {
				task1 := &task.Task{
					UUID:  taskID,
					Flag:  task.FlagActive,
					Title: "Test Task",
				}
				m.On("GetByID", mock.Anything, taskID).Return(task1, nil)
				m.On("Update", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
					return t.Flag == task.FlagArchived && t.UpdatedAt != nil
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name: "error - task not found",
			setupMock: func(m *MockTaskRepository) {
				m.On("GetByID", mock.Anything, taskID).Return(nil, errors.New("not found"))
			},
			expectError: true,
			errorType:   "NotFound",
		},
		{
			name: "error - already archived",
			setupMock: func(m *MockTaskRepository) {
				task := &task.Task{
					UUID:      taskID,
					Flag:      task.FlagArchived,
					UpdatedAt: &now,
				}
				m.On("GetByID", mock.Anything, taskID).Return(task, nil)
			},
			expectError: true,
			errorType:   "BusinessError",
		},
		{
			name: "error - task deleted",
			setupMock: func(m *MockTaskRepository) {
				task := &task.Task{
					UUID:      taskID,
					Flag:      task.FlagDeleted,
					DeletedAt: &now,
				}
				m.On("GetByID", mock.Anything, taskID).Return(task, nil)
			},
			expectError: true,
			errorType:   "BusinessError",
		},
		{
			name: "error - version conflict",
			setupMock: func(m *MockTaskRepository) {
				task := &task.Task{
					UUID: taskID,
					Flag: task.FlagActive,
				}
				m.On("GetByID", mock.Anything, taskID).Return(task, nil)
				m.On("Update", mock.Anything, mock.Anything).Return(errors.New("version conflict"))
			},
			expectError: true,
			errorType:   "BusinessError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			tt.setupMock(mockRepo)

			svc := service.NewTaskService(mockRepo, service.DBType)
			result, err := svc.ArchiveTask(ctx, taskID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType == "BusinessError" {
					_, ok := err.(*service.BusinessError)
					assert.True(t, ok, "Expected BusinessError")
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, task.FlagArchived, result.Flag)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestTaskService_UnarchiveTask тестирует разархивацию
func TestTaskService_UnarchiveTask(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	t.Run("success - unarchive archived task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		task1 := &task.Task{
			UUID: taskID,
			Flag: task.FlagArchived,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(task1, nil)
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
			return t.Flag == task.FlagActive
		})).Return(nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		result, err := svc.UnarchiveTask(ctx, taskID)

		assert.NoError(t, err)
		assert.Equal(t, task.FlagActive, result.Flag)
		mockRepo.AssertExpectations(t)
	})
}

// TestTaskService_CreateTask тестирует создание задачи
func TestTaskService_CreateTask(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		dueTime        time.Time
		expectedStatus task.Status
	}{
		{
			name:           "new task - due in future",
			dueTime:        time.Now().Add(48 * time.Hour),
			expectedStatus: task.StatusNew,
		},
		{
			name:           "in progress - due soon",
			dueTime:        time.Now().Add(12 * time.Hour),
			expectedStatus: task.StatusInProgress,
		},
		{
			name:           "overdue - due in past",
			dueTime:        time.Now().Add(-1 * time.Hour),
			expectedStatus: task.StatusOverdue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
				return t.Title == "Test" && t.Description == "Description" && t.Status == tt.expectedStatus
			})).Return(nil)

			svc := service.NewTaskService(mockRepo, service.DBType)
			result, err := svc.CreateTask(ctx, "Test", "Description", tt.dueTime)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Equal(t, task.FlagActive, result.Flag)
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestTaskService_UpdateTask тестирует обновление задачи
func TestTaskService_UpdateTask(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	t.Run("success - update active task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID:        taskID,
			Title:       "Old Title",
			Description: "Old Desc",
			Status:      task.StatusNew,
			DueTime:     time.Now().Add(24 * time.Hour),
			Flag:        task.FlagActive,
			Version:     1,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
			return t.Title == "New Title" && t.Description == "New Desc"
		})).Return(nil)

		svc := service.NewTaskService(mockRepo, service.DBType)

		updateOpts := []task.TaskOption{
			func(t *task.Task) { t.Title = "New Title" },
			func(t *task.Task) { t.Description = "New Desc" },
		}

		result, err := svc.UpdateTask(ctx, taskID, updateOpts...)

		assert.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)
		assert.Equal(t, "New Desc", result.Description)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error - update archived task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID: taskID,
			Flag: task.FlagArchived,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		_, err := svc.UpdateTask(ctx, taskID)

		assert.Error(t, err)
		_, ok := err.(*service.BusinessError)
		assert.True(t, ok)
		mockRepo.AssertExpectations(t)
	})
}

// TestTaskService_GetTaskByID тестирует получение задачи
func TestTaskService_GetTaskByID(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	t.Run("success - get active task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID:    taskID,
			Flag:    task.FlagActive,
			Status:  task.StatusNew,
			DueTime: time.Now().Add(1 * time.Hour),
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		result, err := svc.GetTaskByID(ctx, taskID)

		assert.NoError(t, err)
		assert.Equal(t, taskID, result.UUID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error - get deleted task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID:      taskID,
			Flag:      task.FlagDeleted,
			DeletedAt: &time.Time{},
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		_, err := svc.GetTaskByID(ctx, taskID)

		assert.Error(t, err)
		_, ok := err.(*service.BusinessError)
		assert.True(t, ok)
		mockRepo.AssertExpectations(t)
	})

	t.Run("auto update to overdue", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID:    taskID,
			Flag:    task.FlagActive,
			Status:  task.StatusNew,
			DueTime: time.Now().Add(-1 * time.Hour), // Просрочена
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		result, err := svc.GetTaskByID(ctx, taskID)

		assert.NoError(t, err)
		assert.Equal(t, task.StatusOverdue, result.Status)
		mockRepo.AssertExpectations(t)
	})
}

// TestTaskService_DeleteTask тестирует удаление задачи
func TestTaskService_DeleteTask(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	tests := []struct {
		name        string
		taskStatus  task.Status
		taskFlag    task.Flag
		expectError bool
	}{
		{
			name:        "success - delete new task",
			taskStatus:  task.StatusNew,
			taskFlag:    task.FlagActive,
			expectError: false,
		},
		{
			name:        "error - delete in progress task",
			taskStatus:  task.StatusInProgress,
			taskFlag:    task.FlagActive,
			expectError: true,
		},
		{
			name:        "error - already deleted",
			taskStatus:  task.StatusDone,
			taskFlag:    task.FlagDeleted,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			existingTask := &task.Task{
				UUID:   taskID,
				Status: tt.taskStatus,
				Flag:   tt.taskFlag,
			}

			if tt.taskFlag == task.FlagActive && tt.taskStatus != task.StatusInProgress {
				mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)
				mockRepo.On("DeleteSoft", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
					return t.Flag == task.FlagDeleted && t.DeletedAt != nil
				})).Return(nil)
			} else {
				mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)
			}

			svc := service.NewTaskService(mockRepo, service.DBType)
			err := svc.DeleteTask(ctx, taskID)

			if tt.expectError {
				assert.Error(t, err)
				_, ok := err.(*service.BusinessError)
				assert.True(t, ok)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// TestTaskService_RestoreTask тестирует восстановление задачи
func TestTaskService_RestoreTask(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	t.Run("success - restore recently deleted task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		deletedTime := time.Now().Add(-24 * time.Hour) // Удалена 1 день назад
		existingTask := &task.Task{
			UUID:      taskID,
			Flag:      task.FlagDeleted,
			DeletedAt: &deletedTime,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
			return t.Flag == task.FlagActive && t.DeletedAt == nil
		})).Return(nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		result, err := svc.RestoreTask(ctx, taskID)

		assert.NoError(t, err)
		assert.Equal(t, task.FlagActive, result.Flag)
		assert.Nil(t, result.DeletedAt)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error - restore expired task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		deletedTime := time.Now().Add(-31 * 24 * time.Hour) // Удалена 31 день назад
		existingTask := &task.Task{
			UUID:      taskID,
			Flag:      task.FlagDeleted,
			DeletedAt: &deletedTime,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		_, err := svc.RestoreTask(ctx, taskID)

		assert.Error(t, err)
		_, ok := err.(*service.BusinessError)
		assert.True(t, ok)
		mockRepo.AssertExpectations(t)
	})
}

// TestTaskService_PurgeTask тестирует полное удаление
func TestTaskService_PurgeTask(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	t.Run("success - purge deleted task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID: taskID,
			Flag: task.FlagDeleted,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)
		mockRepo.On("DeleteFull", mock.Anything, taskID).Return(nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		err := svc.PurgeTask(ctx, taskID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error - purge active task", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID: taskID,
			Flag: task.FlagActive,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		err := svc.PurgeTask(ctx, taskID)

		assert.Error(t, err)
		_, ok := err.(*service.BusinessError)
		assert.True(t, ok)
		mockRepo.AssertExpectations(t)
	})
}

// TestTaskService_GetAllTasks тестирует получение всех задач
func TestTaskService_GetAllTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("success - get all tasks with pagination", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		tasks := []*task.Task{
			{UUID: uuid.New(), Title: "Task 1"},
			{UUID: uuid.New(), Title: "Task 2"},
		}

		mockRepo.On("GetAllWithLimit", mock.Anything, 1, 10).Return(tasks, nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		result, err := svc.GetAllTasks(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		mockRepo.AssertExpectations(t)
	})
}

// TestTaskService_GetFilteredTasks тестирует получение задач по фильтрам
func TestTaskService_GetFilteredTasks(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		method func(*service.TaskService) ([]*task.Task, error)
		flag   task.Flag
	}{
		{
			name: "get active tasks",
			method: func(s *service.TaskService) ([]*task.Task, error) {
				return s.GetActiveTasks(ctx, 1, 10)
			},
			flag: task.FlagActive,
		},
		{
			name: "get archived tasks",
			method: func(s *service.TaskService) ([]*task.Task, error) {
				return s.GetArchivedTasks(ctx, 1, 10)
			},
			flag: task.FlagArchived,
		},
		{
			name: "get deleted tasks",
			method: func(s *service.TaskService) ([]*task.Task, error) {
				return s.GetDeletedTasks(ctx, 1, 10)
			},
			flag: task.FlagDeleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			tasks := []*task.Task{
				{UUID: uuid.New(), Title: "Task 1", Flag: tt.flag},
				{UUID: uuid.New(), Title: "Task 2", Flag: tt.flag},
			}

			mockRepo.On("GetFlaggedWithLimit", mock.Anything, 1, 10, tt.flag).Return(tasks, nil)

			if tt.flag == task.FlagActive {
				// Для активных задач может быть дополнительная логика
				mockRepo.On("GetFlaggedWithLimit", mock.Anything, 1, 10, task.FlagActive).Return(tasks, nil)
			}

			svc := service.NewTaskService(mockRepo, service.DBType)
			result, err := tt.method(&svc)

			assert.NoError(t, err)
			assert.Len(t, result, 2)
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestTaskService_GetOverdueTasks тестирует получение просроченных задач
func TestTaskService_GetOverdueTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("success - get overdue tasks", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		now := time.Now()
		tasks := []*task.Task{
			{UUID: uuid.New(), Status: task.StatusNew, DueTime: now.Add(-1 * time.Hour)},
			{UUID: uuid.New(), Status: task.StatusInProgress, DueTime: now.Add(-2 * time.Hour)},
		}

		mockRepo.On("GetTasksDueBefore", mock.Anything, mock.Anything, 10).
			Return(tasks, nil)
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
			return t.Status == task.StatusOverdue
		})).Return(nil).Twice()

		svc := service.NewTaskService(mockRepo, service.DBType)
		result, err := svc.GetOverdueTasks(ctx, 1, 10)

		assert.NoError(t, err)
		assert.Len(t, result, 2)

		// Проверяем, что статус обновлен
		for _, task1 := range result {
			assert.Equal(t, task.StatusOverdue, task1.Status)
		}

		mockRepo.AssertExpectations(t)
	})
}

// TestTaskService_RepoType проверяет работу с разными типами репозиториев
func TestTaskService_RepoType(t *testing.T) {
	t.Run("DB repository type", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		svc := service.NewTaskService(mockRepo, service.DBType)
		assert.Equal(t, service.DBType, svc.RepoType)
	})

	t.Run("InMemory repository type", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		svc := service.NewTaskService(mockRepo, service.InMemoryType)
		assert.Equal(t, service.InMemoryType, svc.RepoType)
	})
}

// TestTaskService_EdgeCases тестирует граничные случаи
func TestTaskService_EdgeCases(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	t.Run("update task to done archives old tasks", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		oldCreatedAt := time.Now().Add(-31 * 24 * time.Hour) // Создана 31 день назад
		existingTask := &task.Task{
			UUID:      taskID,
			Flag:      task.FlagActive,
			Status:    task.StatusNew,
			CreatedAt: oldCreatedAt,
			Version:   1,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
			return t.Status == task.StatusDone && t.Flag == task.FlagArchived
		})).Return(nil)

		svc := service.NewTaskService(mockRepo, service.DBType)

		updateOpts := []task.TaskOption{
			func(t *task.Task) { t.Status = task.StatusDone },
		}

		result, err := svc.UpdateTask(ctx, taskID, updateOpts...)

		assert.NoError(t, err)
		assert.Equal(t, task.StatusDone, result.Status)
		assert.Equal(t, task.FlagArchived, result.Flag)
		mockRepo.AssertExpectations(t)
	})

	t.Run("update overdue status automatically", func(t *testing.T) {
		mockRepo := new(MockTaskRepository)
		existingTask := &task.Task{
			UUID:    taskID,
			Flag:    task.FlagActive,
			Status:  task.StatusNew,
			DueTime: time.Now().Add(-1 * time.Hour), // Просрочена
			Version: 1,
		}

		mockRepo.On("GetByID", mock.Anything, taskID).Return(existingTask, nil)
		mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *task.Task) bool {
			return t.Status == task.StatusOverdue
		})).Return(nil)

		svc := service.NewTaskService(mockRepo, service.DBType)
		result, err := svc.UpdateTask(ctx, taskID)

		assert.NoError(t, err)
		assert.Equal(t, task.StatusOverdue, result.Status)
		mockRepo.AssertExpectations(t)
	})
}
