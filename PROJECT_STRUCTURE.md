# Project Structure Overview

Generated: 2026-01-17

## Directory Tree

```
todo-app/
├── ARCHITECTURE.md              # System architecture documentation
├── README.md                    # Project overview and getting started
├── Makefile                     # Build automation
├── go.work                      # Go workspace configuration
├── .gitignore                   # Git ignore rules
│
├── services/                    # Microservices
│   ├── ingress/
│   │   ├── cmd/
│   │   │   └── main.go         # Service entrypoint
│   │   ├── internal/           # Private packages (compiler-enforced)
│   │   │   ├── handler/        # HTTP handlers for Twilio webhooks
│   │   │   ├── publisher/      # RabbitMQ message publishing
│   │   │   └── config/         # Configuration management
│   │   └── go.mod
│   │
│   ├── transcribe/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── whisper/        # Whisper AI integration
│   │   │   ├── rabbitmq/       # Queue consumer
│   │   │   └── grpc/           # gRPC client for todo-domain
│   │   └── go.mod
│   │
│   ├── command/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── parser/         # SMS command parsing
│   │   │   └── grpc/           # gRPC client for todo-domain
│   │   └── go.mod
│   │
│   ├── todo-domain/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── domain/         # Business entities
│   │   │   ├── saga/           # Saga orchestration logic
│   │   │   ├── repository/     # Database access layer
│   │   │   └── grpc/           # gRPC server implementation
│   │   └── go.mod
│   │
│   └── notifier/
│       ├── cmd/main.go
│       ├── internal/
│       │   ├── scheduler/      # Cron job scheduling
│       │   ├── grpc/           # gRPC client for todo-domain
│       │   └── twilio/         # Twilio API integration
│       └── go.mod
│
├── shared/                      # Shared libraries (public)
│   ├── idempotency/
│   │   └── key.go              # Idempotency key generation
│   ├── logging/
│   │   └── logger.go           # Structured logging (placeholder for zap/zerolog)
│   ├── errors/
│   │   └── errors.go           # Common error types
│   ├── tracing/                # (placeholder for OpenTelemetry)
│   └── go.mod
│
├── proto/                       # Protocol Buffer definitions (source)
│   └── todo/v1/
│       └── todo.proto          # Service and message definitions
│
├── api/                         # Generated protobuf code (committed)
│   └── todo/v1/
│       ├── todo.pb.go          # Generated (run 'make proto')
│       ├── todo_grpc.pb.go     # Generated (run 'make proto')
│       ├── placeholder.go       # Temporary until code is generated
│       └── go.mod
│
├── infra/                       # Infrastructure as Code
│   ├── terraform/              # Terraform configurations
│   └── k8s/                    # Kubernetes manifests
│
└── scripts/                     # Build and deployment scripts
```

## Go Modules

Each service is an independent Go module:
- `services/ingress/go.mod`
- `services/transcribe/go.mod`
- `services/command/go.mod`
- `services/todo-domain/go.mod`
- `services/notifier/go.mod`
- `shared/go.mod`
- `api/todo/v1/go.mod`

The `go.work` file at the root ties them together for local development.

## Key Files

| File | Purpose |
|------|---------|
| `go.work` | Go workspace - enables local module resolution without replace directives |
| `Makefile` | Build automation - proto generation, building, testing |
| `proto/todo/v1/todo.proto` | Source of truth for gRPC service contracts |
| `shared/` | Code shared across services (logging, errors, etc.) |
| `services/*/internal/` | Service-private code (compiler prevents cross-service imports) |

## Next Steps

1. **Generate protobuf code**: `make proto`
2. **Verify builds**: `make build`
3. **Start implementing**: Begin with `ingress-svc` (simplest service)

## Design Decisions

### Why separate Go modules?
- Independent dependency management
- Clear boundaries between services
- Smaller dependency graphs

### Why commit generated protobuf code?
- Services can build without protoc installed
- CI/CD simplicity
- Code review visibility for API changes

### Why internal/ directories?
- Compiler-enforced encapsulation
- Prevents accidental coupling between services
- Go language feature, not just convention
