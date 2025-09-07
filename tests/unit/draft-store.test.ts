import { describe, it, expect, beforeEach } from 'bun:test'
import { draftStore } from '@/lib/draft-store'

describe('draftStore', () => {
  beforeEach(() => draftStore.reset())

  it('adds a player and updates points', () => {
    const p = draftStore.addPlayer({
      name: 'Test Cat',
      position: 'CC',
      team: 'Lab',
      points: 123,
      tier: 'A',
      image: '/placeholder.svg',
    })
    expect(p.id).toBeTruthy()
    const updated = draftStore.setPlayerPoints(p.id, 200)
    expect(updated.points).toBe(200)
  })

  it('adds a team and drafts a player', () => {
    const t = draftStore.addTeam({ name: 'Unit Team', owner: 'Tester' })
    const p = draftStore.addPlayer({
      name: 'Pickable',
      position: 'SS',
      team: 'QA',
      points: 50,
      tier: 'B',
      image: '/placeholder.svg',
    })
    const res = draftStore.draftPlayer(p.id, t.id)
    expect(res.player.drafted).toBe(true)
    const state = draftStore.getState()
    const team = state.teams.find((x) => x.id === t.id)!
    expect(team.players.find((x) => x.id === p.id)).toBeTruthy()
  })

  it('chat messages and reactions', () => {
    const m = draftStore.addChatMessage('hello', 'user')
    draftStore.addReaction(m.id, '🎉')
    const { chat } = draftStore.getState()
    const msg = chat.find((x) => x.id === m.id)!
    expect(msg.emotes['🎉']).toBe(1)
  })
})
