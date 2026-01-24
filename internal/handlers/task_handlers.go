package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"taskTracker/internal/logger"
	"taskTracker/internal/service"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TaskHandler struct {
	TaskService service.TaskService
}

func NewTaskHandler(taskService service.TaskService) TaskHandler {
	return TaskHandler{
		TaskService: taskService,
	}
}

func (s *TaskHandler) GetTasksWIthLimit(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logger.HttpRequestInfo(r, "HTTP_IN:")

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {

		logger.Warn("HTTP: Ошибка получения параметра",
			zap.Error(err),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "не удалось получить значение limit: "+err.Error())
		return
	}

	if limit == 0 {
		logger.Warn("HTTP: Неверное значение параметра",
			zap.String("querry", "limit"),
			zap.String("error", "0 value"),
			zap.String("client_ip", r.RemoteAddr))
		responseWithError(w, http.StatusBadRequest, "неверное значение limit")
		return
	}

	logger.Info("HTTP: Вызов сервиса для получения задач")

	tasks, err := s.TaskService.GetTasksWIthLimit(r.Context(), limit)
	if err != nil {
		logger.Error("HTTP: Ошибка Service", err)
		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("HTTP_OUT: Задачи получены",
		zap.Duration("ms", time.Since(start)),
		zap.Int("http_status", http.StatusOK))

	responseWithJSON(w, http.StatusOK, tasks)
}

func (s *TaskHandler) PostTask(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logger.HttpRequestInfo(r, "HTTP_IN:")

	if r.Method != http.MethodPost {

		logger.Warn("HTTP: Неверный метод",
			zap.String("expected", "POST"),
			zap.String("received", r.Method),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusMethodNotAllowed, "разрешён только POST метод")
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {

		logger.Warn("HTTP: Неверный тип контента",
			zap.String("expected", "application/json"),
			zap.String("received", r.Header.Get("Content-Type")),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusUnsupportedMediaType, "Content-Type должен быть application/json")
		return
	}

	var request CreateTaskRequest
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

	logger.Info("HTTP: Вызов сервиса создания задач")
	id, err := s.TaskService.CreateNewTask(r.Context(), request.Title, request.Description, request.DueTime)
	if err != nil {

		logger.Error("HTTP: Ошибка Service", err,
			zap.String("operation", "create_task"),
			zap.String("client_ip", r.RemoteAddr),
			zap.Duration("ms", time.Since(start)))

		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	logger.Info("HTTP_OUT: Задача создана",
		zap.String("task_id", id.String()),
		zap.Duration("ms", time.Since(start)),
		zap.Int("http_status", http.StatusCreated))

	responseWithJSON(w, http.StatusCreated, id)
}

func (s *TaskHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {

	logger.HttpRequestInfo(r, "HTTP: Health check")

	healthCheck(w)
}

func (s *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logger.HttpRequestInfo(r, "HTTP_IN:")

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {

		logger.Warn("HTTP: Не удалось получить id",
			zap.Error(err),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "не удалось получить id:"+err.Error())
		return
	}

	if id == uuid.Nil {

		logger.Warn("HTTP: Неверное значение id",
			zap.String("error", "nil id"),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "id не может быть пустым")
		return
	}

	logger.Info("HTTP: Вызов сервиса для получения задачи")

	task, err := s.TaskService.GetTaskByID(r.Context(), id)
	if err != nil {

		logger.Error("HTTP: Ошибка в Service", err,
			zap.String("operation", "get_task"),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("HTTP_OUT: Задача получена",
		zap.String("task_id", task.ID.String()),
		zap.Duration("ms", time.Since(start)),
		zap.Int("http_status", http.StatusOK))

	responseWithJSON(w, http.StatusOK, task)
}

func (s *TaskHandler) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	logger.HttpRequestInfo(r, "HTTP_IN:")

	if r.Method != http.MethodPut {

		logger.Warn("HTTP: Неверный метод",
			zap.String("expected", "PUT"),
			zap.String("received", r.Method),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusMethodNotAllowed, "только PUT метод доступен")
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {

		logger.Warn("HTTP: Не удалось получить id",
			zap.Error(err),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "не удалось получить id:"+err.Error())
		return
	}

	if id == uuid.Nil {

		logger.Warn("HTTP: Неверное значение id",
			zap.String("error", "nil id"),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "id не может быть пустым")
		return
	}

	var request UpdateTaskRequest

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err = decoder.Decode(&request)
	if err != nil {

		logger.Warn("HTTP: ошибка чтения JSON",
			zap.Error(err),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "неверно переданы параметры обновления:"+err.Error())
		return
	}

	logger.Info("HTTP: запрос к сервису обновления данных")

	err = s.TaskService.UpdateTaskByID(r.Context(), id, service.WithDescription(*request.Description), service.WithDueTime(*request.DueTime), service.WithStatus(*request.Status), service.WithTitle(*request.Title))
	if err != nil {

		logger.Error("HTTP: ошибка в Service", err,
			zap.String("operation", "update_task"),
			zap.String("client_addr", r.RemoteAddr))

		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("HTTP_OUT: Задача обновлена",
		zap.Duration("ms", time.Since(start)),
		zap.Int("http_status", http.StatusOK))

	responseWithJSON(w, http.StatusOK, request)
}

func (s *TaskHandler) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	logger.HttpRequestInfo(r, "HTTP_IN:")

	if r.Method != http.MethodDelete {

		logger.Warn("HTTP: Неверный метод",
			zap.String("expected", "DELETE"),
			zap.String("received", r.Method),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusMethodNotAllowed, "только DELETE метод доступен")
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {

		logger.Warn("HTTP: ошибка чтения JSON",
			zap.Error(err),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "не удалось получить id"+err.Error())
		return
	}

	if id == uuid.Nil {

		logger.Warn("HTTP: Неверное значение id",
			zap.String("error", "nil id"),
			zap.String("client_ip", r.RemoteAddr))

		responseWithError(w, http.StatusBadRequest, "id не может быть пустым")
		return
	}

	logger.Info("HTTP: Обращение к сервису для удаления задачи")

	err = s.TaskService.DeleteTaskByID(r.Context(), id)
	if err != nil {

		logger.Error("HTTP: ошибка в Service", err,
			zap.String("operation", "delete_task"),
			zap.String("client_addr", r.RemoteAddr))

		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	logger.Info("HTTP_OUT: Задача удалена",
		zap.Duration("ms", time.Since(start)),
		zap.Int("http_status", http.StatusNoContent))

	responseWithJSON(w, http.StatusNoContent, "No content")
}
