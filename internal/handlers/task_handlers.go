package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"taskTracker/internal/service"
	"time"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TaskHandler struct {
	TaskService service.TaskService
}

func NewTaskHandler(taskService service.TaskService) TaskHandler {
	return TaskHandler{
		TaskService: taskService,
	}
}

// POST /tasks - создание задачи СДЕЛАЛ
// GET /tasks - получение списка задач (с фильтрацией и пагинацией) СДЕЛАЛФильтрация по статусу (Task.status) и по дедлайну (Task.due_date).
// GET /tasks/{id} - получение задачи по id СДЕЛАЛ
// PUT /tasks/{id} - обновление задачи по id СДЕЛАЛ
// DELET /tasks/{id} - soft delete задачи

// тут происходит валидация основных ошибок полученных данных


func (s *TaskHandler) GetTasksWIthLimit(w http.ResponseWriter, r *http.Request) {

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		responseWithError(w, http.StatusBadRequest, "не удалось получить значение limit: " + err.Error() )
		return
	}

	tasks, err := s.TaskService.GetTasksWIthLimit(r.Context(), limit)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err.Error())
	}
	responseWithJSON(w, http.StatusOK, tasks)
}

func (s *TaskHandler) PostTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responseWithError(w, http.StatusMethodNotAllowed, "разрешён только POST метод")
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		responseWithError(w, http.StatusUnsupportedMediaType, "Content-Type должен быть application/json")
		return
	}

	var request CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		responseWithError(w, http.StatusBadRequest, "неверное тело запроса:" + err.Error())
		return
	}

	if request.Title == "" {
		responseWithError(w, http.StatusBadRequest, "название не может быть пустым")
		return
	}

	if request.DueTime.IsZero() {
		responseWithError(w, http.StatusBadRequest, "дедлайн должен быть задан")
	}
	
	if time.Now().After(request.DueTime){
		responseWithError(w,http.StatusBadRequest, "дедлайн не может быть в прошлом")
	}

	err := s.TaskService.CreateNewTask(r.Context(), request.Title, request.Description, request.DueTime)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responseWithJSON(w, http.StatusCreated, request)
}

func (s *TaskHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	healthCheck(w)
}

func (s *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		responseWithError(w, http.StatusBadRequest, "не удалось получить id:" + err.Error())
		return
	}

	if id == uuid.Nil {
		responseWithError(w, http.StatusBadRequest, "id не может быть пустым")
		return
	}

	task, err := s.TaskService.GetTaskByID(r.Context(), id)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responseWithJSON(w, http.StatusOK, task)
}

func (s *TaskHandler) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		responseWithError(w, http.StatusMethodNotAllowed, "только PUT метод доступен")
		return
	}
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		responseWithError(w, http.StatusBadRequest, "не удалось получить id:" + err.Error())
		return
	}

	if id == uuid.Nil {
		responseWithError(w, http.StatusBadRequest, "id не может быть пустым")
		return
	}

	var request UpdateTaskRequest

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err = decoder.Decode(&request)
	if err != nil {
		responseWithError(w, http.StatusBadRequest, "неверно переданы параметры обновления:" + err.Error())
		return
	}

	err = s.TaskService.UpdateTaskByID(r.Context(), id, service.WithDescription(*request.Description), service.WithDueTime(*request.DueTime), service.WithStatus(*request.Status), service.WithTitle(*request.Title))
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responseWithJSON(w, http.StatusOK, request)
}

func (s *TaskHandler) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		responseWithError(w, http.StatusMethodNotAllowed, "только DELETE метод доступен")
		return
	}
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		responseWithError(w, http.StatusBadRequest, "не удалось получить id" + err.Error())
		return
	}

	if id == uuid.Nil {
		responseWithError(w, http.StatusBadRequest, "id не может быть пустым")
		return
	}

	err = s.TaskService.DeleteTaskByID(r.Context(), id)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	responseWithJSON(w, http.StatusNoContent, "No content")
}
