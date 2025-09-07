import type { IDraftDAL } from '../db'
import { draftStore } from '../draft-store'

function genId(prefix = 'id') {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}`
}

export class PostgresDAL implements IDraftDAL {
  private _pool: any
  private _schemaReady = false
  constructor(pool?: any) {
    if (pool) this._pool = pool
  }
  private get pool() {
    if (!this._pool) {
      const pg = require('pg') as any
      const Pool = pg.Pool || pg.default?.Pool
      this._pool = new Pool({
        connectionString: process.env.DATABASE_URL,
        ssl:
          process.env.PGSSL === 'true'
            ? { rejectUnauthorized: process.env.PGSSL_REJECT_UNAUTHORIZED !== 'false' }
            : undefined,
      })
    }
    return this._pool
  }
  private async ensureSchema() {
    if (this._schemaReady) return
    const q = (sql: string) => this.pool.query(sql)
    await q(
      `CREATE TABLE IF NOT EXISTS teams (id TEXT PRIMARY KEY,name TEXT NOT NULL,owner TEXT NOT NULL,mascot TEXT NOT NULL,color TEXT NOT NULL,sort_order INTEGER NOT NULL DEFAULT 0)`,
    )
    await q(
      `CREATE TABLE IF NOT EXISTS players (id TEXT PRIMARY KEY,name TEXT NOT NULL,position TEXT NOT NULL,team TEXT NOT NULL,points INTEGER NOT NULL,tier TEXT NOT NULL,drafted BOOLEAN NOT NULL DEFAULT FALSE,image TEXT NOT NULL)`,
    )
    await q(
      `CREATE TABLE IF NOT EXISTS team_players (team_id TEXT NOT NULL,player_id TEXT NOT NULL,PRIMARY KEY(team_id,player_id))`,
    )
    await q(
      `CREATE TABLE IF NOT EXISTS chat_messages (id TEXT PRIMARY KEY,ts BIGINT NOT NULL,type TEXT NOT NULL,text TEXT NOT NULL)`,
    )
    await q(
      `CREATE TABLE IF NOT EXISTS chat_reactions (message_id TEXT NOT NULL,emote TEXT NOT NULL,count INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(message_id,emote))`,
    )
    this._schemaReady = true
  }
  async getState() {
    await this.ensureSchema()
    const [teamsRes, playersRes, chatRes, reactRes] = await Promise.all([
      this.pool.query(
        'SELECT id,name,owner,mascot,color,sort_order FROM teams ORDER BY sort_order,name',
      ),
      this.pool.query(
        'SELECT id,name,position,team,points,tier,drafted,image FROM players ORDER BY tier,points DESC',
      ),
      this.pool.query('SELECT id,ts,type,text FROM chat_messages ORDER BY ts ASC'),
      this.pool.query('SELECT message_id,emote,count FROM chat_reactions'),
    ])
    const players: any[] = playersRes.rows.map((r: any) => ({ ...r }))
    const pById = new Map(players.map((p: any) => [String(p.id), p]))
    const teams: any[] = teamsRes.rows.map((t: any) => ({ ...t, players: [] as any[] }))
    const rosterRes = await this.pool.query('SELECT team_id,player_id FROM team_players')
    for (const row of rosterRes.rows) {
      const team = teams.find((t: any) => String(t.id) === String(row.team_id))
      const p = pById.get(String(row.player_id))
      if (team && p) team.players.push({ ...p })
    }
    for (const p of players) {
      p.drafted =
        rosterRes.rows.some((r: any) => String(r.player_id) === String(p.id)) || !!p.drafted
      if (p.drafted) {
        const ownerTeam = teams.find((t: any) =>
          t.players.some((tp: any) => String(tp.id) === String(p.id)),
        )
        if (ownerTeam) p.draftedBy = ownerTeam.name
      }
    }
    const reactsMap = new Map<string, Record<string, number>>()
    for (const r of reactRes.rows) {
      const emotes = reactsMap.get(String(r.message_id)) || {}
      emotes[String(r.emote)] = Number(r.count)
      reactsMap.set(String(r.message_id), emotes)
    }
    const chat = chatRes.rows.map((m: any) => ({ ...m, emotes: reactsMap.get(String(m.id)) || {} }))
    return { players, teams, chat }
  }
  async reset() {
    await this.ensureSchema()
    await this.pool.query('DELETE FROM chat_reactions')
    await this.pool.query('DELETE FROM chat_messages')
    await this.pool.query('DELETE FROM team_players')
    await this.pool.query('DELETE FROM players')
    await this.pool.query('DELETE FROM teams')
    draftStore.reset()
    const seed = draftStore.getState()
    for (let i = 0; i < seed.teams.length; i++) {
      const t = seed.teams[i]
      await this.pool.query(
        'INSERT INTO teams (id,name,owner,mascot,color,sort_order) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,owner=EXCLUDED.owner,mascot=EXCLUDED.mascot,color=EXCLUDED.color,sort_order=EXCLUDED.sort_order',
        [String(t.id), t.name, t.owner, t.mascot, t.color, i],
      )
    }
    for (const p of seed.players) {
      await this.pool.query(
        'INSERT INTO players (id,name,position,team,points,tier,drafted,image) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,position=EXCLUDED.position,team=EXCLUDED.team,points=EXCLUDED.points,tier=EXCLUDED.tier,drafted=EXCLUDED.drafted,image=EXCLUDED.image',
        [String(p.id), p.name, p.position, p.team, p.points, p.tier, false, p.image],
      )
    }
  }
  async addPlayer(input: any) {
    await this.ensureSchema()
    const id = input.id ?? genId('player')
    await this.pool.query(
      'INSERT INTO players (id,name,position,team,points,tier,drafted,image) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)',
      [
        String(id),
        input.name,
        input.position,
        input.team,
        Number(input.points),
        input.tier,
        !!input.drafted,
        input.image || '',
      ],
    )
    const r = await this.pool.query(
      'SELECT id,name,position,team,points,tier,drafted,image FROM players WHERE id=$1',
      [String(id)],
    )
    return r.rows[0]
  }
  async setPlayerPoints(id: string, points: number) {
    await this.ensureSchema()
    await this.pool.query('UPDATE players SET points=$2 WHERE id=$1', [String(id), Number(points)])
    const r = await this.pool.query(
      'SELECT id,name,position,team,points,tier,drafted,image FROM players WHERE id=$1',
      [String(id)],
    )
    return r.rows[0]
  }
  async reorderTeams(order: string[]) {
    await this.ensureSchema()
    for (let i = 0; i < order.length; i++)
      await this.pool.query('UPDATE teams SET sort_order=$2 WHERE id=$1', [String(order[i]), i])
    const r = await this.pool.query(
      'SELECT id,name,owner,mascot,color,sort_order FROM teams ORDER BY sort_order,name',
    )
    return r.rows.map((t: any) => ({ ...t }))
  }
  async draftPlayer(playerId: string, teamId: string) {
    await this.ensureSchema()
    const pRes = await this.pool.query(
      'SELECT id,name,position,team,points,tier,drafted,image FROM players WHERE id=$1',
      [String(playerId)],
    )
    const tRes = await this.pool.query(
      'SELECT id,name,owner,mascot,color,sort_order FROM teams WHERE id=$1',
      [String(teamId)],
    )
    const player = pRes.rows[0]
    const team = tRes.rows[0]
    if (!player) throw new Error('Player not found')
    if (!team) throw new Error('Team not found')
    if (player.drafted) throw new Error('Player already drafted')
    await this.pool.query(
      'INSERT INTO team_players (team_id,player_id) VALUES ($1,$2) ON CONFLICT DO NOTHING',
      [String(teamId), String(playerId)],
    )
    await this.pool.query('UPDATE players SET drafted=TRUE WHERE id=$1', [String(playerId)])
    const rosterP = { ...player, drafted: true, draftedBy: team.name }
    return { player: rosterP, team: { ...team, players: [rosterP] } }
  }
  async addChatMessage(text: string, type: 'system' | 'user' = 'user') {
    await this.ensureSchema()
    const id = genId('msg')
    const ts = Date.now()
    await this.pool.query('INSERT INTO chat_messages (id,ts,type,text) VALUES ($1,$2,$3,$4)', [
      id,
      ts,
      type,
      text,
    ])
    return { id, ts, type, text, emotes: {} }
  }
  async addReaction(messageId: string, emote: string) {
    await this.ensureSchema()
    await this.pool.query(
      'INSERT INTO chat_reactions (message_id,emote,count) VALUES ($1,$2,1) ON CONFLICT (message_id,emote) DO UPDATE SET count = chat_reactions.count + 1',
      [String(messageId), String(emote)],
    )
    const m = await this.pool.query('SELECT id,ts,type,text FROM chat_messages WHERE id=$1', [
      String(messageId),
    ])
    const reacts = await this.pool.query(
      'SELECT message_id,emote,count FROM chat_reactions WHERE message_id=$1',
      [String(messageId)],
    )
    const emotes: Record<string, number> = {}
    for (const r of reacts.rows) emotes[String(r.emote)] = Number(r.count)
    return { ...m.rows[0], emotes }
  }
  async addTeam(input: { name: string; owner?: string; mascot?: string; color?: string }) {
    await this.ensureSchema()
    const id = genId('team')
    const owner = input.owner || 'Anonymous'
    const mascot = input.mascot || '🧸'
    const color = input.color || 'bg-amber-100 border-amber-300'
    const next = await this.pool.query(
      'SELECT COALESCE(MAX(sort_order), -1) + 1 as next FROM teams',
    )
    const sortOrder = Number(next.rows[0]?.next ?? 0)
    await this.pool.query(
      'INSERT INTO teams (id,name,owner,mascot,color,sort_order) VALUES ($1,$2,$3,$4,$5,$6)',
      [String(id), input.name, owner, mascot, color, sortOrder],
    )
    return { id, name: input.name, owner, mascot, color, players: [] }
  }
}
