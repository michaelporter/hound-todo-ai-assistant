# Todo App - SMS-Based Microservices

A learning project exploring production-grade distributed systems patterns through an SMS-based todo application.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed system design.

**Key Components:**
- **ingress-svc**: Twilio webhook receiver
- **transcribe-svc**: Whisper AI voice transcription
- **command-svc**: SMS text command parser
- **todo-domain-svc**: Core business logic with saga orchestration
- **notifier-svc**: Scheduled notifications and daily summaries

**Technology Stack:**
- Go 1.21+
- gRPC + Protocol Buffers
- RabbitMQ (message queue)
- PostgreSQL (databases)
- Kubernetes (k3s on Raspberry Pi)
- Terraform (infrastructure)

## Project Structure

```
todo-app/
├── services/          # Microservices
│   ├── ingress/
│   ├── transcribe/
│   ├── command/
│   ├── todo-domain/
│   └── notifier/
├── shared/           # Shared libraries
├── api/              # Generated protobuf code
├── proto/            # Protobuf definitions
├── infra/            # Infrastructure as Code
└── scripts/          # Build/deploy scripts
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (`protoc`)
- Docker (for local development)
- Make

### Install protoc and Go plugins

```bash
# macOS
brew install protobuf

# Linux
sudo apt-get install -y protobuf-compiler

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Build

```bash
# Generate protobuf code
make proto

# Build all services
make build

# Build specific service
make build-svc SVC=ingress
```

### Run Locally

```bash
# Run a specific service
make run-svc SVC=ingress

# Or directly
cd services/ingress && go run ./cmd
```

### Test

```bash
make test
```

## Development Workflow

1. **Modify protobuf schemas** in `proto/`
2. **Regenerate code**: `make proto`
3. **Implement service logic** in `services/[service-name]/internal/`
4. **Build**: `make build-svc SVC=service-name`
5. **Test**: `make test`

## Learning Goals

This project is intentionally overengineered for educational purposes:

✅ Microservices architecture patterns  
✅ gRPC and Protocol Buffers  
✅ Message queue patterns (RabbitMQ)  
✅ Saga orchestration and distributed transactions  
✅ Kubernetes deployment on ARM64  
✅ Infrastructure as Code with Terraform  
✅ Observability (metrics, logs, traces)  

## Resources

- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [gRPC Best Practices](https://grpc.io/docs/guides/performance/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)

## License

MIT
