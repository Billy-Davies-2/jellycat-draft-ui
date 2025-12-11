# PostgreSQL Database Setup Guide

This guide covers setting up and bootstrapping the PostgreSQL database for the Jellycat Fantasy Draft application.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Database Installation](#database-installation)
- [Schema Initialization](#schema-initialization)
- [Data Seeding](#data-seeding)
- [Image Migration](#image-migration)
- [Environment Configuration](#environment-configuration)
- [Database Migrations](#database-migrations)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- PostgreSQL 12 or higher
- Access to create databases and users
- Go 1.24+ (for running the application)

## Database Installation

### On Ubuntu/Debian

```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Start PostgreSQL service
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### On macOS (Homebrew)

```bash
# Install PostgreSQL
brew install postgresql

# Start PostgreSQL service
brew services start postgresql
```

### Using Docker

```bash
# Run PostgreSQL in Docker
docker run --name jellycat-postgres \
  -e POSTGRES_PASSWORD=yourpassword \
  -e POSTGRES_USER=jellycatuser \
  -e POSTGRES_DB=jellycatdraft \
  -p 5432:5432 \
  -d postgres:15
```

## Schema Initialization

The application automatically creates the required schema on first run. The schema includes:

### Tables

#### 1. `players` Table

Stores information about Jellycat plush toys available for drafting.

```sql
CREATE TABLE IF NOT EXISTS players (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    position TEXT NOT NULL,
    team TEXT NOT NULL,
    points INTEGER NOT NULL,
    tier TEXT NOT NULL,
    drafted BOOLEAN NOT NULL DEFAULT false,
    drafted_by TEXT,
    image TEXT,
    image_data BYTEA,  -- Stores binary image data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_players_drafted ON players(drafted);
```

**Columns:**
- `id`: Unique player identifier
- `name`: Jellycat name (e.g., "Bashful Bunny")
- `position`: Player position (CC, SS, HH, CH)
- `team`: Jellycat team/collection (e.g., "Woodland", "Safari")
- `points`: Cuddle points (synced from ClickHouse analytics)
- `tier`: Performance tier (S, A, B, C)
- `drafted`: Whether the player has been drafted
- `drafted_by`: Name of the team that drafted this player
- `image`: Image URL path (e.g., "/images/bashful-bunny.png")
- `image_data`: Binary image data (BYTEA)
- `created_at`: Record creation timestamp
- `updated_at`: Last update timestamp

#### 2. `teams` Table

Stores draft teams.

```sql
CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    owner TEXT NOT NULL,
    mascot TEXT NOT NULL,
    color TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Columns:**
- `id`: Unique team identifier
- `name`: Team name (e.g., "Fluffy Foxes")
- `owner`: Team owner name
- `mascot`: Team mascot emoji (e.g., "🦊")
- `color`: Tailwind CSS color classes for UI

#### 3. `team_players` Table

Junction table linking teams to their drafted players.

```sql
CREATE TABLE IF NOT EXISTS team_players (
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    player_id TEXT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    player_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, player_id)
);
```

**Columns:**
- `team_id`: Reference to teams table
- `player_id`: Reference to players table
- `player_data`: Full player data snapshot (JSONB) at time of draft
- `created_at`: When the player was drafted

#### 4. `chat` Table

Stores chat messages and reactions.

```sql
CREATE TABLE IF NOT EXISTS chat (
    id TEXT PRIMARY KEY,
    ts BIGINT NOT NULL,
    type TEXT NOT NULL,
    text TEXT NOT NULL,
    emotes JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chat_ts ON chat(ts);
```

**Columns:**
- `id`: Unique message identifier
- `ts`: Unix timestamp in milliseconds
- `type`: Message type ("system" or "user")
- `text`: Message content
- `emotes`: Emoji reactions (JSONB map of emoji -> count)
- `created_at`: Record creation timestamp

## Data Seeding

The application automatically seeds initial data when the database is empty. This includes:

### Default Players (18 Jellycats)

The seed data includes popular Jellycat characters across different tiers:
- **Tier S**: Bashful Bunny, Fuddlewuddle Lion, Cordy Roy Elephant, Amuseable Avocado, Amuseable Pineapple
- **Tier A**: Blossom Tulip Bunny, Octopus Ollie, Jellycat Dragon, Bashful Lamb, Cordy Roy Fox, Blossom Peach Bunny, Amuseable Taco, Bashful Unicorn
- **Tier B**: Jellycat Penguin, Amuseable Moon, Cordy Roy Pig, Bashful Tiger, Amuseable Donut

### Default Teams (6 Teams)

Pre-configured draft teams:
1. Fluffy Foxes (Owner: Sarah)
2. Cuddly Bears (Owner: Mike)
3. Snuggly Bunnies (Owner: Emma)
4. Cozy Cats (Owner: Alex)
5. Soft Sheep (Owner: Jordan)
6. Gentle Giraffes (Owner: Taylor)

### System Chat Messages

Initial welcome messages are added to the chat.

## Image Migration

The application can automatically migrate static image files from `static/images/` into the database as binary data.

### Automatic Migration

When you seed the database, images are automatically loaded from the `static/images/` directory:

```go
// This happens automatically during seedData()
if err := postgresDAL.MigrateImagesToDatabase(); err != nil {
    log.Printf("Warning: Failed to migrate images: %v", err)
}
```

### Manual Migration

To manually migrate images to the database:

```bash
# Ensure static/images/ contains your PNG files
ls -la static/images/

# Run the application - it will migrate on first seeding
DB_DRIVER=postgres DATABASE_URL="postgres://..." ./jellycat-draft
```

### Image Serving

Images are served via the `/images/` endpoint:
- First, attempts to retrieve from database (`image_data` column)
- Falls back to serving from `static/images/` if not in database
- Cached with `max-age=31536000` (1 year)

## Environment Configuration

### Development (SQLite)

```bash
export DB_DRIVER=sqlite
export SQLITE_FILE=dev.sqlite
```

### Production (PostgreSQL)

```bash
export DB_DRIVER=postgres
export DATABASE_URL="postgres://username:password@hostname:5432/database?sslmode=disable"
```

### Connection String Format

```
postgres://[user]:[password]@[host]:[port]/[database]?[parameters]
```

**Example with SSL:**
```
postgres://jellycatuser:mypassword@localhost:5432/jellycatdraft?sslmode=require
```

**Example for local development:**
```
postgres://postgres:password@localhost:5432/jellycatdraft?sslmode=disable
```

## Database Migrations

### Reset Database

To reset the database and re-seed:

```sql
-- Connect to your database
psql -U username -d jellycatdraft

-- Truncate all tables (careful - this deletes all data!)
TRUNCATE team_players, chat, teams, players CASCADE;

-- Exit psql
\q
```

Then restart the application to trigger automatic seeding.

### Manual Schema Updates

If you need to add the `image_data` column to an existing database:

```sql
-- Add image_data column if it doesn't exist
ALTER TABLE players ADD COLUMN IF NOT EXISTS image_data BYTEA;
```

## Troubleshooting

### Connection Refused

**Issue:** Cannot connect to PostgreSQL
```
Failed to initialize Postgres: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solution:**
1. Verify PostgreSQL is running: `sudo systemctl status postgresql`
2. Check connection string has correct host/port
3. Verify PostgreSQL is listening on the correct interface

### Authentication Failed

**Issue:** Password authentication failed
```
pq: password authentication failed for user "username"
```

**Solution:**
1. Verify credentials in `DATABASE_URL`
2. Check `pg_hba.conf` authentication settings
3. Reset password if needed:
   ```sql
   ALTER USER username WITH PASSWORD 'newpassword';
   ```

### Permission Denied

**Issue:** Permission denied to create database
```
pq: permission denied to create database
```

**Solution:**
Grant necessary permissions:
```sql
GRANT CREATE ON SCHEMA public TO username;
```

### Images Not Loading

**Issue:** Player images show as broken

**Solution:**
1. Verify `static/images/` directory exists with PNG files
2. Check database has `image_data` column
3. Re-run seeding to migrate images
4. Check application logs for migration errors

### Too Many Connections

**Issue:** Too many database connections

**Solution:**
Adjust PostgreSQL `max_connections` in `postgresql.conf`:
```
max_connections = 100
```

Restart PostgreSQL:
```bash
sudo systemctl restart postgresql
```

## Example: Complete Setup

Here's a complete example of setting up PostgreSQL from scratch:

```bash
# 1. Install PostgreSQL (Ubuntu)
sudo apt update
sudo apt install postgresql postgresql-contrib

# 2. Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE jellycatdraft;
CREATE USER jellycatuser WITH ENCRYPTED PASSWORD 'securepassword';
GRANT ALL PRIVILEGES ON DATABASE jellycatdraft TO jellycatuser;
\c jellycatdraft
GRANT ALL ON SCHEMA public TO jellycatuser;
EOF

# 3. Configure environment
export DB_DRIVER=postgres
export DATABASE_URL="postgres://jellycatuser:securepassword@localhost:5432/jellycatdraft?sslmode=disable"

# 4. Run the application (schema auto-created)
./jellycat-draft

# Application will:
# - Create all tables
# - Seed default data
# - Migrate images from static/images/ to database
# - Start serving on port 3000
```

## Kubernetes Deployment

For deploying PostgreSQL on Kubernetes using the CloudNativePG operator, see:

- **[Kubernetes with CloudNativePG Guide](kubernetes-cloudnative-pg.md)** - Comprehensive guide for deploying on Kubernetes with CloudNativePG operator

The application is fully compatible with CloudNativePG (PostgreSQL 12-17) with no code changes required.

## Additional Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go pq Driver Documentation](https://pkg.go.dev/github.com/lib/pq)
- [Jellycat Draft README](../README.md)
- [Kubernetes CloudNativePG Guide](kubernetes-cloudnative-pg.md)

## Support

For issues or questions, please open an issue on GitHub or contact the development team.
