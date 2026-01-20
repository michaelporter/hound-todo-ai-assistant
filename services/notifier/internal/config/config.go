package config

import (
	"fmt"
	"os"
)

// Config holds the notifier service configuration
type Config struct {
	RabbitMQURL       string
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioPhoneNumber string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		RabbitMQURL:       os.Getenv("RABBITMQ_URL"),
		TwilioAccountSID:  os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:   os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioPhoneNumber: os.Getenv("TWILIO_PHONE_NUMBER"),
	}

	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	if cfg.TwilioAccountSID == "" {
		return nil, fmt.Errorf("TWILIO_ACCOUNT_SID is required")
	}
	if cfg.TwilioAuthToken == "" {
		return nil, fmt.Errorf("TWILIO_AUTH_TOKEN is required")
	}
	if cfg.TwilioPhoneNumber == "" {
		return nil, fmt.Errorf("TWILIO_PHONE_NUMBER is required")
	}

	return cfg, nil
}
