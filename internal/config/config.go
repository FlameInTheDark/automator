package config

import (
	"fmt"
	"os"
)

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	EncryptionKey string
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Path string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("AUTOMATOR_PORT", "8080"),
			Host: getEnv("AUTOMATOR_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Path: getEnv("AUTOMATOR_DB_PATH", "./automator.db"),
		},
		EncryptionKey: getEnv("AUTOMATOR_ENCRYPTION_KEY", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
