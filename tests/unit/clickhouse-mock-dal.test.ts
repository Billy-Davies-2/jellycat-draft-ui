import { describe, it, expect, beforeEach } from 'bun:test'
import { getDAL } from '@/lib/db'
import { draftStore } from '@/lib/draft-store'

describe('ClickHouse mock via sqlite', () => {
  beforeEach(() => {
    draftStore.reset()
    process.env.DB_DRIVER = 'clickhouse-mock'
  })

  it('returns state and allows point mutation passthrough', async () => {
    const dal: any = getDAL()
    const state = await dal.getState()
    const p = state.players[0]
    const updated = await dal.setPlayerPoints(p.id, p.points + 100)
    expect(updated.points).toBe(p.points + 100)
    const state2 = await dal.getState()
    const p2 = state2.players.find((x: any) => x.id === p.id)!
    expect(p2.points).toBe(p.points + 100)
  })
})
