// Server-only ClickHouse DAL implementations (isolated from client bundles)
export class RealClickHouseDAL {
  private _client: any
  constructor(private base: any) {}
  private get client() {
    if (!this._client) {
      if (typeof process === 'undefined') throw new Error('ClickHouse not available')
      if ((process.env.DB_DRIVER || '') !== 'clickhouse')
        throw new Error('Inactive clickhouse driver')
      const nodeRequire = typeof require === 'function' ? require : (0, eval)('require')
      const ch = nodeRequire('@clickhouse/client') as any
      const url =
        process.env.CLICKHOUSE_URL || process.env.CLICKHOUSE_HOST || 'http://localhost:8123'
      const database = process.env.CLICKHOUSE_DATABASE || process.env.CLICKHOUSE_DB || 'default'
      const username = process.env.CLICKHOUSE_USER || 'default'
      const password = process.env.CLICKHOUSE_PASSWORD || ''
      this._client = ch.createClient({ host: url, database, username, password })
    }
    return this._client
  }
  async getState() {
    const baseState = await this.base.getState()
    const query =
      process.env.CLICKHOUSE_POINTS_QUERY || 'SELECT id, points FROM jellycat_points_latest'
    try {
      const result = await this.client.query({ query, format: 'JSONEachRow' })
      const rows: Array<{ id: string; points: number }> = await result.json()
      const map = new Map(rows.map((r) => [String(r.id), Number(r.points)]))
      for (const p of baseState.players) {
        const v = map.get(String(p.id))
        if (typeof v === 'number') p.points = v
      }
      for (const t of baseState.teams) {
        for (const p of t.players) {
          const v = map.get(String(p.id))
          if (typeof v === 'number') p.points = v
        }
      }
    } catch {
      /* silent */
    }
    return baseState
  }
  async setPlayerPoints(id: string, points: number) {
    const table = process.env.CLICKHOUSE_POINTS_TABLE
    if (table) {
      try {
        await this.client.insert({
          table,
          values: [{ id: String(id), points: Number(points) }],
          format: 'JSONEachRow',
        })
      } catch {}
    }
    return this.base.setPlayerPoints(id, points)
  }
  addPlayer(...a: any[]) {
    return this.base.addPlayer(...a)
  }
  addTeam(...a: any[]) {
    return this.base.addTeam(...a)
  }
  draftPlayer(...a: any[]) {
    return this.base.draftPlayer(...a)
  }
  reorderTeams(...a: any[]) {
    return this.base.reorderTeams(...a)
  }
  addChatMessage(...a: any[]) {
    return this.base.addChatMessage(...a)
  }
  addReaction(...a: any[]) {
    return this.base.addReaction(...a)
  }
  reset(...a: any[]) {
    return this.base.reset(...a)
  }
}
export class MockClickHouseDAL extends RealClickHouseDAL {}
