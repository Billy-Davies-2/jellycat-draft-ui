# Jellycat Fantasy Draft

Feature-rich Next.js (App Router) + Bun + tRPC application demonstrating a realtime fantasy draft with multi-backend data abstraction.

## Stack

- Next.js 15 / React 19 / TypeScript 5
- Bun runtime & test runner
- tRPC v11 (HTTP + WebSocket subscriptions in dev)
- Tailwind + shadcn/ui components
- Pluggable DAL: memory, SQLite (dev default), Postgres, ClickHouse, ClickHouse mock

## Data Access Drivers

Set `DB_DRIVER` to select:

- `sqlite` (default in dev) – stored in `dev.sqlite` (override with `SQLITE_FILE`)
- `memory` – ephemeral in-memory store
- `postgres` – uses `DATABASE_URL`
- `clickhouse` – merges points from real ClickHouse
- `clickhouse-mock` – simulates ClickHouse layer over SQLite

## Realtime

- WebSocket server (dev) at `ws://localhost:3001` (override `TRPC_WS_PORT`)
- Subscription: `events` channel; UI refetches queries on event receipt.

## Development

```
bun install
bun run dev
```

SQLite file auto-created & seeded.

## Testing

```
bun test
```

Includes unit tests for store, DALs, tRPC routers, pubsub, health, and Postgres/ClickHouse behaviors.

## Environment Variables

| Name                    | Purpose                                                           |
| ----------------------- | ----------------------------------------------------------------- |
| DB_DRIVER               | Select driver (sqlite/memory/postgres/clickhouse/clickhouse-mock) |
| SQLITE_FILE             | SQLite filename (default dev.sqlite)                              |
| DATABASE_URL            | Postgres connection string                                        |
| CLICKHOUSE_URL          | ClickHouse endpoint                                               |
| CLICKHOUSE_POINTS_QUERY | Override points query                                             |
| CLICKHOUSE_POINTS_TABLE | Overrides table for manual point updates                          |
| TRPC_WS_PORT            | Dev WebSocket port (default 3001)                                 |

## Adding a New Driver

1. Create `lib/dal/<driver>.ts` implementing `IDraftDAL`.
2. Register in `getDAL()` and extend healthcheck logic if needed.
3. Add unit tests mocking underlying client.

## Scripts

| Script | Description             |
| ------ | ----------------------- |
| dev    | Next dev with Bun       |
| build  | Build standalone output |
| start  | Run built app           |
| test   | Run full test suite     |

## Deployment

`next build` outputs a standalone build. Provide production env vars for Postgres/ClickHouse at runtime.

## License

MIT
