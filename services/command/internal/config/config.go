package config

import (
	"fmt"
	"os"
)

// Config holds the configuration for command-svc
type Config struct {
	RabbitMQURL       string
	TodoDomainAddr    string // gRPC address for todo-domain-svc
	OpenAIAPIKey      string
	OpenAIModel       string // Optional override for model
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		RabbitMQURL:    os.Getenv("RABBITMQ_URL"),
		TodoDomainAddr: getEnvOrDefault("TODO_DOMAIN_GRPC_ADDR", "localhost:50051"),
		OpenAIAPIKey:   os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:    os.Getenv("OPENAI_MODEL"), // Optional
	}

	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL environment variable is required")
	}

	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
