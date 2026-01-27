package handlers

import (
	"encoding/json"
	"net/http"
	"taskTracker/internal/handlers/dto"
	"taskTracker/internal/logger"
	"taskTracker/internal/models/task"
	"taskTracker/internal/service"
	"time"

	"go.uber.org/zap"
)

type TaskHandler struct {
	TaskService TaskService
}

func NewTaskHandler(taskService TaskService) TaskHandler {
	return TaskHandler{
		TaskService: taskService,
	}
}

func (s *TaskHandler) GetActiveTasks(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    
    page, limit, ok := validatePagination(w, r)
    if !ok {
        return
    }
    
    logger.Info("HTTP: Получение активных задач",
        zap.Int("page", page),
        zap.Int("limit", limit))
    
    tasks, err := s.TaskService.GetActiveTasks(r.Context(), page, limit)
    if err != nil {
        logger.Error("HTTP: Ошибка получения активных задач", err)
        responseWithError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    logger.Info("HTTP_OUT: Активные задачи получены",
        zap.Duration("ms", time.Since(start)),
        zap.Int("count", len(tasks)),
        zap.Int("http_status", http.StatusOK))
    
    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTaskList(tasks))
}

func (s *TaskHandler) PostTask(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")

    if !checkContentType(r, "application/json") {
        logger.Warn("HTTP: Неверный тип контента",
            zap.String("expected", "application/json"),
            zap.String("received", r.Header.Get("Content-Type")),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusUnsupportedMediaType, "Content-Type должен быть application/json")
        return
    }

    var request dto.CreateTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        logger.Warn("HTTP: ошибка чтения JSON",
            zap.Error(err),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "неверное тело запроса:"+err.Error())
        return
    }

    
    if request.Title == "" {
        logger.Warn("HTTP: Ошибка валидации",
            zap.String("field", "title"),
            zap.String("error", "empty_field"),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "название не может быть пустым")
        return
    }

    if request.DueTime.IsZero() {
        logger.Warn("HTTP: Ошибка валидации",
            zap.String("field", "due_time"),
            zap.String("error", "empty_field"),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "дедлайн должен быть задан")
        return
    }

    if time.Now().After(request.DueTime) {
        logger.Warn("HTTP: Ошибка валидации",
            zap.String("field", "due_time"),
            zap.String("error", "wrong_value"),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "дедлайн не может быть в прошлом")
        return
    }

    logger.Info("HTTP: Вызов сервиса создания задачи")
    
    createdTask, err := s.TaskService.CreateTask(r.Context(), request.Title, request.Description, request.DueTime)
    if err != nil {
        logger.Error("HTTP: Ошибка Service", err,
            zap.String("operation", "create_task"),
            zap.String("client_ip", r.RemoteAddr),
            zap.Duration("ms", time.Since(start)))
        responseWithError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    logger.Info("HTTP_OUT: Задача создана",
        zap.String("task_id", createdTask.UUID.String()),
        zap.Duration("ms", time.Since(start)),
        zap.Int("http_status", http.StatusCreated))

    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTask(createdTask))
}

func (s *TaskHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP: Health check")
    
    err := s.TaskService.HealthCheck(r.Context())
    
    if err != nil {
        logger.Error("HTTP: Health check failed", err,
            zap.Duration("ms", time.Since(start)))
        
        responseWithJSON(w, http.StatusServiceUnavailable, 
            toPayload("status",  "unhealthy"),
            toPayload("error",   err.Error()),
            toPayload("service", "task-tracker"),
            toPayload("timestamp", time.Now().Format(time.RFC3339)),
        )
        return
    }
    
    logger.Info("HTTP: Health check passed",
        zap.Duration("ms", time.Since(start)))
    
    responseWithJSON(w, http.StatusOK, 
        toPayload("status",    "healthy"),
        toPayload("service",   "task-tracker"),
        toPayload("timestamp", time.Now().Format(time.RFC3339)),
    )
}

