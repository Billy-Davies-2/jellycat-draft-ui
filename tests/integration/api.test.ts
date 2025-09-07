import { describe, it, expect, beforeEach } from 'bun:test'
import { appRouter } from '@/lib/trpc-router'

describe('tRPC Integration', () => {
  const caller = appRouter.createCaller({ isAdmin: false })

  beforeEach(async () => {
    await caller.draft.reset()
  })

  it('adds a team, adds a player, drafts it, and updates points', async () => {
    const team = await caller.teams.add({ name: 'Int Team', owner: 'CI' })
    const player = await caller.players.add({
      name: 'Int Player',
      position: 'CC',
      team: 'Ops',
      points: 10,
      tier: 'A',
      image: '/placeholder.svg',
    })

    await caller.draft.pick({ playerId: player.id, teamId: team.id })
    await caller.players.setPoints({ id: player.id, points: 99 })

    const state = await caller.draft.state()
    const drafted = state.teams
      .find((t: any) => t.id === team.id)!
      .players.find((p: any) => p.id === player.id)!
    expect(drafted.points).toBe(99)
  })
})
