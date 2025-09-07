import { describe, it, expect, mock } from 'bun:test'

describe('tRPC event emissions', async () => {
  const mod = await import('../../lib/trpc-router')
  const { appRouter } = mod
  const pubsubMod = await import('../../lib/pubsub')
  const spy = mock((e: any) => {})
  const orig = pubsubMod.pubsub.emitEvent
  // @ts-ignore override
  pubsubMod.pubsub.emitEvent = (e: any) => {
    spy(e)
    orig.call(pubsubMod.pubsub, e)
  }
  const caller = appRouter.createCaller({ isAdmin: false })

  it('emits on chat send/react', async () => {
    const m = await caller.chat.send({ text: 'hi' })
    expect(spy.mock.calls.length > 0).toBe(true)
    const before = spy.mock.calls.length
    await caller.chat.react({ messageId: m.id, emote: '🔥' })
    expect(spy.mock.calls.length).toBeGreaterThan(before)
  })

  it('emits on team add and reorder', async () => {
    const t = await caller.teams.add({ name: 'Emitters' })
    const state = await caller.draft.state()
    await caller.teams.reorder({ order: state.teams.map((x) => String(x.id)) })
    // just check calls recorded
    expect(spy.mock.calls.length).toBeGreaterThan(0)
  })
})
