import { describe, it, expect, mock, beforeEach } from 'bun:test'
import { IDraftDAL } from '../../lib/db'

// We import the module dynamically to swap the pg Pool with a mock

function createMockPool() {
  const data: any = {
    teams: [],
    players: [],
    team_players: [],
    chat_messages: [],
    chat_reactions: [],
  }

  const handlers: Record<string, (params: any[]) => any> = {
    // SELECTS
    'SELECT id, name, owner, mascot, color, sort_order FROM teams ORDER BY sort_order, name':
      () => ({
        rows: [...data.teams].sort(
          (a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name),
        ),
      }),
    'SELECT id, name, position, team, points, tier, drafted, image FROM players ORDER BY tier, points DESC':
      () => ({ rows: [...data.players] }),
    'SELECT id, ts, type, text FROM chat_messages ORDER BY ts ASC': () => ({
      rows: [...data.chat_messages].sort((a, b) => a.ts - b.ts),
    }),
    'SELECT id, ts, type, text FROM chat_messages WHERE id=$1': ([id]) => ({
      rows: data.chat_messages.filter((m: any) => String(m.id) === String(id)),
    }),
    'SELECT message_id, emote, count FROM chat_reactions': () => ({
      rows: [...data.chat_reactions],
    }),
    'SELECT message_id, emote, count FROM chat_reactions WHERE message_id=$1': ([id]) => ({
      rows: data.chat_reactions.filter((r: any) => String(r.message_id) === String(id)),
    }),
    'SELECT team_id, player_id FROM team_players': () => ({ rows: [...data.team_players] }),
    'SELECT id, name, position, team, points, tier, drafted, image FROM players WHERE id=$1': ([
      id,
    ]) => ({ rows: data.players.filter((p: any) => String(p.id) === String(id)) }),
    'SELECT id, name, owner, mascot, color, sort_order FROM teams WHERE id=$1': ([id]) => ({
      rows: data.teams.filter((t: any) => String(t.id) === String(id)),
    }),
    'SELECT COALESCE(MAX(sort_order), -1) + 1 as next FROM teams': () => ({
      rows: [
        { next: data.teams.reduce((m: number, t: any) => Math.max(m, t.sort_order ?? -1), -1) + 1 },
      ],
    }),

    // INSERT/UPDATE/DELETE
    'INSERT INTO teams (id, name, owner, mascot, color, sort_order) VALUES ($1,$2,$3,$4,$5,$6)': ([
      id,
      name,
      owner,
      mascot,
      color,
      sort_order,
    ]) => {
      data.teams.push({ id, name, owner, mascot, color, sort_order })
      return { rowCount: 1 }
    },
    'INSERT INTO teams (id, name, owner, mascot, color, sort_order) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, owner=EXCLUDED.owner, mascot=EXCLUDED.mascot, color=EXCLUDED.color, sort_order=EXCLUDED.sort_order':
      ([id, name, owner, mascot, color, sort_order]) => {
        const i = data.teams.findIndex((t: any) => String(t.id) === String(id))
        if (i >= 0) data.teams[i] = { id, name, owner, mascot, color, sort_order }
        else data.teams.push({ id, name, owner, mascot, color, sort_order })
        return { rowCount: 1 }
      },
    'INSERT INTO players (id, name, position, team, points, tier, drafted, image) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)':
      ([id, name, position, team, points, tier, drafted, image]) => {
        data.players.push({ id, name, position, team, points, tier, drafted, image })
        return { rowCount: 1 }
      },
    'INSERT INTO players (id, name, position, team, points, tier, drafted, image) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, position=EXCLUDED.position, team=EXCLUDED.team, points=EXCLUDED.points, tier=EXCLUDED.tier, drafted=EXCLUDED.drafted, image=EXCLUDED.image':
      ([id, name, position, team, points, tier, drafted, image]) => {
        const i = data.players.findIndex((p: any) => String(p.id) === String(id))
        const row = { id, name, position, team, points, tier, drafted, image }
        if (i >= 0) data.players[i] = row
        else data.players.push(row)
        return { rowCount: 1 }
      },
    'UPDATE players SET points=$2 WHERE id=$1': ([id, points]) => {
      const p = data.players.find((x: any) => String(x.id) === String(id))
      if (p) p.points = points
      return { rowCount: 1 }
    },
    'UPDATE teams SET sort_order=$2 WHERE id=$1': ([id, ord]) => {
      const t = data.teams.find((x: any) => String(x.id) === String(id))
      if (t) t.sort_order = ord
      return { rowCount: 1 }
    },
    'INSERT INTO team_players (team_id, player_id) VALUES ($1,$2) ON CONFLICT DO NOTHING': ([
      team_id,
      player_id,
    ]) => {
      if (!data.team_players.find((x: any) => x.team_id === team_id && x.player_id === player_id)) {
        data.team_players.push({ team_id, player_id })
      }
      const p = data.players.find((x: any) => String(x.id) === String(player_id))
      if (p) p.drafted = true
      return { rowCount: 1 }
    },
    'UPDATE players SET drafted=TRUE WHERE id=$1': () => ({ rowCount: 1 }),
    'INSERT INTO chat_messages (id, ts, type, text) VALUES ($1,$2,$3,$4)': ([
      id,
      ts,
      type,
      text,
    ]) => {
      data.chat_messages.push({ id, ts, type, text })
      return { rowCount: 1 }
    },
    'INSERT INTO chat_reactions (message_id, emote, count) VALUES ($1,$2,1) ON CONFLICT (message_id, emote) DO UPDATE SET count = chat_reactions.count + 1':
      ([message_id, emote]) => {
        const r = data.chat_reactions.find(
          (x: any) => x.message_id === message_id && x.emote === emote,
        )
        if (r) r.count += 1
        else data.chat_reactions.push({ message_id, emote, count: 1 })
        return { rowCount: 1 }
      },
    'DELETE FROM chat_reactions': () => {
      data.chat_reactions = []
      return { rowCount: 1 }
    },
    'DELETE FROM chat_messages': () => {
      data.chat_messages = []
      return { rowCount: 1 }
    },
    'DELETE FROM team_players': () => {
      data.team_players = []
      return { rowCount: 1 }
    },
    'DELETE FROM players': () => {
      data.players = []
      return { rowCount: 1 }
    },
    'DELETE FROM teams': () => {
      data.teams = []
      return { rowCount: 1 }
    },
  }

  const normalize = (sql: string) =>
    sql
      .replace(/\s+/g, ' ') // collapse whitespace
      .replace(/\s*,\s*/g, ', ') // single space after commas
      .replace(/\s*\(\s*/g, '(') // trim inside (
      .replace(/\s*\)\s*/g, ')') // trim inside )
      .trim()
      .toLowerCase() // case-insensitive matching

  // build normalized handler map
  const normHandlers: Record<string, (params: any[]) => any> = {}
  for (const k of Object.keys(handlers)) {
    normHandlers[normalize(k)] = handlers[k]
  }

  const query = mock((sql: string, params?: any[]) => {
    const key = normalize(sql)
    const handler = normHandlers[key]
    if (!handler) {
      // Ignore CREATE TABLE and ON CONFLICT seeds
      if (sql.startsWith('CREATE TABLE')) return { rowCount: 0, rows: [] }
      // Fallback for selects after insert
      if (
        sql.startsWith(
          'SELECT id, name, position, team, points, tier, drafted, image FROM players WHERE id=',
        )
      ) {
        const id = params?.[0]
        return handlers[
          'SELECT id, name, position, team, points, tier, drafted, image FROM players WHERE id=$1'
        ]([id])
      }
      if (sql.startsWith('SELECT id, ts, type, text FROM chat_messages WHERE id=')) {
        const id = params?.[0]
        return handlers['SELECT id, ts, type, text FROM chat_messages WHERE id=$1']([id])
      }
      throw new Error('Unhandled SQL: ' + sql)
    }
    return handler(params || [])
  })

  return { query, _data: data }
}

