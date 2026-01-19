# Todo App - SMS-Based Microservices

A learning project exploring production-grade distributed systems patterns and AI workflows through an SMS-based todo application.

The todo functionality includes persistent reminders and helpful prompts to get me started on tasks.

This is being written in heavy collaboration with Claude Code. I have only middling knowledge of all tools in this project, including Golang and am using this project to actively learn in interaction with Claude.

All code changes are reviewed manually - Claude has no permission to automatically make changes.

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
│   ├── ingress/       # Twilio webhook receiver → RabbitMQ
│   ├── transcribe/    # Voice memo transcription
│   ├── command/       # SMS command parser
│   ├── todo-domain/   # Core business logic
│   └── notifier/      # Scheduled notifications
├── shared/            # Shared libraries (logging, errors, idempotency)
├── tools/             # Development utilities
│   └── peek/          # RabbitMQ message inspector
├── api/               # Generated protobuf code
├── proto/             # Protobuf definitions
├── docker/            # Docker configs and Dockerfile
├── infra/             # Infrastructure as Code
└── scripts/           # Build/deploy scripts
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Docker Desktop (with WSL2 integration if on Windows)
- Protocol Buffers compiler (`protoc`) - for modifying protos

### Quick Start with Docker Compose

The easiest way to run everything locally:

```bash
# Start all services with hot reload
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f ingress
```

This starts:
- **ingress** on http://localhost:8080
- **RabbitMQ** on localhost:5672 (Management UI: http://localhost:15672)
- **PostgreSQL** on localhost:5432
- All other services

Default RabbitMQ credentials: `hound` / `hound_dev`

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | 8080 | Ingress server port |
| `RABBITMQ_URL` | (required) | AMQP connection string |
| `TWILIO_AUTH_TOKEN` | (optional) | Enables webhook signature validation |

These are pre-configured in `docker-compose.yml` for local development.

### Build Tools

```bash
# Build the peek tool (RabbitMQ inspector)
cd tools/peek && go build -o ../../bin/peek .

# Build a service manually
cd services/ingress && go build ./cmd
```

### Run Tests

```bash
# Run all tests for a service
cd services/ingress && go test ./...

# Run with verbose output
go test -v ./...
```

## Receiving Real SMS (Twilio Setup)

To test with real SMS messages locally:

### 1. Expose localhost via Tailscale Funnel

```powershell
# On Windows (or wherever Tailscale is installed)
tailscale funnel 8080
```

This gives you a public URL like `https://your-machine.tail1234.ts.net`

### 2. Configure Twilio Webhook

1. Go to [Twilio Console](https://console.twilio.com/) → Phone Numbers → Your Number
2. Under "Messaging Configuration", set:
   - **Webhook URL**: `https://your-machine.tail1234.ts.net/webhooks/sms`
   - **HTTP Method**: POST
3. Save

### 3. Send a test SMS

Text your Twilio number and watch the logs:

```bash
docker compose logs -f ingress
```

## Debugging Tools

### Peek - RabbitMQ Message Inspector

Inspect messages in queues without consuming them:

```bash
# Build the tool
cd tools/peek && go build -o ../../bin/peek .

# List all queues
./bin/peek -list

# Peek at messages
./bin/peek -queue text.commands
./bin/peek -queue audio.processing
```

### RabbitMQ Management UI

http://localhost:15672 (login: `hound` / `hound_dev`)

- View queue depths
- Inspect and purge messages
- Monitor connections

### Service Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f ingress

# Last 50 lines
docker compose logs --tail=50 ingress
```

## Development Workflow

1. **Start services**: `docker compose up -d`
2. **Modify code** - hot reload picks up changes automatically
3. **Check logs**: `docker compose logs -f [service]`
4. **Inspect messages**: `./bin/peek -queue text.commands`
5. **Run tests**: `cd services/[name] && go test ./...`

## Modifying Protobufs

If you need to change the gRPC service definitions:

```bash
# Install protoc (macOS)
brew install protobuf

# Install protoc (Linux)
sudo apt-get install -y protobuf-compiler

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Regenerate code after modifying .proto files
make proto
```

## Learning Goals

This project is intentionally overengineered for educational purposes:

- Microservices architecture patterns
- gRPC and Protocol Buffers
- Message queue patterns (RabbitMQ)
- Saga orchestration and distributed transactions
- Kubernetes deployment on ARM64
- Infrastructure as Code with Terraform
- Observability (metrics, logs, traces)

## Resources

- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [gRPC Best Practices](https://grpc.io/docs/guides/performance/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)

## License

MIT
