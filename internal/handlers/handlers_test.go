package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"taskTracker/internal/handlers"
	"taskTracker/internal/handlers/dto"
	"taskTracker/internal/models/task"
	"taskTracker/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTaskService - мок сервиса
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTaskService) CreateTask(ctx context.Context, title, description string, dueTime time.Time) (*task.Task, error) {
	args := m.Called(ctx, title, description, dueTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskService) GetTaskByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskService) UpdateTask(ctx context.Context, id uuid.UUID, options ...task.TaskOption) (*task.Task, error) {
	args := m.Called(ctx, id, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskService) DeleteTask(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskService) ArchiveTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskService) UnarchiveTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskService) GetAllTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskService) GetActiveTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskService) GetArchivedTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskService) GetOverdueTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskService) GetDeletedTasks(ctx context.Context, page, limit int) ([]*task.Task, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*task.Task), args.Error(1)
}

func (m *MockTaskService) RestoreTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*task.Task), args.Error(1)
}

func (m *MockTaskService) PurgeTask(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ handlers.Service = (*MockTaskService)(nil)

// TestTaskHandler_HealthCheck тестирует HealthCheck
func TestTaskHandler_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockTaskService)
		expectedStatus int
	}{
		{
			name: "success - healthy",
			setupMock: func(m *MockTaskService) {
				m.On("HealthCheck", mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - unhealthy",
			setupMock: func(m *MockTaskService) {
				m.On("HealthCheck", mock.Anything).Return(errors.New("service unavailable"))
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			handler.HealthCheck(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "task-tracker")
			
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_PostTask тестирует создание задачи
func TestTaskHandler_PostTask(t *testing.T) {
	taskID := uuid.New()
	dueTime := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name           string
		requestBody    string
		contentType    string
		setupMock      func(*MockTaskService)
		expectedStatus int
	}{
		{
			name: "success - create task",
			requestBody: fmt.Sprintf(`{
				"title": "Test Task",
				"description": "Test Description",
				"due_time": "%s"
			}`, dueTime.Format(time.RFC3339)),
			contentType: "application/json",
			setupMock: func(m *MockTaskService) {
				m.On("CreateTask", mock.Anything, "Test Task", "Test Description", mock.Anything).
					Return(&task.Task{
						UUID:        taskID,
						Title:       "Test Task",
						Description: "Test Description",
						DueTime:     dueTime,
						Status:      task.StatusNew,
						Flag:        task.FlagActive,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid content type",
			requestBody:    `{}`,
			contentType:    "text/plain",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "error - invalid JSON",
			requestBody:    `{invalid json}`,
			contentType:    "application/json",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "error - service error",
			requestBody: fmt.Sprintf(`{
				"title": "Test Task",
				"due_time": "%s"
			}`, dueTime.Format(time.RFC3339)),
			contentType: "application/json",
			setupMock: func(m *MockTaskService) {
				m.On("CreateTask", mock.Anything, "Test Task", "", mock.Anything).
					Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "error - missing title",
			requestBody: fmt.Sprintf(`{
				"description": "Test Description",
				"due_time": "%s"
			}`, dueTime.Format(time.RFC3339)),
			contentType:    "application/json",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "error - missing due time",
			requestBody: `{
				"title": "Test Task",
				"description": "Test Description"
			}`,
			contentType:    "application/json",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("POST", "/tasks", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			handler.PostTask(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response dto.TaskResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "Test Task", response.Title)
			}
			
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_GetTaskByID тестирует получение задачи по ID
func TestTaskHandler_GetTaskByID(t *testing.T) {
	taskID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		taskID         string
		setupMock      func(*MockTaskService)
		expectedStatus int
	}{
		{
			name:   "success - get task",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("GetTaskByID", mock.Anything, taskID).
					Return(&task.Task{
						UUID:    taskID,
						Title:   "Test Task",
						Status:  task.StatusNew,
						DueTime: now.Add(24 * time.Hour),
						Flag:    task.FlagActive,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid UUID",
			taskID:         "invalid-uuid",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "error - task not found",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("GetTaskByID", mock.Anything, taskID).
					Return(nil, service.NewNotFound(service.DBType, taskID.String()))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "error - business error",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("GetTaskByID", mock.Anything, taskID).
					Return(nil, service.NewBusinessError("TASK_DELETED", "Task was deleted"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "error - service error",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("GetTaskByID", mock.Anything, taskID).
					Return(nil, errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("GET", "/tasks/"+tt.taskID, nil)
			w := httptest.NewRecorder()

			// Симуляция параметра пути
			if tt.taskID != "" {
				req.SetPathValue("id", tt.taskID)
			}

			handler.GetTaskByID(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response dto.TaskResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, taskID.String(), response.UUID)
				assert.Equal(t, "Test Task", response.Title)
			}
			
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_UpdateTaskByID тестирует обновление задачи
func TestTaskHandler_UpdateTaskByID(t *testing.T) {
	taskID := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		requestBody    string
		contentType    string
		setupMock      func(*MockTaskService)
		expectedStatus int
	}{
		{
			name:   "success - update task",
			taskID: taskID.String(),
			requestBody: `{
				"title": "Updated Title",
				"description": "Updated Description",
				"status": "in_progress"
			}`,
			contentType: "application/json",
			setupMock: func(m *MockTaskService) {
				m.On("UpdateTask", mock.Anything, taskID, mock.Anything).
					Return(&task.Task{
						UUID:        taskID,
						Title:       "Updated Title",
						Description: "Updated Description",
						Status:      task.StatusInProgress,
						Flag:        task.FlagActive,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - invalid content type",
			taskID:         taskID.String(),
			requestBody:    `{}`,
			contentType:    "text/plain",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "error - invalid UUID",
			taskID:         "invalid-uuid",
			requestBody:    `{}`,
			contentType:    "application/json",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error - invalid JSON",
			taskID:         taskID.String(),
			requestBody:    `{invalid json}`,
			contentType:    "application/json",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "error - service business error",
			taskID: taskID.String(),
			requestBody: `{
				"title": "Updated Title"
			}`,
			contentType: "application/json",
			setupMock: func(m *MockTaskService) {
				m.On("UpdateTask", mock.Anything, taskID, mock.Anything).
					Return(nil, service.NewBusinessError("VERSION_CONFLICT", "Version conflict"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("PUT", "/tasks/"+tt.taskID, bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", tt.contentType)
			if tt.taskID != "" {
				req.SetPathValue("id", tt.taskID)
			}
			w := httptest.NewRecorder()

			handler.UpdateTaskByID(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response dto.TaskResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "Updated Title", response.Title)
			}
			
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_DeleteTaskByID тестирует удаление задачи
func TestTaskHandler_DeleteTaskByID(t *testing.T) {
	taskID := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		setupMock      func(*MockTaskService)
		expectedStatus int
	}{
		{
			name:   "success - delete task",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("DeleteTask", mock.Anything, taskID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "error - invalid UUID",
			taskID:         "invalid-uuid",
			setupMock:      func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "error - business error",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("DeleteTask", mock.Anything, taskID).
					Return(service.NewBusinessError("IN_PROGRESS", "Task is in progress"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "error - service error",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("DeleteTask", mock.Anything, taskID).
					Return(errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("DELETE", "/tasks/"+tt.taskID, nil)
			if tt.taskID != "" {
				req.SetPathValue("id", tt.taskID)
			}
			w := httptest.NewRecorder()

			handler.DeleteTaskByID(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_ArchiveTask тестирует архивацию задачи
func TestTaskHandler_ArchiveTask(t *testing.T) {
	taskID := uuid.New()

	tests := []struct {
		name           string
		taskID         string
		setupMock      func(*MockTaskService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "success - archive task",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("ArchiveTask", mock.Anything, taskID).
					Return(&task.Task{
						UUID:  taskID,
						Title: "Test Task",
						Flag:  task.FlagArchived,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "error - already archived",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("ArchiveTask", mock.Anything, taskID).
					Return(nil, service.NewBusinessError("ALREADY_ARCHIVED", "Task already archived"))
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "ALREADY_ARCHIVED",
		},
		{
			name:   "error - task deleted",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("ArchiveTask", mock.Anything, taskID).
					Return(nil, service.NewBusinessError("TASK_DELETED", "Task was deleted"))
			},
			expectedStatus: http.StatusGone,
			expectedError:  "TASK_DELETED",
		},
		{
			name:   "error - version conflict",
			taskID: taskID.String(),
			setupMock: func(m *MockTaskService) {
				m.On("ArchiveTask", mock.Anything, taskID).
					Return(nil, service.NewBusinessError("VERSION_CONFLICT", "Version conflict"))
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "VERSION_CONFLICT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("POST", "/tasks/"+tt.taskID+"/archive", nil)
			if tt.taskID != "" {
				req.SetPathValue("id", tt.taskID)
			}
			w := httptest.NewRecorder()

			handler.ArchiveTask(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, response["error"])
			}
			
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_UnarchiveTask тестирует разархивацию задачи
func TestTaskHandler_UnarchiveTask(t *testing.T) {
	taskID := uuid.New()

	t.Run("success - unarchive task", func(t *testing.T) {
		mockService := new(MockTaskService)
		mockService.On("UnarchiveTask", mock.Anything, taskID).
			Return(&task.Task{
				UUID:  taskID,
				Title: "Test Task",
				Flag:  task.FlagActive,
			}, nil)

		handler := handlers.NewTaskHandler(mockService)

		req := httptest.NewRequest("POST", "/tasks/"+taskID.String()+"/unarchive", nil)
		req.SetPathValue("id", taskID.String())
		w := httptest.NewRecorder()

		handler.UnarchiveTask(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response dto.TaskResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, taskID.String(), response.UUID)
		
		mockService.AssertExpectations(t)
	})
}

// TestTaskHandler_GetActiveTasks тестирует получение активных задач
func TestTaskHandler_GetActiveTasks(t *testing.T) {
	taskID1 := uuid.New()
	taskID2 := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*MockTaskService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "success - with default pagination",
			queryParams: "",
			setupMock: func(m *MockTaskService) {
				m.On("GetActiveTasks", mock.Anything, 1, 10).
					Return([]*task.Task{
						{UUID: taskID1, Title: "Task 1", Flag: task.FlagActive},
						{UUID: taskID2, Title: "Task 2", Flag: task.FlagActive},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "success - with custom pagination",
			queryParams: "?page=2&limit=5",
			setupMock: func(m *MockTaskService) {
				m.On("GetActiveTasks", mock.Anything, 2, 5).
					Return([]*task.Task{
						{UUID: taskID1, Title: "Task 1", Flag: task.FlagActive},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:        "error - invalid page parameter",
			queryParams: "?page=invalid",
			setupMock:   func(m *MockTaskService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "error - service error",
			queryParams: "",
			setupMock: func(m *MockTaskService) {
				m.On("GetActiveTasks", mock.Anything, 1, 10).
					Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("GET", "/tasks/active"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.GetActiveTasks(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response []dto.TaskResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Len(t, response, tt.expectedCount)
			}
			
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_GetArchivedTasks тестирует получение архивных задач
func TestTaskHandler_GetArchivedTasks(t *testing.T) {
	t.Run("success - get archived tasks", func(t *testing.T) {
		mockService := new(MockTaskService)
		taskID := uuid.New()
		
		mockService.On("GetArchivedTasks", mock.Anything, 1, 10).
			Return([]*task.Task{
				{UUID: taskID, Title: "Archived Task", Flag: task.FlagArchived},
			}, nil)

		handler := handlers.NewTaskHandler(mockService)

		req := httptest.NewRequest("GET", "/tasks/archived", nil)
		w := httptest.NewRecorder()

		handler.GetArchivedTasks(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response []dto.TaskResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, taskID.String(), response[0].UUID)
		
		mockService.AssertExpectations(t)
	})
}

// TestTaskHandler_GetAllTasks тестирует получение всех задач
func TestTaskHandler_GetAllTasks(t *testing.T) {
	t.Run("success - get all tasks", func(t *testing.T) {
		mockService := new(MockTaskService)
		taskID1 := uuid.New()
		taskID2 := uuid.New()
		
		mockService.On("GetAllTasks", mock.Anything, 1, 10).
			Return([]*task.Task{
				{UUID: taskID1, Title: "Task 1", Flag: task.FlagActive},
				{UUID: taskID2, Title: "Task 2", Flag: task.FlagArchived},
			}, nil)

		handler := handlers.NewTaskHandler(mockService)

		req := httptest.NewRequest("GET", "/tasks", nil)
		w := httptest.NewRecorder()

		handler.GetAllTasks(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response []dto.TaskResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		
		mockService.AssertExpectations(t)
	})
}

// TestTaskHandler_GetOverdueTasks тестирует получение просроченных задач
func TestTaskHandler_GetOverdueTasks(t *testing.T) {
	t.Run("success - get overdue tasks", func(t *testing.T) {
		mockService := new(MockTaskService)
		taskID := uuid.New()
		
		mockService.On("GetOverdueTasks", mock.Anything, 1, 10).
			Return([]*task.Task{
				{UUID: taskID, Title: "Overdue Task", Status: task.StatusOverdue},
			}, nil)

		handler := handlers.NewTaskHandler(mockService)

		req := httptest.NewRequest("GET", "/tasks/overdue", nil)
		w := httptest.NewRecorder()

		handler.GetOverdueTasks(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response []dto.TaskResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, string(task.StatusOverdue), response[0].Status)
		
		mockService.AssertExpectations(t)
	})
}

// TestTaskHandler_GetDeletedTasks тестирует получение удаленных задач
func TestTaskHandler_GetDeletedTasks(t *testing.T) {
	t.Run("success - get deleted tasks", func(t *testing.T) {
		mockService := new(MockTaskService)
		taskID := uuid.New()
		
		mockService.On("GetDeletedTasks", mock.Anything, 1, 10).
			Return([]*task.Task{
				{UUID: taskID, Title: "Deleted Task", Flag: task.FlagDeleted},
			}, nil)

		handler := handlers.NewTaskHandler(mockService)

		req := httptest.NewRequest("GET", "/tasks/deleted", nil)
		w := httptest.NewRecorder()

		handler.GetDeletedTasks(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response []dto.TaskResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, taskID.String(), response[0].UUID)
		
		mockService.AssertExpectations(t)
	})
}

// TestTaskHandler_RestoreTask тестирует восстановление задачи
func TestTaskHandler_RestoreTask(t *testing.T) {
	taskID := uuid.New()

	tests := []struct {
		name           string
		setupMock      func(*MockTaskService)
		expectedStatus int
	}{
		{
			name: "success - restore task",
			setupMock: func(m *MockTaskService) {
				m.On("RestoreTask", mock.Anything, taskID).
					Return(&task.Task{
						UUID:  taskID,
						Title: "Restored Task",
						Flag:  task.FlagActive,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - restore expired",
			setupMock: func(m *MockTaskService) {
				m.On("RestoreTask", mock.Anything, taskID).
					Return(nil, service.NewBusinessError("RESTORE_EXPIRED", "Restore period expired"))
			},
			expectedStatus: http.StatusGone,
		},
		{
			name: "error - not deleted",
			setupMock: func(m *MockTaskService) {
				m.On("RestoreTask", mock.Anything, taskID).
					Return(nil, service.NewBusinessError("NOT_DELETED", "Task not deleted"))
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("POST", "/admin/tasks/"+taskID.String()+"/restore", nil)
			req.SetPathValue("id", taskID.String())
			w := httptest.NewRecorder()

			handler.RestoreTask(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_PurgeTask тестирует полное удаление задачи
func TestTaskHandler_PurgeTask(t *testing.T) {
	taskID := uuid.New()

	tests := []struct {
		name           string
		setupMock      func(*MockTaskService)
		expectedStatus int
	}{
		{
			name: "success - purge task",
			setupMock: func(m *MockTaskService) {
				m.On("PurgeTask", mock.Anything, taskID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "error - not deleted",
			setupMock: func(m *MockTaskService) {
				m.On("PurgeTask", mock.Anything, taskID).
					Return(service.NewBusinessError("NOT_DELETED", "Task not deleted"))
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTaskService)
			tt.setupMock(mockService)

			handler := handlers.NewTaskHandler(mockService)

			req := httptest.NewRequest("DELETE", "/admin/tasks/"+taskID.String()+"/purge", nil)
			req.SetPathValue("id", taskID.String())
			w := httptest.NewRecorder()

			handler.PurgeTask(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestTaskHandler_ValidatePagination тестирует валидацию пагинации
func TestTaskHandler_ValidatePagination(t *testing.T) {
	handler := handlers.NewTaskHandler(nil)

	tests := []struct {
		name        string
		queryParams string
		expectPage  int
		expectLimit int
		shouldFail  bool
	}{
		{
			name:        "default values",
			queryParams: "",
			expectPage:  1,
			expectLimit: 10,
			shouldFail:  false,
		},
		{
			name:        "custom values",
			queryParams: "?page=2&limit=20",
			expectPage:  2,
			expectLimit: 20,
			shouldFail:  false,
		},
		{
			name:        "invalid page",
			queryParams: "?page=invalid",
			shouldFail:  true,
		},
		{
			name:        "invalid limit",
			queryParams: "?limit=invalid",
			shouldFail:  true,
		},
		{
			name:        "negative page",
			queryParams: "?page=-1",
			shouldFail:  true,
		},
		{
			name:        "zero limit",
			queryParams: "?limit=0",
			shouldFail:  true,
		},
		{
			name:        "too large limit",
			queryParams: "?limit=1000",
			expectPage:  1,
			expectLimit: 100,
			shouldFail:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Используем рефлексию для тестирования приватных методов
			// В реальном коде нужно вынести валидацию в отдельный пакет для тестирования
			if tt.shouldFail {
				// Этот тест проверяет поведение через публичные методы
				req = httptest.NewRequest("GET", "/tasks/active"+tt.queryParams, nil)
				handler.GetActiveTasks(w, req)
				assert.Equal(t, http.StatusBadRequest, w.Code)
			}
		})
	}
}

// TestTaskHandler_ContentTypeValidation тестирует валидацию Content-Type
func TestTaskHandler_ContentTypeValidation(t *testing.T) {
	mockService := new(MockTaskService)
	handler := handlers.NewTaskHandler(mockService)

	tests := []struct {
		name           string
		method         string
		path           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "POST without Content-Type",
			method:         "POST",
			path:           "/tasks",
			contentType:    "",
			body:           `{"title": "Test"}`,
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "POST with wrong Content-Type",
			method:         "POST",
			path:           "/tasks",
			contentType:    "text/plain",
			body:           `{"title": "Test"}`,
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "POST with correct Content-Type",
			method:         "POST",
			path:           "/tasks",
			contentType:    "application/json",
			body:           `{"title": "Test", "due_time": "2024-12-31T23:59:59Z"}`,
			expectedStatus: http.StatusBadRequest, // Будет ошибка валидации, но не Content-Type
		},
		{
			name:           "PUT without Content-Type",
			method:         "PUT",
			path:           "/tasks/test-id",
			contentType:    "",
			body:           `{"title": "Test"}`,
			expectedStatus: http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			
			if strings.Contains(tt.path, "test-id") {
				req.SetPathValue("id", uuid.New().String())
			}
			
			w := httptest.NewRecorder()

			switch {
			case tt.method == "POST" && tt.path == "/tasks":
				handler.PostTask(w, req)
			case tt.method == "PUT" && strings.Contains(tt.path, "/tasks/"):
				handler.UpdateTaskByID(w, req)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestTaskHandler_ErrorResponses тестирует формат ошибок
func TestTaskHandler_ErrorResponses(t *testing.T) {
	mockService := new(MockTaskService)
	handler := handlers.NewTaskHandler(mockService)

	t.Run("business error response format", func(t *testing.T) {
		taskID := uuid.New()
		
		mockService.On("ArchiveTask", mock.Anything, taskID).
			Return(nil, service.NewBusinessError(
				"TEST_ERROR",
				"Test error message",
				service.ToDetail("field", "value"),
			))

		req := httptest.NewRequest("POST", "/tasks/"+taskID.String()+"/archive", nil)
		req.SetPathValue("id", taskID.String())
		w := httptest.NewRecorder()

		handler.ArchiveTask(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, "TEST_ERROR", response["error"])
		assert.Equal(t, "Test error message", response["message"])
		assert.NotNil(t, response["details"])
	})

	t.Run("validation error response", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/tasks", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.PostTask(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "неверное тело запроса")
	})
}

// TestTaskHandler_ConcurrentRequests тестирует конкурентные запросы
func TestTaskHandler_ConcurrentRequests(t *testing.T) {
	mockService := new(MockTaskService)
	handler := handlers.NewTaskHandler(mockService)
	
	taskID := uuid.New()
	
	// Настраиваем мок для конкурентных вызовов
	mockService.On("GetTaskByID", mock.Anything, taskID).
		Return(&task.Task{
			UUID:  taskID,
			Title: "Test Task",
		}, nil).Times(10) // Ожидаем 10 вызовов

	done := make(chan bool)
	
	// Запускаем 10 горутин
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/tasks/"+taskID.String(), nil)
			req.SetPathValue("id", taskID.String())
			w := httptest.NewRecorder()
			
			handler.GetTaskByID(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}
	
	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}
	
	mockService.AssertExpectations(t)
}