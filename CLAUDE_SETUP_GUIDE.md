# Claude Code Setup Guide - hound-todo

This file provides context for Claude Code to set up the development environment on a new Linux system.

## Project Overview

SMS-based todo application using microservices architecture. The user is learning Go, gRPC, and distributed systems through this project. All code changes should be reviewed manually.

**Repository**: `git@github.com:michaelporter/hound-todo-ai-assistant.git`

## Prerequisites to Install

### Required

1. **Docker & Docker Compose** - All services run in containers
   ```bash
   # Install Docker
   sudo apt-get update
   sudo apt-get install -y docker.io docker-compose-v2
   sudo usermod -aG docker $USER
   # Log out and back in for group changes
   ```

2. **Go 1.22+** - For running tests and building tools locally
   ```bash
   # Download from https://go.dev/dl/ or use package manager
   # The project uses go.work with Go 1.22.7
   ```

3. **Git** - For version control
   ```bash
   sudo apt-get install -y git
   ```

### Optional (for modifying protobufs)

4. **Protocol Buffers Compiler**
   ```bash
   sudo apt-get install -y protobuf-compiler
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

## Project Structure

```
hound-todo/
├── services/           # Microservices (Go)
│   ├── ingress/        # Twilio webhook receiver → RabbitMQ
│   ├── transcribe/     # Whisper AI voice transcription
│   ├── command/        # SMS command parser (AI-powered)
│   ├── todo-domain/    # Core business logic, gRPC server
│   └── notifier/       # Scheduled notifications
├── shared/             # Shared Go libraries
├── tools/peek/         # RabbitMQ message inspector
├── api/todo/v1/        # Generated protobuf code
├── proto/              # Protobuf definitions
├── docker/             # Docker configs
│   ├── Dockerfile.dev  # Dev dockerfile with Air hot-reload
│   ├── .air.toml       # Air configuration
│   └── postgres/init.sql  # Database initialization
└── docker-compose.yml  # Local dev environment
```

## Environment Setup Steps

### 1. Clone the repository

```bash
git clone git@github.com:michaelporter/hound-todo-ai-assistant.git
cd hound-todo-ai-assistant
```

### 2. Create environment file

```bash
cp .env.example .env
```

Edit `.env` and add API keys if available:
- `OPENAI_API_KEY` - Required for transcription and command parsing
- `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_PHONE_NUMBER` - Required for SMS

The PostgreSQL and RabbitMQ credentials have working defaults for local dev.

### 3. Start all services

```bash
docker compose up -d
```

This starts:
- **PostgreSQL** on localhost:5432 (creates 3 databases: todo_db, audit_db, transcription_db)
- **RabbitMQ** on localhost:5672 (Management UI: http://localhost:15672)
- **ingress** on http://localhost:8080
- **transcribe**, **command**, **todo-domain**, **notifier** services

### 4. Verify services are running

```bash
docker compose ps
docker compose logs -f  # Watch all logs
```

All services should show as "running" or "healthy".

### 5. Build the peek tool (optional but useful)

```bash
cd tools/peek && go build -o ../../bin/peek .
cd ../..
./bin/peek -list  # List RabbitMQ queues
```

## Key Configuration

### Default Credentials (local dev only)

| Service    | Username | Password   |
|------------|----------|------------|
| PostgreSQL | hound    | hound_dev  |
| RabbitMQ   | hound    | hound_dev  |

### Ports

| Service          | Port  |
|------------------|-------|
| ingress (HTTP)   | 8080  |
| todo-domain (gRPC) | 50051 |
| PostgreSQL       | 5432  |
| RabbitMQ AMQP    | 5672  |
| RabbitMQ UI      | 15672 |

## Common Development Tasks

### Run tests for a service

```bash
cd services/ingress && go test ./...
cd services/command && go test ./...
cd services/todo-domain && go test ./...
cd services/notifier && go test ./...
```

### View logs

```bash
docker compose logs -f ingress      # Single service
docker compose logs -f              # All services
docker compose logs --tail=50 command  # Last 50 lines
```

### Restart a service after code changes

Hot reload is enabled via Air, but if needed:
```bash
docker compose restart command
```

### Rebuild containers (after Dockerfile changes)

```bash
docker compose up -d --build
```

### Reset databases

```bash
docker compose down -v  # Removes volumes
docker compose up -d    # Recreates with fresh data
```

### Inspect RabbitMQ messages

```bash
./bin/peek -queue text.commands
./bin/peek -queue audio.processing
```

Or use the Management UI at http://localhost:15672

## Live Debugging Sessions

One of the most effective debugging workflows is having Claude monitor live logs while the user interacts with the product via SMS. This section describes how to conduct these sessions.

### Starting a Debug Session

1. **User announces they're starting a session**: "I'm going to send some SMS messages, watch the logs"

2. **Claude starts monitoring logs**:
   ```bash
   docker compose logs -f --tail=0
   ```
   The `--tail=0` flag starts fresh, showing only new logs from this point forward.

3. **User interacts with the product** via SMS (creating todos, completing them, etc.)

4. **Claude captures observations** in a structured format (see below)

5. **User signals end of session**: "Okay, I'm done"

6. **Claude provides analysis and improvement suggestions**

### Observation Format

During the session, Claude should capture each interaction in this structure:

```
### Interaction N: [Brief description]
- **User Action**: What the user did (e.g., "Sent SMS: 'add buy groceries'")
- **Expected Flow**: ingress → command → todo-domain → notifier
- **Observed Flow**: What actually happened based on logs
- **Result**: SUCCESS / PARTIAL / FAILURE
- **Timing**: Approximate end-to-end latency if observable
- **Notes**: Any anomalies, warnings, or interesting behavior
```

### What to Watch For

**Success indicators:**
- Webhook received by ingress-svc
- Message published to RabbitMQ
- Command parsed correctly by command-svc
- gRPC call to todo-domain-svc completed
- Database operation successful
- Reply SMS sent by notifier-svc

**Failure indicators:**
- Error logs (look for `level=error` or stack traces)
- Timeout messages
- RabbitMQ connection issues
- gRPC call failures
- OpenAI API errors in command-svc
- Twilio API errors in notifier-svc

**Quality signals:**
- AI command parsing confidence (command-svc logs this)
- Unexpected command interpretations
- Slow responses (user experience impact)
- Retries or repeated processing

### Post-Session Analysis

After the session, Claude should provide:

1. **Session Summary**: Overview of interactions and success rate
2. **Issues Identified**: Specific problems observed, with log evidence
3. **Improvement Suggestions**: Prioritized list of fixes or enhancements
4. **Questions**: Any unclear behavior that needs investigation

### Example Session Output

```
## Debug Session Summary - [Date]