func (s *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")

    id, ok := validateUUID(w, r, "id")
    if !ok {
        return
    }

    logger.Info("HTTP: Вызов сервиса для получения задачи",
        zap.String("task_id", id.String()))

    task, err := s.TaskService.GetTaskByID(r.Context(), id)
    if err != nil {
        if handleBusinessError(w, err, "ошибка получения задачи") {
            return
        }
        
        logger.Error("HTTP: Системная ошибка в Service", err,
            zap.String("operation", "get_task"),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }

    logger.Info("HTTP_OUT: Задача получена",
        zap.String("task_id", task.UUID.String()),
        zap.Duration("ms", time.Since(start)),
        zap.Int("http_status", http.StatusOK))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTask(task))
}

func (s *TaskHandler) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")

    if !checkContentType(r, "application/json") {
        logger.Warn("HTTP: Неверный тип контента",
            zap.String("expected", "application/json"),
            zap.String("received", r.Header.Get("Content-Type")),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusUnsupportedMediaType, "Content-Type должен быть application/json")
        return
    }

    id, ok := validateUUID(w, r, "id")
    if !ok {
        return
    }

    var request dto.UpdateTaskRequest
    decoder := json.NewDecoder(r.Body)
    defer r.Body.Close()

    err := decoder.Decode(&request)
    if err != nil {
        logger.Warn("HTTP: ошибка чтения JSON",
            zap.Error(err),
            zap.String("client_ip", r.RemoteAddr))
        responseWithError(w, http.StatusBadRequest, "неверно переданы параметры обновления:"+err.Error())
        return
    }
    
    opts := []task.TaskOption{}

    if request.Status != nil {
        opts = append(opts, task.WithStatus(*request.Status))
    }

    if request.Description != nil {
        opts = append(opts, task.WithDescription(*request.Description))
    }

    if request.Title != nil {
        opts = append(opts, task.WithTitle(*request.Title))
    }

    if request.DueTime != nil {
        opts = append(opts, task.WithDueTime(*request.DueTime))
    }

    logger.Info("HTTP: запрос к сервису обновления данных",
        zap.String("task_id", id.String()))

    updatedTask, err := s.TaskService.UpdateTask(r.Context(), id, opts...)
    if err != nil {
        if handleBusinessError(w, err, "ошибка обновления задачи") {
            return
        }
        
        logger.Error("HTTP: ошибка в Service", err,
            zap.String("operation", "update_task"),
            zap.String("client_addr", r.RemoteAddr))
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }

    logger.Info("HTTP_OUT: Задача обновлена",
        zap.Duration("ms", time.Since(start)),
        zap.String("task_id", id.String()),
        zap.Int("http_status", http.StatusOK))

    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTask(updatedTask))
}

func (s *TaskHandler) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")

    id, ok := validateUUID(w, r, "id")
    if !ok {
        return
    }

    logger.Info("HTTP: Обращение к сервису для удаления задачи",
        zap.String("task_id", id.String()))

    err := s.TaskService.DeleteTask(r.Context(), id)
    if err != nil {
        if handleBusinessError(w, err, "ошибка удаления задачи") {
            return
        }
        
        logger.Error("HTTP: ошибка в Service", err,
            zap.String("operation", "delete_task"),
            zap.String("client_addr", r.RemoteAddr))
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }

    logger.Info("HTTP_OUT: Задача удалена",
        zap.Duration("ms", time.Since(start)),
        zap.String("task_id", id.String()),
        zap.Int("http_status", http.StatusNoContent))

    w.WriteHeader(http.StatusNoContent)
}

func (s *TaskHandler) GetArchivedTasks(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    
    page, limit, ok := validatePagination(w, r)
    if !ok {
        return
    }
    
    logger.Info("HTTP: Получение архивных задач",
        zap.Int("page", page),
        zap.Int("limit", limit))
    
    tasks, err := s.TaskService.GetArchivedTasks(r.Context(), page, limit)
    if err != nil {
        logger.Error("HTTP: Ошибка получения архивных задач", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Архивные задачи получены",
        zap.Duration("ms", time.Since(start)),
        zap.Int("count", len(tasks)),
        zap.Int("http_status", http.StatusOK))
    
    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTaskList(tasks))
}

