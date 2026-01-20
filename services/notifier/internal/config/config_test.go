package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set all required env vars
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("TWILIO_ACCOUNT_SID", "AC123456")
	os.Setenv("TWILIO_AUTH_TOKEN", "test-token")
	os.Setenv("TWILIO_PHONE_NUMBER", "+15551234567")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("TWILIO_ACCOUNT_SID")
		os.Unsetenv("TWILIO_AUTH_TOKEN")
		os.Unsetenv("TWILIO_PHONE_NUMBER")
	}()

	cfg, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RabbitMQURL != "amqp://localhost:5672/" {
		t.Errorf("expected RabbitMQURL amqp://localhost:5672/, got %s", cfg.RabbitMQURL)
	}
	if cfg.TwilioAccountSID != "AC123456" {
		t.Errorf("expected TwilioAccountSID AC123456, got %s", cfg.TwilioAccountSID)
	}
	if cfg.TwilioAuthToken != "test-token" {
		t.Errorf("expected TwilioAuthToken test-token, got %s", cfg.TwilioAuthToken)
	}
	if cfg.TwilioPhoneNumber != "+15551234567" {
		t.Errorf("expected TwilioPhoneNumber +15551234567, got %s", cfg.TwilioPhoneNumber)
	}
}

func TestLoad_MissingRabbitMQURL(t *testing.T) {
	os.Unsetenv("RABBITMQ_URL")
	os.Setenv("TWILIO_ACCOUNT_SID", "AC123456")
	os.Setenv("TWILIO_AUTH_TOKEN", "test-token")
	os.Setenv("TWILIO_PHONE_NUMBER", "+15551234567")
	defer func() {
		os.Unsetenv("TWILIO_ACCOUNT_SID")
		os.Unsetenv("TWILIO_AUTH_TOKEN")
		os.Unsetenv("TWILIO_PHONE_NUMBER")
	}()

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when RABBITMQ_URL is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}

func TestLoad_MissingTwilioAccountSID(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Unsetenv("TWILIO_ACCOUNT_SID")
	os.Setenv("TWILIO_AUTH_TOKEN", "test-token")
	os.Setenv("TWILIO_PHONE_NUMBER", "+15551234567")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("TWILIO_AUTH_TOKEN")
		os.Unsetenv("TWILIO_PHONE_NUMBER")
	}()

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when TWILIO_ACCOUNT_SID is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}

func TestLoad_MissingTwilioAuthToken(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("TWILIO_ACCOUNT_SID", "AC123456")
	os.Unsetenv("TWILIO_AUTH_TOKEN")
	os.Setenv("TWILIO_PHONE_NUMBER", "+15551234567")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("TWILIO_ACCOUNT_SID")
		os.Unsetenv("TWILIO_PHONE_NUMBER")
	}()

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when TWILIO_AUTH_TOKEN is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}

func TestLoad_MissingTwilioPhoneNumber(t *testing.T) {
	os.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	os.Setenv("TWILIO_ACCOUNT_SID", "AC123456")
	os.Setenv("TWILIO_AUTH_TOKEN", "test-token")
	os.Unsetenv("TWILIO_PHONE_NUMBER")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("TWILIO_ACCOUNT_SID")
		os.Unsetenv("TWILIO_AUTH_TOKEN")
	}()

	cfg, err := Load()

	if err == nil {
		t.Fatal("expected error when TWILIO_PHONE_NUMBER is missing")
	}
	if cfg != nil {
		t.Error("expected nil config when error occurs")
	}
}
