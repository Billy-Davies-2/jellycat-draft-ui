# Jellycat Fantasy Draft – Usage Guide

This guide covers running, configuring, and exploring the fantasy draft app with its pluggable data access layer (DAL) and mock integrations.

## Quick Start

Install dependencies (Bun):

```
bun install
bun run dev
```

The dev server uses the SQLite DAL by default and auto-seeds demo data plus plushie "jellycat" players (from `public/jellycats`).

## DAL Drivers

Configure via `DB_DRIVER` env var:

| Driver               | Value             | Description                                                           |
| -------------------- | ----------------- | --------------------------------------------------------------------- |
| Memory               | `memory`          | Ephemeral in-memory seed (fastest)                                    |
| SQLite               | `sqlite`          | File-backed `dev.sqlite` (default in dev) with jellycat image seeding |
| Postgres             | `postgres`        | Full SQL implementation (requires `DATABASE_URL`)                     |
| ClickHouse (overlay) | `clickhouse`      | Wraps memory DAL and merges external points from ClickHouse           |
| ClickHouse Mock      | `clickhouse-mock` | SQLite-backed mock overlay simulating ClickHouse points               |

### Switching Drivers

```
DB_DRIVER=postgres DATABASE_URL=postgres://user:pass@host/db bun run dev
```

If unset in development, it falls back to SQLite and creates/updates `dev.sqlite` eagerly.

## Jellycat Image Players

When using SQLite (or `clickhouse-mock` which wraps SQLite), the reset/seed step scans `public/jellycats` and creates one extra player per image with:

- `id`: `jcat_<filename>` (without extension)
- `position`: `PLUSH`
- `team`: `Jellycat`
- `tier`: `Z`
- `image`: `/jellycats/<file>`

Disable by removing the folder or switching drivers.

## Common Operations (tRPC Caller Examples)

- Add team: `draft.addTeam({ name: 'Testers' })`
- Add player: `draft.addPlayer({ name: 'Speedy', position: 'QB', team: 'Cats', points: 0, tier: 'A', image: '' })`
- Draft player: `draft.draftPlayer(playerId, teamId)`
- Set points: `draft.setPlayerPoints(playerId, 42)`
- Send chat: `chat.send({ text: 'Hello world' })`
- React: `chat.react({ id: messageId, emote: '👍' })`

## Realtime

In dev, a local WebSocket server broadcasts mutation events causing clients to refetch impacted queries.

Env overrides:

- `TRPC_WS_PORT` (server)
- `NEXT_PUBLIC_TRPC_WS_PORT` (client)

## Healthcheck

`/api/health` returns JSON indicating DAL and (if configured) ClickHouse readiness.

## Testing

Run all tests:

```
bun test
```

Postgres unit tests mock SQL with normalized pattern matching. The ClickHouse mock tests exercise the overlay logic and ensure points updates flow correctly.

## Adding a New Driver

1. Create `lib/dal/<name>.ts` exporting a class implementing `IDraftDAL`.
2. Add selection logic in `getDAL()` within `lib/db.ts`.
3. Add targeted unit tests under `tests/unit/`.

## Environment Variables

| Variable                                    | Purpose                                                                          |
| ------------------------------------------- | -------------------------------------------------------------------------------- |
| `DB_DRIVER`                                 | Selects driver (`memory`, `sqlite`, `postgres`, `clickhouse`, `clickhouse-mock`) |
| `SQLITE_FILE`                               | Override SQLite filename (default `dev.sqlite`)                                  |
| `DATABASE_URL`                              | Postgres connection string                                                       |
| `CLICKHOUSE_URL` / `CLICKHOUSE_HOST`        | ClickHouse base URL                                                              |
| `CLICKHOUSE_DB` / `CLICKHOUSE_DATABASE`     | ClickHouse database name                                                         |
| `CLICKHOUSE_USER` / `CLICKHOUSE_PASSWORD`   | ClickHouse auth                                                                  |
| `CLICKHOUSE_POINTS_TABLE`                   | Table with player points                                                         |
| `CLICKHOUSE_POINTS_QUERY`                   | Custom SELECT for points merge                                                   |
| `TRPC_WS_PORT` / `NEXT_PUBLIC_TRPC_WS_PORT` | Realtime port config                                                             |

## Resetting Data

Calling `dal.reset()` (or restarting dev server in SQLite) reseeds core + jellycat players.

## Contributing

See `CONTRIBUTING.md` for guidelines.

---

Happy drafting!