func (s *TaskHandler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    
    page, limit, ok := validatePagination(w, r)
    if !ok {
        return
    }
    
    logger.Info("HTTP: Получение всех задач",
        zap.Int("page", page),
        zap.Int("limit", limit))
    
    tasks, err := s.TaskService.GetAllTasks(r.Context(), page, limit)
    if err != nil {
        logger.Error("HTTP: Ошибка получения всех задач", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Все задачи получены",
        zap.Duration("ms", time.Since(start)),
        zap.Int("count", len(tasks)),
        zap.Int("http_status", http.StatusOK))
    
    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTaskList(tasks))
}

func (s *TaskHandler) GetOverdueTasks(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    
   
    page, limit, ok := validatePagination(w, r)
    if !ok {
        return
    }
    
    logger.Info("HTTP: Получение просроченных задач",
        zap.Int("page", page),
        zap.Int("limit", limit))
    
    tasks, err := s.TaskService.GetOverdueTasks(r.Context(), page, limit)
    if err != nil {
        logger.Error("HTTP: Ошибка получения просроченных задач", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Просроченные задачи получены",
        zap.Duration("ms", time.Since(start)),
        zap.Int("count", len(tasks)),
        zap.Int("http_status", http.StatusOK))
    
    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTaskList(tasks))
}

func (s *TaskHandler) GetDeletedTasks(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    

    page, limit, ok := validatePagination(w, r)
    if !ok {
        return
    }
    
    logger.Info("HTTP: Получение удаленных задач",
        zap.Int("page", page),
        zap.Int("limit", limit))
    
    tasks, err := s.TaskService.GetDeletedTasks(r.Context(), page, limit)
    if err != nil {
        logger.Error("HTTP: Ошибка получения удаленных задач", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Удаленные задачи получены",
        zap.Duration("ms", time.Since(start)),
        zap.Int("count", len(tasks)),
        zap.Int("http_status", http.StatusOK))
    
    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTaskList(tasks))
}

func (s *TaskHandler) ArchiveTask(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    
    
    id, ok := validateUUID(w, r, "id")
    if !ok {
        return
    }
    
    logger.Info("HTTP: Запрос на архивацию задачи",
        zap.String("task_id", id.String()))
    
    archivedTask, err := s.TaskService.ArchiveTask(r.Context(), id)
    if err != nil {
        if businessErr, ok := err.(*service.BusinessError); ok {
            statusCode := http.StatusBadRequest
            switch businessErr.Code {
            case "ALREADY_ARCHIVED":
                statusCode = http.StatusConflict
            case "TASK_DELETED":
                statusCode = http.StatusGone
            case "VERSION_CONFLICT":
                statusCode = http.StatusConflict
            }
            
            logger.Warn("HTTP: Бизнес-ошибка при архивации",
                zap.String("error_code", businessErr.Code),
                zap.String("task_id", id.String()),
                zap.Int("http_status", statusCode))
            
            responseWithJSON(w, statusCode, 
                toPayload("error", businessErr.Code),
                toPayload("message", businessErr.Message),
                toPayload("details", businessErr.Details),
            )
            return
        }
        
        logger.Error("HTTP: Системная ошибка при архивации", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Задача архивирована",
        zap.Duration("ms", time.Since(start)),
        zap.String("task_id", id.String()),
        zap.Int("http_status", http.StatusOK))
    
   w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTask(archivedTask))
}

