import { describe, it, expect, beforeAll } from 'bun:test'
import { getDAL } from '../../lib/db'

// Force sqlite for this suite
process.env.DB_DRIVER = 'sqlite'

describe('SQLite jellycat image seeding', () => {
  let players: any[]
  beforeAll(async () => {
    const dal: any = getDAL()
    await dal.reset()
    const state = await dal.getState()
    players = state.players
  })

  it('adds jellycat image players with PLUSH position', () => {
    const plush = players.filter((p) => p.position === 'PLUSH' && p.team === 'Jellycat')
    expect(plush.length).toBeGreaterThan(0)
    // Ensure image path present
    expect(plush.every((p) => p.image && p.image.startsWith('/jellycats/'))).toBe(true)
  })

  it('does not duplicate jellycat players on second reset', async () => {
    const dal: any = getDAL()
    await dal.reset()
    const second = await dal.getState()
    const firstPlush = players.filter((p: any) => p.position === 'PLUSH').length
    const secondPlush = second.players.filter((p: any) => p.position === 'PLUSH').length
    expect(secondPlush).toBe(firstPlush) // stable count
  })
})
