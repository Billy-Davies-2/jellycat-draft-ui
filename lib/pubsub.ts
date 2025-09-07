import { EventEmitter } from 'events'

export type AppEvent =
  | { type: 'chat:add'; payload: { id: string } }
  | { type: 'chat:react'; payload: { id: string; emote: string } }
  | { type: 'draft:pick'; payload: { playerId: string; teamId: string } }
  | { type: 'teams:reorder' }
  | { type: 'players:updatePoints'; payload: { id: string; points: number } }
  | { type: 'teams:add'; payload: { id: string } }

class PubSub extends EventEmitter {
  emitEvent(e: AppEvent) {
    this.emit('event', e)
  }
  onEvent(handler: (e: AppEvent) => void) {
    this.on('event', handler)
    return () => this.off('event', handler)
  }
}

export const pubsub = new PubSub()
