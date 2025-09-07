import { describe, it, expect, beforeAll } from 'bun:test'
import { getDAL } from '../../lib/db'

process.env.DB_DRIVER = 'mega-mock'

describe('SqliteMegaMockDAL', () => {
  let dal: any
  beforeAll(() => {
    dal = getDAL()
  })

  it('applies override points table like clickhouse', async () => {
    const state = await dal.getState()
    const player = state.players[0]
    await dal.setPlayerPoints(player.id, 555)
    const state2 = await dal.getState()
    const updated = state2.players.find((p: any) => p.id === player.id)
    expect(updated.points).toBe(555)
  })
})
