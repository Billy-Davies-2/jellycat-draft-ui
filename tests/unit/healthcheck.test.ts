import { describe, it, expect, mock } from 'bun:test'

describe('dbHealthcheck', () => {
  it('returns true in memory mode', async () => {
    const mod = await import('../../lib/db')
    const orig = process.env.DB_DRIVER
    process.env.DB_DRIVER = 'memory'
    const ok = await mod.dbHealthcheck()
    expect(ok).toBe(true)
    if (orig === undefined) delete process.env.DB_DRIVER
    else process.env.DB_DRIVER = orig
  })

  it('treats clickhouse as healthy when skipped (no client required)', async () => {
    const mod = await import('../../lib/db')
    const orig = process.env.DB_DRIVER
    process.env.DB_DRIVER = 'clickhouse'
    const ok = await mod.dbHealthcheck()
    expect(ok).toBe(true)
    if (orig === undefined) delete process.env.DB_DRIVER
    else process.env.DB_DRIVER = orig
  })
})
