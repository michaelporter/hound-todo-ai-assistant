package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	os.Setenv("TODO_DOMAIN_GRPC_ADDR", "localhost:50052")
	os.Setenv("OPENAI_MODEL", "gpt-4")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("TODO_DOMAIN_GRPC_ADDR")
		os.Unsetenv("OPENAI_MODEL")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RabbitMQURL != "amqp://localhost:5672/" {
		t.Errorf("expected RabbitMQURL amqp://localhost:5672/, got %s", cfg.RabbitMQURL)
	}
	if cfg.OpenAIAPIKey != "sk-test-key" {
		t.Errorf("expected OpenAIAPIKey sk-test-key, got %s", cfg.OpenAIAPIKey)
	}
	if cfg.TodoDomainAddr != "localhost:50052" {
		t.Errorf("expected TodoDomainAddr localhost:50052, got %s", cfg.TodoDomainAddr)
	}
	if cfg.OpenAIModel != "gpt-4" {
		t.Errorf("expected OpenAIModel gpt-4, got %s", cfg.OpenAIModel)
	}
}

func TestLoad_MissingRabbitMQURL(t *testing.T) {
	os.Unsetenv("RABBITMQ_URL")
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when RABBITMQ_URL is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}

func TestLoad_MissingOpenAIAPIKey(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Unsetenv("OPENAI_API_KEY")
	defer os.Unsetenv("RABBITMQ_URL")

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when OPENAI_API_KEY is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}

func TestLoad_DefaultTodoDomainAddr(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	os.Unsetenv("TODO_DOMAIN_GRPC_ADDR")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TodoDomainAddr != "localhost:50051" {
		t.Errorf("expected default TodoDomainAddr localhost:50051, got %s", cfg.TodoDomainAddr)
	}
}

func TestLoad_OptionalOpenAIModel(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	os.Unsetenv("OPENAI_MODEL")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty model is OK - service will use default
	if cfg.OpenAIModel != "" {
		t.Errorf("expected empty OpenAIModel, got %s", cfg.OpenAIModel)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		envValue   string
		setEnv     bool
		expected   string
	}{
		{
			name:       "returns env value when set",
			key:        "TEST_CMD_VAR_1",
			defaultVal: "default",
			envValue:   "custom",
			setEnv:     true,
			expected:   "custom",
		},
		{
			name:       "returns default when not set",
			key:        "TEST_CMD_VAR_2",
			defaultVal: "default",
			setEnv:     false,
			expected:   "default",
		},
		{
			name:       "returns default when empty",
			key:        "TEST_CMD_VAR_3",
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
