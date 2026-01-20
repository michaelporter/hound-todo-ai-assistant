package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost:5432/todos")
	os.Setenv("GRPC_PORT", "50052")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("GRPC_PORT")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://localhost:5432/todos" {
		t.Errorf("expected DatabaseURL postgres://localhost:5432/todos, got %s", cfg.DatabaseURL)
	}
	if cfg.GRPCPort != "50052" {
		t.Errorf("expected GRPCPort 50052, got %s", cfg.GRPCPort)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	os.Unsetenv("DATABASE_URL")

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when DATABASE_URL is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}

func TestLoad_DefaultGRPCPort(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost:5432/todos")
	os.Unsetenv("GRPC_PORT")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GRPCPort != "50051" {
		t.Errorf("expected default GRPCPort 50051, got %s", cfg.GRPCPort)
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
			key:        "TEST_DOMAIN_VAR_1",
			defaultVal: "default",
			envValue:   "custom",
			setEnv:     true,
			expected:   "custom",
		},
		{
			name:       "returns default when not set",
			key:        "TEST_DOMAIN_VAR_2",
			defaultVal: "default",
			setEnv:     false,
			expected:   "default",
		},
		{
			name:       "returns default when empty",
			key:        "TEST_DOMAIN_VAR_3",
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
