package main

import (
	"net/http"
	"taskTracker/internal/handlers"
	"taskTracker/internal/repository/inmemory"
	"taskTracker/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	TaskRepo := inmemory.NewUserStorage()
	TaskService := service.NewTaskService(TaskRepo)
	TaskHandler := handlers.NewTaskHandler(TaskService)

	r := chi.NewRouter()
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/", TaskHandler.GetTasksWIthLimit)
		r.Post("/", TaskHandler.PostTask)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", TaskHandler.GetTaskByID)
			r.Put("/", TaskHandler.UpdateTaskByID)
			r.Delete("/", TaskHandler.DeleteTaskByID)
		})
	})
	r.Get("/health", TaskHandler.HealthCheck)
	http.ListenAndServe(":8080", r)
}
