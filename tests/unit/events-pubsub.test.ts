import { describe, it, expect } from 'bun:test'
import { pubsub } from '@/lib/pubsub'

function once<T = any>(): { next: Promise<T>; emit: (v: T) => void } {
  let resolve!: (v: T) => void
  const next = new Promise<T>((res) => (resolve = res))
  return { next, emit: (v) => resolve(v) }
}

describe('events pubsub', () => {
  it('emits and receives events', async () => {
    const { next, emit } = once()
    const off = pubsub.onEvent((e) => emit(e))
    try {
      pubsub.emitEvent({ type: 'chat:add', payload: { id: 'x' } })
      const e = await next
      expect(e).toEqual({ type: 'chat:add', payload: { id: 'x' } })
    } finally {
      off()
    }
  })
})
