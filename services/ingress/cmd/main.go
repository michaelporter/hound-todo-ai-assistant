package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hound-todo/services/ingress/internal/config"
	"hound-todo/services/ingress/internal/handler"
	"hound-todo/services/ingress/internal/publisher"
	"hound-todo/shared/logging"
)

func main() {
	logger := logging.New("ingress")
	logger.Info("Starting ingress-svc...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(err)
	}

	// Initialize RabbitMQ publisher
	pub, err := publisher.New(cfg.RabbitMQURL)
	if err != nil {
		logger.Fatal(err)
	}
	defer pub.Close()
	logger.Info("Connected to RabbitMQ")

	// Create webhook handler
	webhookHandler := handler.NewWebhookHandler(pub, logger, cfg.TwilioAuthToken)

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/webhooks/sms", webhookHandler.HandleSMS)
	mux.HandleFunc("/health", webhookHandler.HandleHealth)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server listening on port %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped")
}