func (s *TaskHandler) UnarchiveTask(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    
    id, ok := validateUUID(w, r, "id")
    if !ok {
        return
    }
    
    logger.Info("HTTP: Запрос на разархивацию задачи",
        zap.String("task_id", id.String()))

    unarchivedTask, err := s.TaskService.UnarchiveTask(r.Context(), id)
    if err != nil {
        if businessErr, ok := err.(*service.BusinessError); ok {
            statusCode := http.StatusBadRequest
            switch businessErr.Code {
            case "NOT_ARCHIVED":
                statusCode = http.StatusConflict
            case "TASK_DELETED":
                statusCode = http.StatusGone
            case "VERSION_CONFLICT":
                statusCode = http.StatusConflict
            }
            
            logger.Warn("HTTP: Бизнес-ошибка при разархивации",
                zap.String("error_code", businessErr.Code),
                zap.String("task_id", id.String()),
                zap.Int("http_status", statusCode))
            
            responseWithJSON(w, statusCode, 
                toPayload("error", businessErr.Code),
                toPayload("message", businessErr.Message),
                toPayload("details", businessErr.Details),
            )
            return
        }
        
        logger.Error("HTTP: Системная ошибка при разархивации", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Задача разархивирована",
        zap.Duration("ms", time.Since(start)),
        zap.String("task_id", id.String()),
        zap.Int("http_status", http.StatusOK))
    
    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTask(unarchivedTask))
}

func (s *TaskHandler) RestoreTask(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
 
    id, ok := validateUUID(w, r, "id")
    if !ok {
        return
    }

    if !checkContentType(r, "application/json") && r.ContentLength > 0 {
        logger.Warn("HTTP: Неверный тип контента",
            zap.String("expected", "application/json"),
            zap.String("received", r.Header.Get("Content-Type")))
        responseWithError(w, http.StatusUnsupportedMediaType, "Content-Type должен быть application/json")
        return
    }
    
    logger.Info("HTTP: Запрос на восстановление задачи",
        zap.String("task_id", id.String()))

    restoredTask, err := s.TaskService.RestoreTask(r.Context(), id)
    if err != nil {
        if businessErr, ok := err.(*service.BusinessError); ok {
            statusCode := http.StatusBadRequest
            switch businessErr.Code {
            case "NOT_DELETED":
                statusCode = http.StatusConflict
            case "RESTORE_EXPIRED":
                statusCode = http.StatusGone
            case "VERSION_CONFLICT":
                statusCode = http.StatusConflict
            }
            
            logger.Warn("HTTP: Бизнес-ошибка при восстановлении",
                zap.String("error_code", businessErr.Code),
                zap.String("task_id", id.String()),
                zap.Int("http_status", statusCode))
            
            responseWithJSON(w, statusCode, 
                toPayload("error", businessErr.Code),
                toPayload("message", businessErr.Message),
                toPayload("details", businessErr.Details),
            )
            return
        }
        
        logger.Error("HTTP: Системная ошибка при восстановлении", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Задача восстановлена",
        zap.Duration("ms", time.Since(start)),
        zap.String("task_id", id.String()),
        zap.Int("http_status", http.StatusOK))
    
    w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(dto.FromTask(restoredTask))
}

func (s *TaskHandler) PurgeTask(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    logger.HttpRequestInfo(r, "HTTP_IN:")
    
    id, ok := validateUUID(w, r, "id")
    if !ok {
        return
    }
    
    logger.Info("HTTP: Запрос на полное удаление задачи",
        zap.String("task_id", id.String()))
    
    err := s.TaskService.PurgeTask(r.Context(), id)
    if err != nil {
        if businessErr, ok := err.(*service.BusinessError); ok {
            statusCode := http.StatusBadRequest
            if businessErr.Code == "NOT_DELETED" {
                statusCode = http.StatusConflict
            }
            
            logger.Warn("HTTP: Бизнес-ошибка при полном удалении",
                zap.String("error_code", businessErr.Code),
                zap.String("task_id", id.String()),
                zap.Int("http_status", statusCode))
            
            responseWithJSON(w, statusCode, 
                toPayload("error", businessErr.Code),
                toPayload("message", businessErr.Message),
                toPayload("details", businessErr.Details),
            )
            return
        }
        
        logger.Error("HTTP: Системная ошибка при полном удалении", err)
        responseWithError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
        return
    }
    
    logger.Info("HTTP_OUT: Задача полностью удалена",
        zap.Duration("ms", time.Since(start)),
        zap.String("task_id", id.String()),
        zap.Int("http_status", http.StatusNoContent))
    
    w.WriteHeader(http.StatusNoContent)
}