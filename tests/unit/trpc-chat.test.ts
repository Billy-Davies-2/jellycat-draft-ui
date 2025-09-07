import { describe, it, expect, beforeEach } from 'bun:test'
import { appRouter } from '@/lib/trpc-router'

describe('tRPC chat', () => {
  const caller = appRouter.createCaller({ isAdmin: false })
  beforeEach(async () => {
    await caller.draft.reset()
  })

  it('sends and reacts to messages', async () => {
    const before = (await caller.chat.list()) ?? []
    const msg = await caller.chat.send({ text: 'hello from test' })
    await caller.chat.react({ messageId: msg.id, emote: '🎉' })
    const after = (await caller.chat.list()) ?? []
    const updated = after.find((m: any) => m.id === msg.id)!
    expect(updated).toBeTruthy()
    expect(updated.emotes['🎉']).toBe(1)
  })
})
