// Server-only full DAL implementation. Not bundled client-side.
import { MemoryDAL } from './dal/memory'
import type { IDraftDAL } from './dal/types'

// Postgres is optional; hide require from bundler (dynamic indirection)
let _PostgresDAL: any
function loadPostgres() {
  if (process.env.DB_DRIVER !== 'postgres') return null
  if (_PostgresDAL) return _PostgresDAL
  try {
    const p1 = './dal'
    const p2 = '/postgres'
    // eslint-disable-next-line no-eval
    const req = eval('require') as NodeRequire
    _PostgresDAL = (req(p1 + p2) as any).PostgresDAL
  } catch {
    _PostgresDAL = null
  }
  return _PostgresDAL
}

// IDraftDAL imported from shared types

const APP_START_TIME = Date.now()
let APP_VERSION = process.env.APP_VERSION || '0.0.0'
try {
  const pkg = require('../package.json')
  if (pkg?.version) APP_VERSION = pkg.version
} catch {}

// Dynamic SQLite loader (hidden from bundler)
type SqliteCtor = new () => IDraftDAL
function loadSqlite(): SqliteCtor | null {
  if (
    !process.env.DB_DRIVER ||
    ['sqlite', 'mega-mock', 'sqlite-mega', 'clickhouse-mock', 'memory'].includes(
      process.env.DB_DRIVER,
    )
  ) {
    try {
      const p = './dal'
      const s = '/sqlite.server'
      // eslint-disable-next-line no-eval
      const req = eval('require') as NodeRequire
      return (req(p + s) as any).SqliteDAL || null
    } catch {
      return null
    }
  }
  return null
}

// Mega mock wrapper defers to loaded sqlite instance and augments behavior
class SqliteMegaMockDAL implements IDraftDAL {
  private base: any
  private ensured = false
  private overrideTable = 'clickhouse_points_overrides'
  constructor(base: any) {
    this.base = base
  }
  private ensureExtra() {
    if (this.ensured) return
    try {
      this.base?.db?.exec(
        `CREATE TABLE IF NOT EXISTS ${this.overrideTable} (id TEXT PRIMARY KEY, points INTEGER NOT NULL);`,
      )
    } catch {}
    this.ensured = true
  }
  async getState() {
    this.ensureExtra()
    const b = await this.base.getState()
    try {
      const rows = this.base?.db?.query(`SELECT id, points FROM ${this.overrideTable}`).all() || []
      const map = new Map(rows.map((r: any) => [String(r.id), Number(r.points)]))
      for (const p of b.players) {
        const v = map.get(String(p.id))
        if (typeof v === 'number') p.points = v
      }
      for (const t of b.teams) {
        for (const p of t.players) {
          const v = map.get(String(p.id))
          if (typeof v === 'number') p.points = v
        }
      }
    } catch {}
    return b
  }
  async reset() {
    return this.base.reset()
  }
  addPlayer(i: any) {
    return this.base.addPlayer(i)
  }
  setPlayerPoints(id: string, points: number) {
    this.ensureExtra()
    try {
      this.base?.db
        ?.query(
          `INSERT INTO ${this.overrideTable} (id, points) VALUES (?,?) ON CONFLICT(id) DO UPDATE SET points=excluded.points`,
        )
        .run(String(id), Number(points))
    } catch {}
    return this.base.setPlayerPoints(id, points)
  }
  reorderTeams(o: string[]) {
    return this.base.reorderTeams(o)
  }
  draftPlayer(p: string, t: string) {
    return this.base.draftPlayer(p, t)
  }
  addChatMessage(t: string, ty: 'system' | 'user' = 'user') {
    return this.base.addChatMessage(t, ty)
  }
  addReaction(m: string, e: string) {
    return this.base.addReaction(m, e)
  }
  addTeam(inp: any) {
    return this.base.addTeam(inp)
  }
}

// Re-use a single instance per process to make dev UX consistent & retain mock data
declare global {
  // eslint-disable-next-line no-var
  var __DAL_INSTANCE__: IDraftDAL | null | undefined
}
let _dalInstance: IDraftDAL | null = globalThis.__DAL_INSTANCE__ ?? null