### Overview
- Total interactions: 5
- Successful: 3
- Partial/Failed: 2

### Interactions

#### Interaction 1: Add todo
- **User Action**: SMS "remind me to call mom"
- **Observed Flow**: ingress ✓ → command ✓ → todo-domain ✓ → notifier ✓
- **Result**: SUCCESS
- **Notes**: Command parsed as "add todo: call mom" with high confidence (0.94)

#### Interaction 2: List todos (system failure)
- **User Action**: SMS "what's on my list"
- **Observed Flow**: ingress ✓ → command ✓ → todo-domain ✓ → notifier ✗
- **Result**: PARTIAL - system failure
- **Notes**: Todo retrieved but SMS reply failed - Twilio rate limit hit

#### Interaction 3: Complete todo (semantic failure)
- **User Action**: SMS "I finished the laundry"
- **Observed Flow**: ingress ✓ → command ✓ → todo-domain ✓ → notifier ✓
- **Result**: FAILURE - semantic error
- **Notes**: System worked correctly but AI interpreted this as "add todo: finish the laundry"
  instead of completing the existing "do laundry" todo. Confidence was 0.72 (borderline).
  User received confirmation of new todo creation, not completion.

### Issues Identified
1. **Semantic**: Past tense phrases like "I finished X" not recognized as completion commands
2. **System**: Twilio rate limiting when multiple replies sent quickly
3. **UX**: Low-confidence interpretations (< 0.8) proceed without user confirmation

### Suggested Improvements
1. Train command-svc to recognize past tense as completion intent ("I did X", "finished X", "done with X")
2. Add confirmation prompt for low-confidence (<0.8) command interpretations
3. Add exponential backoff retry for Twilio API calls
4. Fuzzy match "finish the laundry" to existing "do laundry" todo
```

## Go Workspace

The project uses Go workspaces (`go.work`). All modules are linked:
- services/ingress, transcribe, command, todo-domain, notifier
- shared (common libraries)
- api/todo/v1 (generated protobuf code)
- tools/peek

## Testing SMS Locally

To receive real SMS messages locally:

1. Install and configure Tailscale
2. Run `tailscale funnel 8080` to expose localhost
3. Configure Twilio webhook URL to `https://your-machine.ts.net/webhooks/sms`

## Troubleshooting

### Docker permission denied

```bash
sudo usermod -aG docker $USER
# Then log out and back in
```

### Port already in use

```bash
sudo lsof -i :8080  # Find what's using the port
docker compose down  # Stop all services
```

### Services failing to start

Check logs for the specific service:
```bash
docker compose logs command
```

Common issues:
- Missing `OPENAI_API_KEY` (command and transcribe services need it)
- RabbitMQ not ready yet (services have health checks but may need a moment)

### Database connection issues

Ensure PostgreSQL is healthy:
```bash
docker compose ps postgres
docker compose logs postgres
```

## Architecture Notes

See `ARCHITECTURE.md` for detailed system design. Key points:
- Services communicate via gRPC (internal) and RabbitMQ (async)
- ingress-svc receives webhooks and publishes to RabbitMQ
- command-svc uses OpenAI to parse natural language commands
- todo-domain-svc is the source of truth for business logic
- notifier-svc handles scheduled notifications and SMS replies
