// Central shared DAL interface to avoid duplication across server/client shims.
export interface IDraftDAL {
  getState(): Promise<{ players: any[]; teams: any[]; chat?: any[] }>
  reset(): Promise<void>
  addPlayer(input: any): Promise<any>
  setPlayerPoints(id: string, points: number): Promise<any>
  reorderTeams(order: string[]): Promise<any[]>
  draftPlayer(playerId: string, teamId: string): Promise<any>
  addChatMessage(text: string, type?: 'system' | 'user'): Promise<any>
  addReaction(messageId: string, emote: string, userId?: string): Promise<any>
  addTeam(input: { name: string; owner?: string; mascot?: string; color?: string }): Promise<any>
}
