// Client-safe shim delegating to server implementation when available.
import { MemoryDAL } from './dal/memory'
import type { IDraftDAL } from './dal/types'
function loadServer() {
  if (typeof window !== 'undefined') return null
  try {
    return require('./db.server')
  } catch {
    return null
  }
}
export function getDAL(): IDraftDAL {
  const srv = loadServer()
  if (srv?.getDAL) return srv.getDAL()
  return new MemoryDAL()
}
export async function dbHealthcheck(): Promise<boolean> {
  const srv = loadServer()
  if (srv?.dbHealthcheck) return srv.dbHealthcheck()
  return true
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
  const srv = loadServer()
  if (srv?.dbHealthDetails) return srv.dbHealthDetails()
  const now = Date.now()
  return { ok: true, db: true, driver: 'memory', timestamp: now, version: '0.0.0', uptimeMs: 0 }
}
