package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	todov1 "hound-todo/api/todo/v1"
	"hound-todo/services/todo-domain/internal/config"
	"hound-todo/services/todo-domain/internal/server"
	"hound-todo/services/todo-domain/internal/store"
	"hound-todo/shared/logging"
)

func main() {
	logger := logging.New("todo-domain")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	db, err := store.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL")

	// Create the store and server
	todoStore := store.New(db)
	todoServer := server.New(todoStore, logger)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register our service implementation
	todov1.RegisterTodoDomainServer(grpcServer, todoServer)

	// Enable reflection for debugging tools like grpcurl
	reflection.Register(grpcServer)

	// Start listening
	listener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("Failed to listen: %v", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	logger.Info("gRPC server listening on port %s", cfg.GRPCPort)

	// Start serving (blocks until shutdown)
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("Failed to serve: %v", err)
		os.Exit(1)
	}

	logger.Info("Server stopped")
}
