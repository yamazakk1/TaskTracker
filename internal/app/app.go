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
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type App struct {
	config     *config.Config //
	server     *http.Server
	router     *chi.Mux
	repository service.TaskRepository //
	service    handlers.Service       //
	shutdowns  []func()               //
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

	// if a.config.Repository.Type == "postgres" {
	// 	if err := a.runMigrations(); err != nil {
	// 		return fmt.Errorf("миграции: %w", err)
	// 	}
	// }

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
	logger.Info("Попытка инициализации репозитория", zap.String("type", a.config.Repository.Type))

	switch a.config.Repository.Type {
	case "postgres":
		// 1. Подключаемся к БД
		conn, err := pgx.Connect(ctx, a.config.Database.URL)
		if err != nil {
			return nil, fmt.Errorf("подключение к БД: %w", err)
		}

		// 2. ВСЕ МИГРАЦИИ ЗДЕСЬ - ПРОСТО ВЫПОЛНЯЕМ SQL
		logger.Info("Применение миграций...")

		// Создаем таблицу
		_, err = conn.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS tasks (
				uuid        UUID PRIMARY KEY,
				title       TEXT NOT NULL,
				description TEXT,
				status      VARCHAR(20) NOT NULL,
				due_time    TIMESTAMPTZ NOT NULL,
				created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at  TIMESTAMPTZ,
				deleted_at  TIMESTAMPTZ,
				version     INTEGER NOT NULL DEFAULT 1,
				flag        VARCHAR(20) NOT NULL DEFAULT 'active'
			)
		`)
		if err != nil {
			conn.Close(ctx)
			return nil, fmt.Errorf("создание таблицы tasks: %w", err)
		}

		// Создаем индексы
		indexes := []string{
			`CREATE INDEX IF NOT EXISTS idx_tasks_flag ON tasks(flag)`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_active_created ON tasks(created_at DESC) WHERE flag = 'active'`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_overdue ON tasks(due_time, status) WHERE flag = 'active' AND status IN ('new', 'in progress')`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_archived_created ON tasks(created_at DESC) WHERE flag = 'archived'`,
			`CREATE INDEX IF NOT EXISTS idx_tasks_deleted_created ON tasks(created_at DESC) WHERE flag = 'deleted'`,
		}

		for i, idx := range indexes {
			_, err = conn.Exec(ctx, idx)
			if err != nil {
				conn.Close(ctx)
				return nil, fmt.Errorf("создание индекса %d: %w", i+1, err)
			}
		}

		logger.Info("Миграции успешно применены")

		// 3. Закрываем временное соединение
		conn.Close(ctx)

		// 4. Создаем репозиторий для работы приложения
		repo, err := postgres.New(ctx, a.config.Database.URL)
		if err != nil {
			return nil, err
		}

		// 5. Добавляем в shutdowns откат миграций при завершении
		a.shutdowns = append(a.shutdowns, func() {
			logger.Info("Откат миграций при завершении...")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, err := pgx.Connect(ctx, a.config.Database.URL)
			if err != nil {
				logger.Error("Ошибка подключения для отката", err)
				return
			}
			defer conn.Close(ctx)

			// УДАЛЯЕМ ИНДЕКСЫ
			dropIndexes := []string{
				`DROP INDEX IF EXISTS idx_tasks_deleted_created`,
				`DROP INDEX IF EXISTS idx_tasks_archived_created`,
				`DROP INDEX IF EXISTS idx_tasks_overdue`,
				`DROP INDEX IF EXISTS idx_tasks_active_created`,
				`DROP INDEX IF EXISTS idx_tasks_flag`,
			}

			for _, dropIdx := range dropIndexes {
				_, err := conn.Exec(ctx, dropIdx)
				if err != nil {
					logger.Error(fmt.Sprintf("Ошибка удаления индекса: %s", dropIdx), err)
				}
			}

			// УДАЛЯЕМ ТАБЛИЦУ
			_, err = conn.Exec(ctx, `DROP TABLE IF EXISTS tasks`)
			if err != nil {
				logger.Error("Ошибка удаления таблицы tasks", err)
			} else {
				logger.Info("Миграции успешно откачены")
			}
		})

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