describe('PostgresDAL', () => {
  let dal: IDraftDAL
  let pool: any

  beforeEach(async () => {
    pool = createMockPool()
    const { PostgresDAL }: any = await import('../../lib/dal/postgres')
    dal = new PostgresDAL(pool)
    await dal.reset() // seeds from draftStore defaults
  })

  it('lists state with seeded teams and players', async () => {
    const s = await dal.getState()
    expect(s.teams.length).toBeGreaterThan(0)
    expect(s.players.length).toBeGreaterThan(0)
  })

  it('adds a team, reorders, and persists', async () => {
    const team = await dal.addTeam({ name: 'Testers' })
    expect(team.name).toBe('Testers')
    const s1 = await dal.getState()
    const order = s1.teams.map((t) => t.id)
    // move last to first
    const last = order.pop()!
    order.unshift(last)
    const reordered = await dal.reorderTeams(order)
    expect(reordered[0].id).toBe(last)
  })

  it('adds a player and updates points', async () => {
    const p = await dal.addPlayer({
      name: 'Zaz',
      position: 'CC',
      team: 'Lab',
      points: 1,
      tier: 'C',
      image: '',
    })
    expect(p.name).toBe('Zaz')
    const p2 = await dal.setPlayerPoints(p.id, 42)
    expect(p2.points).toBe(42)
  })

  it('drafts a player and announces in chat via caller', async () => {
    const s = await dal.getState()
    const player = s.players.find((x) => !x.drafted)!
    const team = s.teams[0]
    const res = await dal.draftPlayer(String(player.id), String(team.id))
    expect(res.player.drafted).toBe(true)
    // add a system message manually to mimic router side effect is fine here
    await dal.addChatMessage(
      `${team.mascot} ${team.name} drafted ${player.name} (${player.team} • ${player.position})`,
      'system',
    )
    const s2 = await dal.getState()
    expect((s2.chat || []).length).toBeGreaterThan(0)
  })

  it('chat send and react', async () => {
    const m = await dal.addChatMessage('hello', 'user')
    const m2 = await dal.addReaction(m.id, '👍')
    expect(m2.emotes['👍']).toBe(1)
  })
})
