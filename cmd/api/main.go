// cmd/test/main.go
package main

import (
	"context"
	"fmt"
	"taskTracker/internal/app"
	"taskTracker/internal/config"
)

func main() {
	// Тестовый конфиг
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: "8080",
		},
		Logging: config.LoggingConfig{
			Development: true,
		},
		Repository: config.RepositoryConfig{
			Type: "inmemory", // для быстрого теста
		},
	}

	app := app.New(cfg)
	ctx := context.Background()

	fmt.Println("🔄 Инициализация приложения...")
	if err := app.Init(ctx); err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	fmt.Println("✅ Приложение успешно инициализировано!")
	fmt.Println("   Тип сервиса:", cfg.Repository.Type)

	app.Run(ctx)
}
