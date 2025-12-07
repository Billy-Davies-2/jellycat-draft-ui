# Jellycat Fantasy Draft

A realtime fantasy draft application for Jellycat plush toys, built with **Go**, **htmx**, **Alpine.js**, **gRPC**, **TailwindCSS**, and **Authentik authentication**.

## Stack

- **Backend**: Go 1.24+ with dual interface:
  - HTTP/REST server for SSR pages and htmx frontend  
  - gRPC server for programmatic API access
- **Frontend**: 
  - htmx for server-side HTML updates and dynamic interactions
  - Alpine.js for enriched client-side reactivity (search, filtering, UI state)
  - Server-Sent Events (SSE) for realtime updates
- **Authentication**: Authentik OAuth2/OIDC with role-based access control
- **Styling**: TailwindCSS for modern, responsive design
- **Templates**: Go's html/template for server-side rendering
- **Data**: Pluggable DAL supporting memory, SQLite, and PostgreSQL
- **Messaging**: NATS JetStream for distributed pub/sub
- **Analytics**: ClickHouse for cuddle points calculation
- **API**: Protocol Buffers with gRPC streaming for events

## Features

- 🧸 Draft cute Jellycat plush toys in a fantasy draft format
- 👥 Team creation and management
- 💬 Live chat with emoji reactions
- 📊 Real-time updates via SSE (HTTP) and gRPC streaming
- 🔐 Secure authentication via Authentik OAuth2/OIDC
- 👑 Role-based access control (admins can access admin panel)
- 🎨 Beautiful UI with TailwindCSS
- ⚡ Alpine.js for reactive client-side interactions (search, filters, notifications)
- 🗄️ Multiple storage backends (memory, SQLite, PostgreSQL)
- 📡 NATS JetStream for distributed messaging
- 📈 ClickHouse integration for analytics
- 🔌 Dual API: HTTP/REST + gRPC
- 🧪 Comprehensive fuzz testing for both interfaces

## Quick Start

### Prerequisites

- Go 1.24 or higher
- PostgreSQL database (or use memory/SQLite for development)
- NATS server with JetStream
- ClickHouse server
- Authentik OAuth2 provider
- (Optional) Protocol Buffers compiler for regenerating proto files
- (Optional) TailwindCSS CLI for stylesheet changes

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/Billy-Davies-2/jellycat-draft-ui.git
   cd jellycat-draft-ui
   ```

2. **Build the application**
   ```bash
   make build
   # or
   go build -o jellycat-draft main.go
   ```

3. **Configure environment variables**
   ```bash
   # Database (choose one)
   export DB_DRIVER=memory                                    # In-memory (no persistence)
   export DB_DRIVER=sqlite                                    # SQLite (file-based)
   export SQLITE_FILE=dev.sqlite
   export DB_DRIVER=postgres                                  # PostgreSQL (production)
   export DATABASE_URL="postgres://user:pass@localhost/draft"
   
   # NATS JetStream
   export NATS_URL="nats://localhost:4222"
   export NATS_SUBJECT="draft.events"
   
   # ClickHouse
   export CLICKHOUSE_ADDR="localhost:9000"
   export CLICKHOUSE_DB="default"
   export CLICKHOUSE_USER="default"
   export CLICKHOUSE_PASSWORD=""
   
   # Authentik OAuth2
   export AUTHENTIK_BASE_URL="https://auth.yourdomain.com"
   export AUTHENTIK_CLIENT_ID="your-client-id"
   export AUTHENTIK_CLIENT_SECRET="your-client-secret"
   export AUTHENTIK_REDIRECT_URL="http://localhost:3000/auth/callback"
   ```

4. **Run the server**
   ```bash
   ./jellycat-draft
   ```

5. **Access the application**
   - HTTP UI: `http://localhost:3000`
   - gRPC API: `localhost:50051`

## Role-Based Access Control

The application implements role-based access control using Authentik groups:

- **Users**: All authenticated users can access the draft and team management features
- **Admins**: Users in the `admins` group can access the admin panel at `/admin`
  - Add new Jellycat players
  - Manage player points
  - Reset the draft
  - View all team data

To grant admin access, add users to the `admins` group in your Authentik configuration.

## Testing

The application includes comprehensive testing with mock implementations:

```bash
# Run all tests
make test

# Fuzz test HTTP endpoints (30s each)
make fuzz-http

# Fuzz test gRPC endpoints (30s each)
make fuzz-grpc

# Custom fuzz duration
FUZZTIME=5m make fuzz-test
```

### Mock Implementations for Testing

Mock implementations are provided in `internal/mocks/` for use in tests only:
- `MockPostgresDAL`: SQLite-based mock for PostgreSQL
- `MockNATSPubSub`: In-memory pub/sub for NATS
- `MockClickHouseClient`: Static data for ClickHouse

These mocks should **only** be imported and used in test files (`*_test.go`).

## Project Structure

