import { draftStore } from '../draft-store'
import type { IDraftDAL } from './types'

export class MemoryDAL implements IDraftDAL {
  async getState() {
    return draftStore.getState()
  }
  async reset() {
    draftStore.reset()
  }
  async addPlayer(i: any) {
    return draftStore.addPlayer(i)
  }
  async setPlayerPoints(id: string, points: number) {
    return draftStore.setPlayerPoints(id, points)
  }
  async reorderTeams(order: string[]) {
    return draftStore.reorderTeams(order)
  }
  async draftPlayer(playerId: string, teamId: string) {
    return draftStore.draftPlayer(playerId, teamId)
  }
  async addChatMessage(text: string, type: 'system' | 'user' = 'user') {
    return draftStore.addChatMessage(text, type)
  }
  async addReaction(messageId: string, emote: string, userId?: string) {
    return draftStore.addReaction(messageId, emote, userId)
  }
  async addTeam(input: { name: string; owner?: string; mascot?: string; color?: string }) {
    return draftStore.addTeam(input)
  }
}
