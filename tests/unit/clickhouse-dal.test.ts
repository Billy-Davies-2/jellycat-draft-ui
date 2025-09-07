import { describe, it, expect, beforeEach, afterEach, mock } from 'bun:test'
import { getDAL, dbHealthcheck } from '@/lib/db'
import { draftStore } from '@/lib/draft-store'

// Track calls on the mocked ClickHouse client
const calls = { query: 0, insert: 0 }

mock.module('@clickhouse/client', () => ({
  createClient() {
    return {
      async query(_opts: any) {
        calls.query++
        // healthcheck asks for SELECT 1 AS ok
        if (_opts?.query?.toLowerCase?.().includes('select 1')) {
          return { json: async () => [{ ok: 1 }] }
        }
        // points query returns a single override for id "1"
        return { json: async () => [{ id: '1', points: 777 }] }
      },
      async insert(_opts: any) {
        calls.insert++
      },
    }
  },
}))

const OLD_ENV = { ...process.env }

describe('ClickHouse DAL', () => {
  beforeEach(() => {
    Object.assign(process.env, OLD_ENV)
    process.env.DB_DRIVER = 'clickhouse'
    delete process.env.CLICKHOUSE_POINTS_TABLE
    delete process.env.CLICKHOUSE_POINTS_QUERY
    draftStore.reset()
    calls.query = 0
    calls.insert = 0
  })

  afterEach(() => {
    Object.assign(process.env, OLD_ENV)
  })

  it('merges points from ClickHouse into state', async () => {
    const dal = getDAL()
    const state = await dal.getState()
    const p1 = state.players.find((p) => p.id === '1')!
    expect(p1.points).toBe(777)
    expect(calls.query).toBeGreaterThan(0)
  })

  it('writes to overrides table on setPlayerPoints when configured', async () => {
    process.env.CLICKHOUSE_POINTS_TABLE = 'jellycat_points_overrides'
    const dal = getDAL()
    const updated = await dal.setPlayerPoints('1', 999)
    expect(updated.points).toBe(999)
    expect(calls.insert).toBe(1)
  })

  it('dbHealthcheck succeeds via ClickHouse mock', async () => {
    process.env.DB_DRIVER = 'clickhouse'
    const ok = await dbHealthcheck()
    expect(ok).toBe(true)
  })
})