export function getDAL(): IDraftDAL {
  if (_dalInstance) return _dalInstance
  const driver =
    process.env.DB_DRIVER ||
    process.env.NEXT_PUBLIC_DB_DRIVER ||
    (process.env.NODE_ENV === 'development' ? 'mega-mock' : 'memory')

  // Lightweight log (once) so devs know what backend they are on
  if (process.env.NODE_ENV === 'development' && !process.env.QUIET_DB_INIT) {
    // eslint-disable-next-line no-console
    console.log(`[db] Using driver: ${driver}`)
  }

  if (process.env.NODE_ENV === 'development') {
    const Sqlite = loadSqlite()
    if (
      Sqlite &&
      (!process.env.DB_DRIVER || process.env.DB_DRIVER === 'sqlite' || driver === 'memory')
    ) {
      _dalInstance = new Sqlite()
      return _dalInstance
    }
  }
  if (driver === 'postgres') {
    const PG = loadPostgres()
    if (PG) {
      _dalInstance = new PG() as IDraftDAL
      return _dalInstance
    }
    _dalInstance = new MemoryDAL()
    return _dalInstance
  }
  if (driver === 'clickhouse' || driver === 'clickhouse-mock') {
    try {
      const partA = './dal-clickhouse'
      const partB = '.server'
      const mod = require(partA + partB)
      if (driver === 'clickhouse') {
        _dalInstance = new mod.RealClickHouseDAL(new MemoryDAL()) as IDraftDAL
        return _dalInstance
      }
      const Sqlite = loadSqlite()
      _dalInstance = new mod.MockClickHouseDAL(Sqlite ? new Sqlite() : new MemoryDAL()) as IDraftDAL
      return _dalInstance
    } catch {
      _dalInstance = new MemoryDAL()
      return _dalInstance
    }
  }
  if (driver === 'mega-mock' || driver === 'sqlite-mega') {
    const Sqlite = loadSqlite()
    if (Sqlite) {
      _dalInstance = new SqliteMegaMockDAL(new Sqlite())
      return _dalInstance
    }
  }
  if (driver === 'sqlite') {
    const Sqlite = loadSqlite()
    if (Sqlite) {
      _dalInstance = new Sqlite()
      return _dalInstance
    }
  }
  _dalInstance = new MemoryDAL()
  globalThis.__DAL_INSTANCE__ = _dalInstance
  return _dalInstance
}

export async function dbHealthcheck(): Promise<boolean> {
  const driver = process.env.DB_DRIVER || process.env.NEXT_PUBLIC_DB_DRIVER || 'memory'
  if (driver !== 'postgres') return true
  try {
    const pg: any = await import('pg')
    const Pool = pg.Pool || pg.default?.Pool
    if (!Pool) return false
    const pool = new Pool({
      connectionString: process.env.DATABASE_URL,
      ssl:
        process.env.PGSSL === 'true'
          ? { rejectUnauthorized: process.env.PGSSL_REJECT_UNAUTHORIZED !== 'false' }
          : undefined,
    })
    const res = await pool.query('SELECT 1 AS ok')
    await pool.end()
    const okVal = res?.rows?.[0]?.ok
    return okVal === 1 || okVal === '1'
  } catch {
    return false
  }
}

export async function dbHealthDetails(): Promise<{
  ok: boolean
  db: boolean
  driver: string
  timestamp: number
  version: string
  uptimeMs: number
  clickhouse?: { enabled: boolean; ok: boolean }
}> {
  const driver = process.env.DB_DRIVER || process.env.NEXT_PUBLIC_DB_DRIVER || 'memory'
  const db = await dbHealthcheck()
  let ch: { enabled: boolean; ok: boolean } | undefined
  if (driver === 'clickhouse' || driver === 'clickhouse-mock') ch = { enabled: true, ok: true }
  const now = Date.now()
  return {
    ok: db,
    db,
    driver,
    timestamp: now,
    version: APP_VERSION,
    uptimeMs: now - APP_START_TIME,
    ...(ch ? { clickhouse: ch } : {}),
  }
}
