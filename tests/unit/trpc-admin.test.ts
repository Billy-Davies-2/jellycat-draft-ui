import { describe, it, expect, beforeEach } from 'bun:test'
import { appRouter } from '@/lib/trpc-router'

describe('tRPC admin', () => {
  const caller = appRouter.createCaller({ isAdmin: true })
  beforeEach(async () => {
    await caller.draft.reset()
  })

  it('reorders teams and adds a player', async () => {
    // add a new team
    const t = await caller.teams.add({ name: 'Reorder A', owner: 'X' })
    const t2 = await caller.teams.add({ name: 'Reorder B', owner: 'Y' })
    const list = await caller.teams.list()
    const order = [
      t2.id,
      t.id,
      ...list.filter((x) => x.id !== t.id && x.id !== t2.id).map((x) => x.id),
    ]
    const reordered = await caller.teams.reorder({ order })
    expect(reordered[0].id).toBe(t2.id)

    const p = await caller.players.add({
      name: 'AdminCat',
      position: 'CH',
      team: 'Backoffice',
      points: 1,
      tier: 'C',
      image: '/placeholder.svg',
    })
    expect(p.id).toBeTruthy()
  })
})