```
.
├── main.go                     # Application entry point (HTTP + gRPC servers)
├── proto/                      # Protocol Buffer definitions
│   ├── draft.proto             # Service and message definitions
│   ├── draft.pb.go             # Generated Go protobuf code
│   └── draft_grpc.pb.go        # Generated gRPC server/client code
├── internal/
│   ├── auth/                   # Authentication
│   │   └── authentik.go        # Authentik OAuth2 implementation
│   ├── dal/                    # Data Access Layer
│   │   ├── types.go            # DAL interface
│   │   ├── memory.go           # In-memory implementation
│   │   ├── sqlite.go           # SQLite implementation
│   │   └── postgres.go         # PostgreSQL implementation
│   ├── pubsub/                 # Pub/Sub implementations
│   │   ├── pubsub.go           # In-memory pub/sub
│   │   └── nats.go             # NATS JetStream client
│   ├── clickhouse/             # ClickHouse integration
│   │   └── client.go           # Cuddle points analytics
│   ├── mocks/                  # Mock implementations (testing only)
│   │   ├── postgres.go         # Mock PostgreSQL (SQLite wrapper)
│   │   ├── nats.go             # Mock NATS (in-memory)
│   │   └── clickhouse.go       # Mock ClickHouse (static data)
│   ├── grpc/                   # gRPC server implementation
│   │   └── server.go           # DraftService implementation
│   ├── handlers/               # HTTP handlers
│   ├── models/                 # Data models
│   └── fuzz/                   # Fuzz tests
│       ├── http_fuzz_test.go   # HTTP endpoint fuzz tests
│       └── grpc_fuzz_test.go   # gRPC endpoint fuzz tests
├── templates/                  # HTML templates
│   ├── base.html               # Base layout with Alpine.js
│   ├── start.html              # Team creation page
│   ├── draft.html              # Main draft page
│   └── admin.html              # Admin panel (requires admin role)
└── static/                     # Static assets
    ├── css/                    # TailwindCSS stylesheets
    └── images/                 # Jellycat images (18 plushies)
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

Individual fuzz tests:
```bash
# HTTP endpoints
go test -fuzz=FuzzHTTPDraftPick -fuzztime=30s ./internal/fuzz
go test -fuzz=FuzzHTTPAddTeam -fuzztime=30s ./internal/fuzz
go test -fuzz=FuzzHTTPSendChat -fuzztime=30s ./internal/fuzz

# gRPC endpoints
go test -fuzz=FuzzGRPCDraftPlayer -fuzztime=30s ./internal/fuzz
go test -fuzz=FuzzGRPCAddTeam -fuzztime=30s ./internal/fuzz
go test -fuzz=FuzzGRPCSendChatMessage -fuzztime=30s ./internal/fuzz
```

## Docker Deployment

The application uses a **scratch-based** Docker image for minimal size and maximum security.

### Build and Run

```bash
# Build the image
docker build -t jellycat-draft .

# Run with memory storage (default, no volume needed)
docker run -p 3000:3000 -p 50051:50051 jellycat-draft

# Run with SQLite (requires volume mount)
docker run -p 3000:3000 -p 50051:50051 \
  -e DB_DRIVER=sqlite \
  -e SQLITE_FILE=/data/draft.sqlite \
  -v $(pwd)/data:/data \
  jellycat-draft
```

### Image Details

- **Base**: `scratch` (empty image, ~0 MB overhead)
- **Binary**: Statically linked (no runtime dependencies)
- **Total Size**: ~20 MB (vs ~800 MB for Node.js)
- **Security**: Minimal attack surface, no shell, no package manager
- **Default Storage**: In-memory (for maximum portability with scratch)

**Note**: The scratch-based image has no writable filesystem. Use the memory driver or mount a volume for SQLite.

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

### Using the gRPC API

Example client (Go):
```go
import (
    "context"
    pb "github.com/Billy-Davies-2/jellycat-draft-ui/proto"
    "google.golang.org/grpc"
)

conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
defer conn.Close()

client := pb.NewDraftServiceClient(conn)
state, _ := client.GetState(context.Background(), &pb.Empty{})
```

## Regenerating Protocol Buffers

If you modify `proto/draft.proto`:

```bash
make proto
# or manually:
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/draft.proto
```

## Rebuilding Styles

If you modify TailwindCSS styles:

```bash
# Download TailwindCSS CLI (one time)
curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
chmod +x tailwindcss-linux-x64

# Rebuild styles
./tailwindcss-linux-x64 -c tailwind.config.go.js -i static/css/input.css -o static/css/styles.css --minify
```

## Adding New Data Drivers

1. Create a new file in `internal/dal/` (e.g., `postgres.go`)
2. Implement the `DraftDAL` interface defined in `internal/dal/types.go`
3. Register the driver in `main.go`'s switch statement
4. Set `DB_DRIVER` environment variable to use it

## License

MIT
