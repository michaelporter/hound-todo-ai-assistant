package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"hound-todo/services/command/internal/ai"
	"hound-todo/services/command/internal/config"
	"hound-todo/services/command/internal/consumer"
	"hound-todo/services/command/internal/domain"
	"hound-todo/services/command/internal/handler"
	"hound-todo/shared/logging"
)

func main() {
	logger := logging.New("command")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Create OpenAI client
	aiClient := ai.NewClient(cfg.OpenAIAPIKey)
	if cfg.OpenAIModel != "" {
		aiClient.SetModel(cfg.OpenAIModel)
		logger.Info("Using OpenAI model: %s", cfg.OpenAIModel)
	}

	// Connect to todo-domain-svc via gRPC
	domainClient, err := domain.NewClient(cfg.TodoDomainAddr)
	if err != nil {
		logger.Error("Failed to connect to todo-domain: %v", err)
		os.Exit(1)
	}
	defer domainClient.Close()
	logger.Info("Connected to todo-domain at %s", cfg.TodoDomainAddr)

	// Create RabbitMQ consumer
	cons, err := consumer.New(cfg.RabbitMQURL, logger)
	if err != nil {
		logger.Error("Failed to create consumer: %v", err)
		os.Exit(1)
	}
	defer cons.Close()
	logger.Info("Connected to RabbitMQ")

	// Create command handler
	h := handler.New(aiClient, domainClient, logger)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		logger.Info("Shutting down...")
		cancel()
	}()

	logger.Info("Starting command-svc, waiting for messages...")

	// Start consuming messages
	if err := cons.Start(ctx, h.Handle); err != nil && err != context.Canceled {
		logger.Error("Consumer error: %v", err)
		os.Exit(1)
	}

	logger.Info("Command-svc stopped")
}
