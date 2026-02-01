package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Logging    LoggingConfig
	Worker     WorkerConfig
	Repository RepositoryConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	URL            string
	MaxConnections int
	MinConnections int
	IdleTimeout    time.Duration
}

type LoggingConfig struct {
	Development bool
}

type WorkerConfig struct {
	Interval  time.Duration
	BatchSize int
}

type RepositoryConfig struct {
	Type string
}

// ВАЖНО: Убираем ошибку, всегда возвращаем Config
func Load() (*Config, error) {
	// Всегда создаем конфиг из env
	cfg := LoadFromEnv()
	return cfg, nil
}

func LoadFromEnv() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL",
				"postgres://postgres:password@localhost:5432/tasktracker?sslmode=disable"),
			MaxConnections: getEnvAsInt("DB_MAX_CONNECTIONS", 10),
			MinConnections: getEnvAsInt("DB_MIN_CONNECTIONS", 2),
			IdleTimeout:    getEnvAsDuration("DB_IDLE_TIMEOUT", 5*time.Minute),
		},
		Logging: LoggingConfig{
			Development: getEnvAsBool("LOGGING_DEVELOPMENT", true),
		},
		Worker: WorkerConfig{
			Interval:  getEnvAsDuration("WORKER_INTERVAL", 5*time.Minute),
			BatchSize: getEnvAsInt("WORKER_BATCH_SIZE", 100),
		},
		Repository: RepositoryConfig{
			Type: getEnv("REPOSITORY_TYPE", "postgres"),
		},
	}
}

func (c *Config) GetServerAddr() string {
	return c.Server.Host + ":" + c.Server.Port
}

// Вспомогательные функции
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
