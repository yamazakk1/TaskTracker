package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"taskTracker/internal/config"
	"taskTracker/internal/handlers"
	"taskTracker/internal/logger"
	"taskTracker/internal/middleware"
	"taskTracker/internal/repository/task/inmemory"
	"taskTracker/internal/repository/task/postgres"
	"taskTracker/internal/service"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type App struct {
	config     *config.Config //
	server     *http.Server
	router     *chi.Mux
	repository service.TaskRepository //
	service    handlers.Service       //
	shutdowns  []func() //
}

func New(cfg *config.Config) *App {
	return &App{
		config:    cfg,
		shutdowns: make([]func(), 0),
	}
}

// 1. Логгер - сделал
// 2. Репозиторий (зависит только от логгера) - сделал
// 3. Сервис (зависит от репозитория) - сделал
// 4. Роутер и хендлеры (зависит от сервиса) - сделал
// 5. Сервер (зависит от роутера) - сделал
// 6. Воркер (зависит от репозитория)

func (a *App) Init(ctx context.Context) error {
	// логгер
	if err := logger.Init(a.config.Logging.Development); err != nil {
		return fmt.Errorf("инициализация логгера: %w", err)
	}
	logger.Info("Успешная инициализация логгера")

	a.shutdowns = append(a.shutdowns, func() {
		logger.Info("Завершение работы логгирования...")
		logger.Sync()
	})

	// репозиторий
	repo, err := a.initRepository(ctx)
	if err != nil {
		return fmt.Errorf("инициализация репозитория: %w", err)
	}
	a.repository = repo
	logger.Info("Успешная инициализация репозитория", zap.String("type", a.config.Repository.Type))

	// сервис
	servi, err := a.initService()
	if err != nil {
		return fmt.Errorf("инициализация сервиса: %w", err)
	}
	a.service = servi
	logger.Info("Успешная инициализация сервиса")

	//хендлеры и роутинг
	a.initRouter()
	logger.Info("Успешная инициализация роутера")

	// сервер
	a.initServer()
	logger.Info("Успешная инициализация сервера")


	logger.Info("Приложение успешно инициализировано")
	return nil
}

func (a *App) Run(ctx context.Context) error {


	serverErr := make(chan error, 1)

	go func() {
		logger.Info("Запуск сервера", zap.String("addr", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Приложение запущено. Ctrl+C для остановки.")

	select {
	case <-quit:
		logger.Info("Получен сигнал завершения")
	case err := <-serverErr:
		logger.Error("Ошибка сервера", err)
	case <-ctx.Done():
		logger.Info("Контекст отменен")
	}

	return a.Shutdown(context.Background())
}

func (a *App) Shutdown(ctx context.Context) error {
	logger.Info("Начало graceful shutdown")

	// Выполняем в обратном порядке
	for i := len(a.shutdowns) - 1; i >= 0; i-- {
		a.shutdowns[i]()
	}

	logger.Info("Graceful shutdown завершен")
	return nil
}

func (a *App) initRepository(ctx context.Context) (service.TaskRepository, error) {
	logger.Info("Попытка инициализация репозитория", zap.String("type", a.config.Repository.Type))

	switch a.config.Repository.Type {
	case "postgres":
		repo, err := postgres.New(ctx, a.config.Database.URL)
		if err != nil {
			return nil, err
		}

		a.shutdowns = append(a.shutdowns, func() {
			logger.Info("Завершение работы БД...")
			repo.Close()
		})
		return repo, nil
	case "inmemory":
		repo := inmemory.NewTaskStorage()
		return repo, nil
	default:
		return nil, fmt.Errorf("неизвестный тип репозитория: %s", a.config.Repository.Type)
	}

}

func (a *App) initService() (handlers.Service, error) {
	logger.Info("Попытка инициализации сервиса")

	switch a.config.Repository.Type {
	case "postgres":
		ser := service.NewTaskService(a.repository, "postgres")

		return &ser, nil
	case "inmemory":
		service := service.NewTaskService(a.repository, "inmemory")
		return &service, nil
	default:
		return nil, fmt.Errorf("неизвестный тип репозитория")
	}

}

func (a *App) initRouter() {
	TaskHandler := handlers.NewTaskHandler(a.service)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	// r.Use(middleware.Timeout(30 * time.Second))
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

	a.router = r
}

func (a *App) initServer() {
	a.server = &http.Server{
		Addr:         a.config.GetServerAddr(),
		Handler:      a.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	a.shutdowns = append(a.shutdowns,
		func() {
			logger.Info("Graceful shutdown сервера")
			shutdownctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if err := a.server.Shutdown(shutdownctx); err != nil {
				logger.Error("Ошибка Shutdown сервера", err)
			}
		})

}

