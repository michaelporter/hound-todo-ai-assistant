package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set required env vars
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("TWILIO_AUTH_TOKEN", "test-token")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("TWILIO_AUTH_TOKEN")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPPort != "9090" {
		t.Errorf("expected HTTPPort 9090, got %s", cfg.HTTPPort)
	}
	if cfg.RabbitMQURL != "amqp://localhost:5672/" {
		t.Errorf("expected RabbitMQURL amqp://localhost:5672/, got %s", cfg.RabbitMQURL)
	}
	if cfg.TwilioAuthToken != "test-token" {
		t.Errorf("expected TwilioAuthToken test-token, got %s", cfg.TwilioAuthToken)
	}
}

func TestLoad_MissingRabbitMQURL(t *testing.T) {
	// Ensure RABBITMQ_URL is not set
	os.Unsetenv("RABBITMQ_URL")

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when RABBITMQ_URL is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}

func TestLoad_DefaultHTTPPort(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Unsetenv("HTTP_PORT")
	defer os.Unsetenv("RABBITMQ_URL")

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("expected default HTTPPort 8080, got %s", cfg.HTTPPort)
	}
}

func TestLoad_EmptyTwilioAuthToken(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Unsetenv("TWILIO_AUTH_TOKEN")
	defer os.Unsetenv("RABBITMQ_URL")

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty auth token is allowed (disables signature validation)
	if cfg.TwilioAuthToken != "" {
		t.Errorf("expected empty TwilioAuthToken, got %s", cfg.TwilioAuthToken)
	}
}

func TestLoad_EmptyHTTPPort_UsesDefault(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("HTTP_PORT", "")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("HTTP_PORT")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty string should fall back to default
	if cfg.HTTPPort != "8080" {
		t.Errorf("expected default HTTPPort 8080 when empty, got %s", cfg.HTTPPort)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultVal   string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:       "returns env value when set",
			key:        "TEST_VAR_1",
			defaultVal: "default",
			envValue:   "custom",
			setEnv:     true,
			expected:   "custom",
		},
		{
			name:       "returns default when not set",
			key:        "TEST_VAR_2",
			defaultVal: "default",
			setEnv:     false,
			expected:   "default",
		},
		{
			name:       "returns default when empty",
			key:        "TEST_VAR_3",
			defaultVal: "default",
			envValue:   "",
			setEnv:     true,
			expected:   "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
