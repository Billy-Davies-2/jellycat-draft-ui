# Jellycat Fantasy Draft

A realtime fantasy draft application for Jellycat plush toys, built with **Go**, **htmx**, and **TailwindCSS**.

## Stack

- **Backend**: Go 1.24+ with standard library HTTP server
- **Frontend**: htmx for dynamic interactions, Server-Sent Events (SSE) for realtime updates
- **Styling**: TailwindCSS for modern, responsive design
- **Templates**: Go's html/template
- **Data**: Pluggable DAL supporting memory and SQLite storage

## Features

- 🧸 Draft cute Jellycat plush toys in a fantasy draft format
- 👥 Team creation and management
- 💬 Live chat with emoji reactions
- 📊 Real-time updates via Server-Sent Events
- 🎨 Beautiful UI with TailwindCSS
- 🔄 Multiple storage backends (memory, SQLite)

## Quick Start

### Prerequisites

- Go 1.24 or higher
- (Optional) TailwindCSS CLI for stylesheet changes

### Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/Billy-Davies-2/jellycat-draft-ui.git
   cd jellycat-draft-ui
   ```

2. **Build the application**
   ```bash
   go build -o jellycat-draft main.go
   ```

3. **Run the server**
   ```bash
   # Using in-memory storage
   DB_DRIVER=memory ./jellycat-draft

   # Using SQLite (default for dev)
   DB_DRIVER=sqlite SQLITE_FILE=draft.db ./jellycat-draft
   ```

4. **Open your browser**
   ```
   http://localhost:3000
   ```

## Environment Variables

| Variable     | Description                           | Default      |
|--------------|---------------------------------------|--------------|
| `PORT`       | Server port                           | `3000`       |
| `DB_DRIVER`  | Database driver (`memory`, `sqlite`)  | `memory`     |
| `SQLITE_FILE`| SQLite database file path             | `dev.sqlite` |

## Project Structure

```
.
├── main.go                 # Application entry point
├── internal/
│   ├── dal/               # Data Access Layer
│   │   ├── types.go       # DAL interface
│   │   ├── memory.go      # In-memory implementation
│   │   └── sqlite.go      # SQLite implementation
│   ├── handlers/          # HTTP handlers
│   ├── models/            # Data models
│   └── pubsub/            # Pub/sub for realtime events
├── templates/             # HTML templates
│   ├── base.html         # Base layout
│   ├── start.html        # Team creation page
│   ├── draft.html        # Main draft page
│   └── admin.html        # Admin panel
└── static/               # Static assets
    ├── css/              # Stylesheets
    └── images/           # Jellycat images
```

## Docker Deployment

Build and run with Docker:

```bash
docker build -t jellycat-draft .
docker run -p 3000:3000 -v $(pwd)/data:/data jellycat-draft
```

## API Endpoints

### Draft Operations
- `GET /api/draft/state` - Get current draft state
- `POST /api/draft/pick` - Draft a player
- `POST /api/draft/reset` - Reset the draft

### Team Operations
- `GET /api/teams` - List all teams
- `POST /api/teams/add` - Create a new team
- `POST /api/teams/reorder` - Reorder teams

### Player Operations
- `POST /api/players/add` - Add a new player
- `POST /api/players/points` - Update player points
- `GET /api/players/profile` - Get player profile

### Chat Operations
- `GET /api/chat/list` - Get all chat messages
- `POST /api/chat/send` - Send a chat message
- `POST /api/chat/react` - Add a reaction to a message

### Realtime
- `GET /api/events` - Server-Sent Events stream for live updates

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
