.PHONY: all proto build test lint clean help
.PHONY: up down logs rebuild ps shell db-shell rabbitmq-ui

# Default target
all: proto build

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Development (Native):"
	@echo "    proto        - Generate protobuf code"
	@echo "    build        - Build all services"
	@echo "    build-svc    - Build specific service (e.g., make build-svc SVC=ingress)"
	@echo "    test         - Run tests"
	@echo "    lint         - Run linter"
	@echo "    clean        - Clean build artifacts"
	@echo "    run-svc      - Run specific service (e.g., make run-svc SVC=ingress)"
	@echo ""
	@echo "  Docker:"
	@echo "    up           - Start all services with Docker Compose"
	@echo "    up-infra     - Start only infrastructure (Postgres, RabbitMQ)"
	@echo "    down         - Stop all services"
	@echo "    logs         - Tail logs for all services"
	@echo "    logs-svc     - Tail logs for specific service (e.g., make logs-svc SVC=ingress)"
	@echo "    rebuild      - Rebuild and restart all services"
	@echo "    rebuild-svc  - Rebuild specific service (e.g., make rebuild-svc SVC=ingress)"
	@echo "    ps           - Show running containers"
	@echo "    shell        - Open shell in service container (e.g., make shell SVC=ingress)"
	@echo "    db-shell     - Open PostgreSQL shell"
	@echo "    rabbitmq-ui  - Print RabbitMQ management UI URL"

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@mkdir -p api/todo/v1
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/todo/v1/todo.proto
	@echo "✓ Protobuf code generated"

# Build all services
build:
	@echo "Building all services..."
	@mkdir -p bin
	@cd services/ingress && go build -o ../../bin/ingress ./cmd
	@cd services/transcribe && go build -o ../../bin/transcribe ./cmd
	@cd services/command && go build -o ../../bin/command ./cmd
	@cd services/todo-domain && go build -o ../../bin/todo-domain ./cmd
	@cd services/notifier && go build -o ../../bin/notifier ./cmd
	@echo "✓ All services built successfully"

# Build specific service
build-svc:
	@echo "Building $(SVC)..."
	@mkdir -p bin
	@cd services/$(SVC) && go build -o ../../bin/$(SVC) ./cmd
	@echo "✓ $(SVC) built successfully"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run ./... || echo "golangci-lint not installed. Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f api/todo/v1/*.pb.go
	@echo "✓ Cleaned"

# Run specific service locally
run-svc:
	@echo "Running $(SVC)..."
	@cd services/$(SVC) && go run ./cmd

# =============================================================================
# Docker Commands
# =============================================================================

# Start all services
up:
	@echo "Starting all services..."
	@docker compose up -d
	@echo "✓ Services started. Run 'make logs' to view output."
	@echo "  RabbitMQ UI: http://localhost:15672 (hound/hound_dev)"

# Start only infrastructure (useful for native development)
up-infra:
	@echo "Starting infrastructure services..."
	@docker compose up -d postgres rabbitmq
	@echo "✓ Infrastructure started"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  RabbitMQ:   localhost:5672 (AMQP), localhost:15672 (UI)"

# Stop all services
down:
	@echo "Stopping all services..."
	@docker compose down
	@echo "✓ Services stopped"

# Stop and remove volumes (full reset)
down-clean:
	@echo "Stopping services and removing volumes..."
	@docker compose down -v
	@echo "✓ Services stopped and volumes removed"

# Tail logs for all services
logs:
	@docker compose logs -f

# Tail logs for specific service
logs-svc:
	@docker compose logs -f $(SVC)

# Rebuild and restart all services
rebuild:
	@echo "Rebuilding all services..."
	@docker compose up -d --build
	@echo "✓ Services rebuilt and started"

# Rebuild specific service
rebuild-svc:
	@echo "Rebuilding $(SVC)..."
	@docker compose up -d --build $(SVC)
	@echo "✓ $(SVC) rebuilt and started"

# Show running containers
ps:
	@docker compose ps

# Open shell in service container
shell:
	@docker compose exec $(SVC) sh

# Open PostgreSQL shell
db-shell:
	@docker compose exec postgres psql -U hound -d todo_db

# Print RabbitMQ management UI URL
rabbitmq-ui:
	@echo "RabbitMQ Management UI: http://localhost:15672"
	@echo "Username: hound"
	@echo "Password: hound_dev"

# Create .env from example if it doesn't exist
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✓ Created .env from .env.example"; \
		echo "  Please edit .env with your Twilio and OpenAI credentials"; \
	else \
		echo ".env already exists"; \
	fi
