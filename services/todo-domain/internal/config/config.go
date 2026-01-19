package config

import (
	"fmt"
	"os"
)

// Config holds the configuration for todo-domain-svc
type Config struct {
	GRPCPort    string
	DatabaseURL string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:    getEnvOrDefault("GRPC_PORT", "50051"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
