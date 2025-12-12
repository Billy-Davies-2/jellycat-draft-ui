# Jellycat Fantasy Draft Microservice

A realtime fantasy draft microservice for Jellycat plush toys, built with **Go**, **htmx**, **gRPC**, **Postgres**, **NATS JetStream**, and **ClickHouse**.

## Architecture

This is a **microservice** designed to integrate with a larger ecosystem:

- **Database**: PostgreSQL for persistent storage
- **Pub/Sub**: NATS with JetStream for event streaming
- **Analytics**: ClickHouse for cuddle points calculation
- **APIs**: Dual interface (HTTP REST + gRPC) for flexibility
- **Local Development**: Automatic mocks for all external services

## Stack

- **Backend**: Go 1.25+ with standard library HTTP server
- **Database**: PostgreSQL (mock: SQLite for local dev)
- **Pub/Sub**: NATS JetStream (mock: in-memory for local dev)
- **Analytics**: ClickHouse (mock: static data for local dev)
- **Frontend**: htmx for dynamic interactions, Server-Sent Events (SSE) for realtime updates
- **API**: gRPC with Protocol Buffers for type-safe programmatic access
- **Styling**: TailwindCSS for modern, responsive design

## Features

- 🧸 Draft cute Jellycat plush toys in a fantasy draft format
- 👥 Team creation and management
- 💬 Live chat with emoji reactions
- 📊 Real-time updates via SSE (HTTP) and gRPC streaming
- 🎨 Beautiful UI with TailwindCSS
- 🔌 Dual API: HTTP REST + gRPC
- 🧪 Comprehensive fuzz testing for both interfaces
- 📈 Cuddle points synced from ClickHouse analytics

## Quick Start

### Prerequisites

- Go 1.25 or higher
- (Optional for production) PostgreSQL, NATS, ClickHouse

### Local Development (with Mocks)

```bash
# Clone the repository
git clone https://github.com/Billy-Davies-2/jellycat-draft-ui.git
cd jellycat-draft-ui

# Build the application
make build
# or
go build -o jellycat-draft main.go

# Run with mocks (default for local dev)
ENVIRONMENT=development ./jellycat-draft

# Access the application
# HTTP UI: http://localhost:3000
# gRPC API: localhost:50051
```

**Local development automatically uses mocks:**
- ✅ SQLite instead of Postgres
- ✅ In-memory pub/sub instead of NATS
- ✅ Mock ClickHouse with static cuddle points

### Production Deployment

```bash
# Set environment to production
export ENVIRONMENT=production

# Configure PostgreSQL
export DATABASE_URL="postgres://user:pass@localhost/jellycatdraft?sslmode=disable"

# Configure NATS JetStream
export NATS_URL="nats://localhost:4222"
export NATS_SUBJECT="draft.events"

# Configure ClickHouse
export CLICKHOUSE_ADDR="localhost:9000"
export CLICKHOUSE_DB="analytics"
export CLICKHOUSE_USER="default"
export CLICKHOUSE_PASSWORD="secret"

# Run the microservice
./jellycat-draft
```

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ENVIRONMENT` | Environment mode (`development`, `production`) | `development` | No |
| `PORT` | HTTP server port | `3000` | No |
| `GRPC_PORT` | gRPC server port | `50051` | No |
| `LOG_LEVEL` | Logging level (`debug`, `info`, `warn`, `error`) | `info` | No |
| **PostgreSQL** ||||
| `DATABASE_URL` | PostgreSQL connection string | - | Yes (prod) |
| **NATS JetStream** ||||
| `NATS_URL` | NATS server URL | `nats://localhost:4222` | Yes (prod) |
| `NATS_SUBJECT` | JetStream subject for events | `draft.events` | No |
| **ClickHouse** ||||
| `CLICKHOUSE_ADDR` | ClickHouse server address | `localhost:9000` | Yes (prod) |
| `CLICKHOUSE_DB` | ClickHouse database name | `default` | No |
| `CLICKHOUSE_USER` | ClickHouse username | `default` | No |
| `CLICKHOUSE_PASSWORD` | ClickHouse password | - | No |

