.PHONY: all proto build test lint clean help

# Default target
all: proto build

# Help target
help:
	@echo "Available targets:"
	@echo "  proto        - Generate protobuf code"
	@echo "  build        - Build all services"
	@echo "  build-svc    - Build specific service (e.g., make build-svc SVC=ingress)"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-svc      - Run specific service (e.g., make run-svc SVC=ingress)"

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
