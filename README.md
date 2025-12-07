# Jellycat Fantasy Draft

A realtime fantasy draft application for Jellycat plush toys, built with **Go**, **htmx**, **gRPC**, and **TailwindCSS**.

## Stack

- **Backend**: Go 1.24+ with dual interface:
  - HTTP/REST server for SSR pages and htmx frontend  
  - gRPC server for programmatic API access
- **Frontend**: htmx for dynamic interactions, Server-Sent Events (SSE) for realtime updates
- **Styling**: TailwindCSS for modern, responsive design
- **Templates**: Go's html/template for server-side rendering
- **Data**: Pluggable DAL supporting memory and SQLite storage
- **API**: Protocol Buffers with gRPC streaming for events

## Features

- 🧸 Draft cute Jellycat plush toys in a fantasy draft format
- 👥 Team creation and management
- 💬 Live chat with emoji reactions
- 📊 Real-time updates via SSE (HTTP) and gRPC streaming
- 🎨 Beautiful UI with TailwindCSS
- 🔄 Multiple storage backends (memory, SQLite)
- 🔌 Dual API: HTTP/REST + gRPC
- 🧪 Comprehensive fuzz testing for both interfaces

## Quick Start

### Prerequisites

- Go 1.24 or higher
- (Optional) Protocol Buffers compiler for regenerating proto files
- (Optional) TailwindCSS CLI for stylesheet changes

### Development

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

3. **Run the server**
   ```bash
   # Using in-memory storage
   DB_DRIVER=memory ./jellycat-draft

   # Using SQLite (default for dev)
   DB_DRIVER=sqlite SQLITE_FILE=draft.db ./jellycat-draft
   ```

4. **Access the application**
   - HTTP UI: `http://localhost:3000`
   - gRPC API: `localhost:50051`

## Environment Variables

| Variable      | Description                           | Default      |
|---------------|---------------------------------------|--------------|
| `PORT`        | HTTP server port                      | `3000`       |
| `GRPC_PORT`   | gRPC server port                      | `50051`      |
| `DB_DRIVER`   | Database driver (`memory`, `sqlite`)  | `memory`     |
| `SQLITE_FILE` | SQLite database file path             | `dev.sqlite` |

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
│   │   └── sqlite.go      # SQLite implementation
│   ├── grpc/              # gRPC server implementation
│   │   └── server.go      # DraftService implementation
│   ├── handlers/          # HTTP handlers
│   ├── models/            # Data models
│   ├── pubsub/            # Pub/sub for realtime events
│   └── fuzz/              # Fuzz tests
│       ├── http_fuzz_test.go   # HTTP endpoint fuzz tests
│       └── grpc_fuzz_test.go   # gRPC endpoint fuzz tests
├── templates/             # HTML templates
│   ├── base.html         # Base layout
│   ├── start.html        # Team creation page
│   ├── draft.html        # Main draft page
│   └── admin.html        # Admin panel
└── static/               # Static assets
    ├── css/              # Stylesheets
    └── images/           # Jellycat images
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

Build and run with Docker:

```bash
docker build -t jellycat-draft .
docker run -p 3000:3000 -p 50051:50051 -v $(pwd)/data:/data jellycat-draft
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
