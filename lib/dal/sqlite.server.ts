// Isolated SQLite DAL (server-only) kept minimal to reduce bundler surface.
import { draftStore } from '../draft-store'

export class SqliteDAL {
  db: any
  private schemaReady = false
  private seeded = false
  constructor(filename?: string) {
    const file = filename || process.env.SQLITE_FILE || 'dev.sqlite'
    const { Database } = require('bun:sqlite')
    this.db = new Database(file)
  }
  private ensureSchema() {
    if (this.schemaReady) return
    this.db.exec(
      `BEGIN;CREATE TABLE IF NOT EXISTS teams (id TEXT PRIMARY KEY,name TEXT NOT NULL,owner TEXT NOT NULL,mascot TEXT NOT NULL,color TEXT NOT NULL,sort_order INTEGER NOT NULL DEFAULT 0);CREATE TABLE IF NOT EXISTS players (id TEXT PRIMARY KEY,name TEXT NOT NULL,position TEXT NOT NULL,team TEXT NOT NULL,points INTEGER NOT NULL,tier TEXT NOT NULL,drafted INTEGER NOT NULL DEFAULT 0,image TEXT NOT NULL);CREATE TABLE IF NOT EXISTS team_players (team_id TEXT NOT NULL,player_id TEXT NOT NULL,PRIMARY KEY (team_id, player_id));CREATE TABLE IF NOT EXISTS chat_messages (id TEXT PRIMARY KEY,ts INTEGER NOT NULL,type TEXT NOT NULL,text TEXT NOT NULL);CREATE TABLE IF NOT EXISTS chat_reactions (message_id TEXT NOT NULL,emote TEXT NOT NULL,count INTEGER NOT NULL DEFAULT 0,PRIMARY KEY (message_id, emote));CREATE TABLE IF NOT EXISTS chat_reaction_users (message_id TEXT NOT NULL,emote TEXT NOT NULL,user_id TEXT NOT NULL,PRIMARY KEY (message_id, emote, user_id));COMMIT;`,
    )
    this.schemaReady = true
  }
  private seedIfEmpty() {
    if (this.seeded) return
    try {
      const teamCount = (this.db.query('SELECT COUNT(1) as c FROM teams').get() as any)?.c || 0
      const playerCount = (this.db.query('SELECT COUNT(1) as c FROM players').get() as any)?.c || 0
      if (teamCount === 0 && playerCount === 0) {
        draftStore.reset()
        const snap = draftStore.getState()
        let i = 0
        const insTeam = this.db.query(
          'INSERT INTO teams (id,name,owner,mascot,color,sort_order) VALUES (?,?,?,?,?,?)',
        )
        for (const t of snap.teams)
          insTeam.run(String(t.id), t.name, t.owner, t.mascot, t.color, i++)
        const insPlayer = this.db.query(
          'INSERT INTO players (id,name,position,team,points,tier,drafted,image) VALUES (?,?,?,?,?,?,?,?)',
        )
        for (const p of snap.players)
          insPlayer.run(String(p.id), p.name, p.position, p.team, p.points, p.tier, 0, p.image)
        const now = Date.now()
        const seedMsgs = [
          'Welcome to the Jellycat Draft! 🎉',
          'Create a team to join the lobby.',
          'Use set points to test live updates.',
        ]
        const insMsg = this.db.query('INSERT INTO chat_messages (id,ts,type,text) VALUES (?,?,?,?)')
        seedMsgs.forEach((m) =>
          insMsg.run(`msg_${Math.random().toString(36).slice(2, 10)}`, now, 'system', m),
        )
        try {
          const seedMod = require('../sqlite-seed.server')
          seedMod?.seedJellycats?.(this.db, insPlayer)
        } catch {}
      }
      this.seeded = true
    } catch {
      /* ignore */
    }
  }
  private genId(prefix = 'id') {
    return `${prefix}_${Math.random().toString(36).slice(2, 10)}`
  }
  async getState() {
    this.ensureSchema()
    this.seedIfEmpty()
    const teams = this.db
      .query('SELECT id,name,owner,mascot,color,sort_order FROM teams ORDER BY sort_order,name')
      .all()
    const players = this.db
      .query(
        'SELECT id,name,position,team,points,tier,drafted,image FROM players ORDER BY tier,points DESC',
      )
      .all()
      .map((p: any) => ({ ...p, drafted: !!p.drafted }))
    const roster = this.db.query('SELECT team_id,player_id FROM team_players').all()
    const teamMap: Record<string, any> = {}
    for (const t of teams) teamMap[String(t.id)] = { ...t, players: [] }
    for (const r of roster) {
      const p = players.find((pl: any) => String(pl.id) === String(r.player_id))
      const team = teamMap[String(r.team_id)]
      if (p && team) {
        const copy = { ...p, drafted: true, draftedBy: team.name }
        team.players.push(copy)
        p.drafted = true
        p.draftedBy = team.name
      }
    }
    const chatRows = this.db
      .query('SELECT id,ts,type,text FROM chat_messages ORDER BY ts ASC')
      .all()
    const reactRows = this.db.query('SELECT message_id,emote,count FROM chat_reactions').all()
    const reactsMap = new Map<string, Record<string, number>>()
    for (const r of reactRows) {
      const key = String((r as any).message_id)
      const emotes = reactsMap.get(key) || {}
      emotes[String((r as any).emote)] = Number((r as any).count)
      reactsMap.set(key, emotes)
    }
    const chat = chatRows.map((m: any) => ({ ...m, emotes: reactsMap.get(String(m.id)) || {} }))
    return { players, teams: teams.map((t: any) => teamMap[String(t.id)]), chat }
  }
  async reset() {
    this.ensureSchema()
    this.db.exec(
      'DELETE FROM chat_reactions;DELETE FROM chat_messages;DELETE FROM team_players;DELETE FROM players;DELETE FROM teams;',
    )
    draftStore.reset()
    const seed = draftStore.getState()
    let i = 0
    const insTeam = this.db.query(
      'INSERT INTO teams (id,name,owner,mascot,color,sort_order) VALUES (?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,owner=excluded.owner,mascot=excluded.mascot,color=excluded.color,sort_order=excluded.sort_order',
    )
    for (const t of seed.teams) insTeam.run(String(t.id), t.name, t.owner, t.mascot, t.color, i++)
    const insPlayer = this.db.query(
      'INSERT INTO players (id,name,position,team,points,tier,drafted,image) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,position=excluded.position,team=excluded.team,points=excluded.points,tier=excluded.tier,drafted=excluded.drafted,image=excluded.image',
    )
    for (const p of seed.players)
      insPlayer.run(String(p.id), p.name, p.position, p.team, p.points, p.tier, 0, p.image)
    try {
      const seedMod = require('../sqlite-seed.server')
      seedMod?.seedJellycats?.(this.db, insPlayer)
    } catch {}
  }
  async addPlayer(input: any) {
    this.ensureSchema()
    const id = input.id ?? this.genId('player')
    this.db
      .query(
        'INSERT INTO players (id,name,position,team,points,tier,drafted,image) VALUES (?,?,?,?,?,?,?,?)',
      )
      .run(
        String(id),
        input.name,
        input.position,
        input.team,
        Number(input.points),
        input.tier,
        input.drafted ? 1 : 0,
        input.image || '',
      )
    return this.db
      .query('SELECT id,name,position,team,points,tier,drafted,image FROM players WHERE id=?')
      .get(String(id))
  }
  async setPlayerPoints(id: string, points: number) {
    this.ensureSchema()
    this.db.query('UPDATE players SET points=? WHERE id=?').run(Number(points), String(id))
    return this.db
      .query('SELECT id,name,position,team,points,tier,drafted,image FROM players WHERE id=?')
      .get(String(id))
  }
  async reorderTeams(order: string[]) {
    this.ensureSchema()
    const stmt = this.db.query('UPDATE teams SET sort_order=? WHERE id=?')
    order.forEach((id, idx) => stmt.run(idx, String(id)))
    return this.db
      .query('SELECT id,name,owner,mascot,color,sort_order FROM teams ORDER BY sort_order,name')
      .all()
  }
  async draftPlayer(playerId: string, teamId: string) {
    this.ensureSchema()
    const player: any = this.db
      .query('SELECT id,name,position,team,points,tier,drafted,image FROM players WHERE id=?')
      .get(String(playerId))
    const team: any = this.db
      .query('SELECT id,name,owner,mascot,color,sort_order FROM teams WHERE id=?')
      .get(String(teamId))
    if (!player) throw new Error('Player not found')
    if (!team) throw new Error('Team not found')
    if (player.drafted) throw new Error('Player already drafted')
    this.db
      .query(
        'INSERT INTO team_players (team_id,player_id) VALUES (?,?) ON CONFLICT(team_id,player_id) DO NOTHING',
      )
      .run(String(teamId), String(playerId))
    this.db.query('UPDATE players SET drafted=1 WHERE id=?').run(String(playerId))
    const updated = { ...player, drafted: true, draftedBy: team.name }
    return { player: updated, team: { ...team, players: [updated] } }
  }
  async addChatMessage(text: string, type: 'system' | 'user' = 'user') {
    this.ensureSchema()
    const id = this.genId('msg')
    const ts = Date.now()
    this.db
      .query('INSERT INTO chat_messages (id,ts,type,text) VALUES (?,?,?,?)')
      .run(id, ts, type, text)
    return { id, ts, type, text, emotes: {} }
  }
  async addReaction(messageId: string, emote: string, userId?: string) {
    this.ensureSchema()
    const uid = userId || 'anon'
    try {
      this.db
        .query('INSERT INTO chat_reaction_users (message_id,emote,user_id) VALUES (?,?,?)')
        .run(String(messageId), String(emote), String(uid))
      this.db
        .query(
          'INSERT INTO chat_reactions (message_id,emote,count) VALUES (?,?,1) ON CONFLICT(message_id,emote) DO UPDATE SET count = count + 1',
        )
        .run(String(messageId), String(emote))
    } catch {
      /* conflict => already reacted */
    }
    const m: any = this.db
      .query('SELECT id,ts,type,text FROM chat_messages WHERE id=?')
      .get(String(messageId))
    const reacts = this.db
      .query('SELECT message_id,emote,count FROM chat_reactions WHERE message_id=?')
      .all(String(messageId))
    const emotes: Record<string, number> = {}
    reacts.forEach((r: any) => {
      emotes[String(r.emote)] = Number(r.count)
    })
    return { ...m, emotes }
  }
  async addTeam(input: { name: string; owner?: string; mascot?: string; color?: string }) {
    this.ensureSchema()
    const id = this.genId('team')
    const owner = input.owner || 'Anonymous'
    const mascot = input.mascot || '🧸'
    const color = input.color || 'bg-amber-100 border-amber-300'
    const nextOrder =
      (this.db.query('SELECT COALESCE(MAX(sort_order), -1)+1 AS next FROM teams').get() as any)
        ?.next || 0
    this.db
      .query('INSERT INTO teams (id,name,owner,mascot,color,sort_order) VALUES (?,?,?,?,?,?)')
      .run(String(id), input.name, owner, mascot, color, nextOrder)
    if (process.env.NODE_ENV === 'development' && !process.env.QUIET_DB_INIT) {
      try {
        const count = (this.db.query('SELECT COUNT(1) as c FROM teams').get() as any)?.c
        console.log(`[db][sqlite] addTeam -> total teams=${count}`)
      } catch {}
    }
    return { id, name: input.name, owner, mascot, color, players: [] }
  }
}
