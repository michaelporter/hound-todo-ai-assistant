package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"hound-todo/services/notifier/internal/config"
	"hound-todo/services/notifier/internal/consumer"
	"hound-todo/services/notifier/internal/twilio"
	"hound-todo/shared/logging"
)

func main() {
	logger := logging.New("notifier")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Create Twilio client
	twilioClient := twilio.NewClient(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioPhoneNumber)
	logger.Info("Twilio client initialized")

	// Create RabbitMQ consumer
	cons, err := consumer.New(cfg.RabbitMQURL, logger)
	if err != nil {
		logger.Error("Failed to create consumer: %v", err)
		os.Exit(1)
	}
	defer cons.Close()
	logger.Info("Connected to RabbitMQ")

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		logger.Info("Shutting down...")
		cancel()
	}()

	logger.Info("Starting notifier-svc, waiting for messages...")

	// Start consuming messages and sending SMS
	if err := cons.Start(ctx, func(ctx context.Context, msg *consumer.SMSReply) error {
		return twilioClient.SendSMS(ctx, msg.UserID, msg.Message)
	}); err != nil && err != context.Canceled {
		logger.Error("Consumer error: %v", err)
		os.Exit(1)
	}

	logger.Info("Notifier-svc stopped")
}
