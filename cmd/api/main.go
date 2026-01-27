package main

import (
	"net/http"
	"taskTracker/internal/handlers"
	"taskTracker/internal/logger"
	"taskTracker/internal/middleware"
	"taskTracker/internal/repository/task/inmemory"
	"taskTracker/internal/service"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	TaskRepo := inmemory.NewTaskStorage()
	TaskService := service.NewTaskService(TaskRepo, service.InMemoryType)
	TaskHandler := handlers.NewTaskHandler(&TaskService)
	logger.Init(true)
	defer logger.Sync()
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.RateLimit(100))
	
	r.Route("/tasks", func(r chi.Router) {

		r.Get("/", TaskHandler.GetActiveTasks) // GET /tasks
		r.Post("/", TaskHandler.PostTask)      // POST /tasks

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", TaskHandler.GetTaskByID)       // GET /tasks/{id}
			r.Put("/", TaskHandler.UpdateTaskByID)    // PUT /tasks/{id}
			r.Delete("/", TaskHandler.DeleteTaskByID) // DELETE /tasks/{id}

			r.Post("/archive", TaskHandler.ArchiveTask)     // POST /tasks/{id}/archive
			r.Post("/unarchive", TaskHandler.UnarchiveTask) // POST /tasks/{id}/unarchive
		})

		r.Get("/archived", TaskHandler.GetArchivedTasks) // GET /tasks/archived
		r.Get("/all", TaskHandler.GetAllTasks)           // GET /tasks/all
		r.Get("/overdue", TaskHandler.GetOverdueTasks)   // GET /tasks/overdue
	})

	r.Route("/admin/tasks", func(r chi.Router) {
		r.Get("/deleted", TaskHandler.GetDeletedTasks) // GET /admin/tasks/deleted

		r.Route("/{id}", func(r chi.Router) {
			r.Post("/restore", TaskHandler.RestoreTask) // POST /admin/tasks/{id}/restore
			r.Delete("/purge", TaskHandler.PurgeTask)   // DELETE /admin/tasks/{id}/purge
		})
	})

	r.Get("/health", TaskHandler.HealthCheck)
	logger.Info("Server started")
	http.ListenAndServe(":8080", r)
}
