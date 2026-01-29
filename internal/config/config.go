// internal/config/config.go
package config

import (
    "fmt"
    "os"
    "time"
    
    "gopkg.in/yaml.v3"
)

type Config struct {
    Server     ServerConfig     `yaml:"server"`
    Database   DatabaseConfig   `yaml:"database"`
    Logging    LoggingConfig    `yaml:"logging"`
    Repository RepositoryConfig `yaml:"repository"`
}

type ServerConfig struct {
    Port string `yaml:"port"`
    Host string `yaml:"host"`
}

type DatabaseConfig struct {
    URL           string        `yaml:"url"`
    MaxConnections int           `yaml:"max_connections"`
    MinConnections int           `yaml:"min_connections"`
    IdleTimeout   time.Duration `yaml:"idle_timeout"`
}

type LoggingConfig struct {
    Development bool `yaml:"development"`
}


type RepositoryConfig struct {
    Type string `yaml:"type"` // "postgres" или "inmemory"
}

func Load() (*Config, error) {
    file, err := os.Open("config.yml")
    if err != nil {
        return nil, fmt.Errorf("не могу открыть config.yml: %w", err)
    }
    defer file.Close()
    
    var cfg Config
    decoder := yaml.NewDecoder(file)
    if err := decoder.Decode(&cfg); err != nil {
        return nil, fmt.Errorf("ошибка парсинга config.yml: %w", err)
    }
    
    return &cfg, nil
}

func (c *Config) GetServerAddr() string {
    return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}