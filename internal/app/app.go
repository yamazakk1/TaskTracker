package app

import (
	"context"
	"fmt"
	"net/http"
	"taskTracker/internal/config"
	"taskTracker/internal/handlers"
	"taskTracker/internal/logger"
	"taskTracker/internal/service"
	"taskTracker/internal/worker"

	"github.com/go-chi/chi/v5"
)

type App struct {
	config     *config.Config
	server     *http.Server
	router     *chi.Mux
	repository service.TaskRepository // интерфейс!
	service    handlers.Service
	worker     *worker.OverdueWorker
	shutdowns  []func() // функции для graceful shutdown
}

func New(cfg *config.Config) *App {
	return &App{
		config:    cfg,
		shutdowns: make([]func(), 0),
	}
}

func (a *App) Init(ctx context.Context) (*App, error) {

	if err := logger.Init(a.config.Logging.Development); err != nil {
		return nil, fmt.Errorf("инициализация логгера: %w", err)
	}

	a.shutdowns = append(a.shutdowns, func() {
		logger.Info("Завершение работы логгирования...")
		logger.Sync()
	})

	return nil, nil
}