## Project Structure

```
.
├── main.go                 # Application entry point (HTTP + gRPC servers)
├── proto/                  # Protocol Buffer definitions
│   ├── draft.proto         # Service and message definitions
│   ├── draft.pb.go         # Generated Go protobuf code
│   └── draft_grpc.pb.go    # Generated gRPC server/client code
├── internal/
│   ├── dal/               # Data Access Layer
│   │   ├── types.go       # DAL interface
│   │   ├── memory.go      # In-memory implementation
│   │   ├── sqlite.go      # SQLite implementation (mock)
│   │   └── postgres.go    # PostgreSQL implementation (production)
│   ├── pubsub/            # Pub/Sub implementations
│   │   ├── pubsub.go      # In-memory pub/sub (mock)
│   │   └── nats.go        # NATS JetStream (production)
│   ├── clickhouse/        # ClickHouse integration
│   │   └── client.go      # ClickHouse client for cuddle points
│   ├── mocks/             # Mock implementations for local dev
│   │   ├── postgres.go    # Mock Postgres (uses SQLite)
│   │   ├── nats.go        # Mock NATS (uses in-memory)
│   │   └── clickhouse.go  # Mock ClickHouse (static data)
│   ├── grpc/              # gRPC server implementation
│   │   └── server.go      # DraftService implementation
│   ├── handlers/          # HTTP handlers
│   ├── models/            # Data models
│   └── fuzz/              # Fuzz tests
│       ├── http_fuzz_test.go   # HTTP endpoint fuzz tests
│       └── grpc_fuzz_test.go   # gRPC endpoint fuzz tests
├── templates/             # HTML templates
└── static/               # Static assets
```

## Testing

### Run All Tests
```bash
make test
# or
go test ./...
```

### Fuzz Testing
The application includes comprehensive fuzz tests for both HTTP and gRPC endpoints:

```bash
# Fuzz test all HTTP endpoints (30s each)
make fuzz-http

# Fuzz test all gRPC endpoints (30s each)
make fuzz-grpc

# Run all fuzz tests
make fuzz-test

# Custom fuzz time (e.g., 5 minutes)
FUZZTIME=5m make fuzz-test
```

## Docker Deployment

```bash
# Build the scratch-based image
docker build -t jellycat-draft .

# Run with environment variables
docker run -p 3000:3000 -p 50051:50051 \
  -e ENVIRONMENT=production \
  -e DATABASE_URL="postgres://..." \
  -e NATS_URL="nats://..." \
  -e CLICKHOUSE_ADDR="..." \
  jellycat-draft
```

## API Endpoints

### HTTP/REST API

#### Draft Operations
- `GET /api/draft/state` - Get current draft state
- `POST /api/draft/pick` - Draft a player
- `POST /api/draft/reset` - Reset the draft

#### Team Operations
- `GET /api/teams` - List all teams
- `POST /api/teams/add` - Create a new team
- `POST /api/teams/reorder` - Reorder teams

#### Player Operations
- `POST /api/players/add` - Add a new player
- `POST /api/players/points` - Update player points
- `GET /api/players/profile` - Get player profile

#### Chat Operations
- `GET /api/chat/list` - Get all chat messages
- `POST /api/chat/send` - Send a chat message
- `POST /api/chat/react` - Add a reaction to a message

#### Realtime
- `GET /api/events` - Server-Sent Events stream for live updates

### gRPC API

The gRPC service provides the same functionality with type-safe interfaces:

