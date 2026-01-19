package config

import (
	"fmt"
	"os"
)

// Config holds the service configuration
type Config struct {
	HTTPPort        string
	RabbitMQURL     string
	TwilioAuthToken string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:        getEnvOrDefault("HTTP_PORT", "8080"),
		RabbitMQURL:     os.Getenv("RABBITMQ_URL"),
		TwilioAuthToken: os.Getenv("TWILIO_AUTH_TOKEN"),
	}

	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL environment variable is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
