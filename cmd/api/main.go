package main

import (
    "context"
    "log"
    "taskTracker/internal/app"
    "taskTracker/internal/config"
)

func main() {
    // Загружаем конфигурацию (теперь всегда работает)
    cfg, err := config.Load()
    if err != nil {
        // Если ошибка - создаем дефолтный конфиг
        log.Printf("Warning: config error: %v, using defaults", err)
        cfg = config.LoadFromEnv()
    }

    // Создаем приложение
    application := app.New(cfg)
    
    // Создаем корневой контекст
    ctx := context.Background()
    
    // Инициализируем все компоненты
    if err := application.Init(ctx); err != nil {
        log.Fatalf("Failed to init app: %v", err)
    }

    log.Println("Application initialized")
    log.Println("Server starting on:", cfg.GetServerAddr())
    
    // Запускаем приложение
    if err := application.Run(ctx); err != nil {
        log.Fatalf("Application error: %v", err)
    }

    log.Println("Application stopped")
}