- `GetState()` - Get current draft state
- `DraftPlayer()` - Draft a player to a team
- `ResetDraft()` - Reset the draft
- `AddTeam()` - Create a new team
- `ListTeams()` - List all teams
- `ReorderTeams()` - Reorder teams
- `AddPlayer()` - Add a new player
- `SetPlayerPoints()` - Update player points
- `GetPlayerProfile()` - Get player profile with metrics
- `ListChat()` - Get all chat messages
- `SendChatMessage()` - Send a chat message
- `AddReaction()` - Add reaction to a message
- `StreamEvents()` - Stream realtime events (replaces SSE for gRPC clients)

See `proto/draft.proto` for complete API definitions.

## Integration with External Services

### PostgreSQL Schema

The microservice automatically creates the required schema:
- `players` - Jellycat players with stats
- `teams` - Draft teams
- `team_players` - Drafted players per team
- `chat` - Chat messages with reactions

### NATS JetStream

Events published to NATS:
- `draft:pick` - Player drafted
- `draft:reset` - Draft reset
- `teams:add` - Team added
- `teams:reorder` - Teams reordered
- `players:add` - Player added
- `players:updatePoints` - Points updated
- `chat:add` - Chat message sent
- `chat:react` - Reaction added

### ClickHouse Analytics

Queries cuddle points from:
```sql
SELECT 
  jellycat_id,
  toInt32(
    countDistinct(user_id) * 10 +
    count() / 10 +
    sum(duration) / 60
  ) as cuddle_points
FROM jellycat_interactions
WHERE timestamp >= now() - INTERVAL 30 DAY
GROUP BY jellycat_id
```

Points are synced every 5 minutes automatically.

## Logging

The microservice uses **structured logging** with `slog` for all components. Logs are output in JSON format to stdout for easy integration with log aggregation systems.

### Configuration

Configure the logging level using the `LOG_LEVEL` environment variable:

```bash
# Production: minimal logging (default)
export LOG_LEVEL=info

# Development: verbose logging
export LOG_LEVEL=debug

# Only warnings and errors
export LOG_LEVEL=warn

# Only errors
export LOG_LEVEL=error
```

### Log Levels

- **`debug`**: Detailed information for debugging, including:
  - HTTP request/response details
  - gRPC method calls
  - Database query information
  - Pub/sub event details
  - Internal component state changes

- **`info`**: General informational messages (default for production):
  - Service startup and configuration
  - Database connections
  - Major operations (draft picks, team additions)
  - Periodic sync operations

- **`warn`**: Warning messages for non-critical issues:
  - Slow subscribers in pub/sub
  - Degraded health check status
  - Retry operations

- **`error`**: Error messages requiring attention:
  - Failed database operations
  - gRPC/HTTP handler errors
  - External service failures

### Example Usage

```bash
# Development with debug logging
LOG_LEVEL=debug ENVIRONMENT=development ./jellycat-draft

# Production with info logging (default)
LOG_LEVEL=info ENVIRONMENT=production \
  DATABASE_URL="postgres://..." \
  NATS_URL="nats://..." \
  ./jellycat-draft

# Production with minimal logging
LOG_LEVEL=error ENVIRONMENT=production ./jellycat-draft
```

### Log Format

All logs are output in JSON format with structured fields:

```json
{"time":"2024-01-15T10:30:45Z","level":"INFO","msg":"Server starting","address":"0.0.0.0:3000"}
{"time":"2024-01-15T10:30:46Z","level":"DEBUG","msg":"gRPC: Getting draft state"}
{"time":"2024-01-15T10:30:47Z","level":"INFO","msg":"Drafting player","player_id":"1","team_id":"team-1"}
{"time":"2024-01-15T10:30:48Z","level":"ERROR","msg":"Failed to draft player","error":"player already drafted","player_id":"1"}
```

## Development with Mocks

For local development, set `ENVIRONMENT=development`:

```bash
export ENVIRONMENT=development
./jellycat-draft
```

This automatically uses:
- **SQLite** instead of Postgres (file: `dev.sqlite`)
- **In-memory pub/sub** instead of NATS
- **Mock ClickHouse** with static cuddle points

No external services required!

## License

MIT
