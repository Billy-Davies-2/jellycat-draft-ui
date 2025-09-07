// In-memory draft store (temporary until DB is wired)
// Shared types
export type Tier = 'S' | 'A' | 'B' | 'C'

export interface Player {
  id: string
  name: string
  position: string
  team: string
  points: number
  tier: Tier
  drafted: boolean
  draftedBy?: string
  image: string
}

export interface Team {
  id: string
  name: string
  owner: string
  mascot: string
  color: string
  players: Player[]
}

export interface ChatMessage {
  id: string
  ts: number
  type: 'system' | 'user'
  text: string
  emotes: Record<string, number>
}

function genId(prefix = 'id'): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 10)}`
}

const defaultPlayers: Player[] = [
  {
    id: '1',
    name: 'Bashful Bunny',
    position: 'CC',
    team: 'Woodland',
    points: 324,
    tier: 'S',
    drafted: false,
    image: '/jellycats/bashful-bunny.png',
  },
  {
    id: '2',
    name: 'Fuddlewuddle Lion',
    position: 'SS',
    team: 'Safari',
    points: 298,
    tier: 'S',
    drafted: false,
    image: '/jellycats/fuddlewuddle-lion.png',
  },
  {
    id: '3',
    name: 'Cordy Roy Elephant',
    position: 'HH',
    team: 'Safari',
    points: 287,
    tier: 'S',
    drafted: false,
    image: '/jellycats/cordy-roy-elephant.png',
  },
  {
    id: '4',
    name: 'Blossom Tulip Bunny',
    position: 'CH',
    team: 'Garden',
    points: 251,
    tier: 'A',
    drafted: false,
    image: '/jellycats/blossom-tulip-bunny.png',
  },
  {
    id: '5',
    name: 'Amuseable Avocado',
    position: 'CC',
    team: 'Kitchen',
    points: 312,
    tier: 'S',
    drafted: false,
    image: '/jellycats/amuseable-avocado.png',
  },
  {
    id: '6',
    name: 'Octopus Ollie',
    position: 'SS',
    team: 'Ocean',
    points: 276,
    tier: 'A',
    drafted: false,
    image: '/jellycats/octopus-ollie.png',
  },
  {
    id: '7',
    name: 'Jellycat Dragon',
    position: 'HH',
    team: 'Fantasy',
    points: 268,
    tier: 'A',
    drafted: false,
    image: '/jellycats/jellycat-dragon.png',
  },
  {
    id: '8',
    name: 'Bashful Lamb',
    position: 'CH',
    team: 'Farm',
    points: 245,
    tier: 'A',
    drafted: false,
    image: '/jellycats/bashful-lamb.png',
  },
  {
    id: '9',
    name: 'Amuseable Pineapple',
    position: 'CC',
    team: 'Tropical',
    points: 289,
    tier: 'S',
    drafted: false,
    image: '/jellycats/amuseable-pineapple.png',
  },
  {
    id: '10',
    name: 'Cordy Roy Fox',
    position: 'SS',
    team: 'Woodland',
    points: 234,
    tier: 'A',
    drafted: false,
    image: '/jellycats/cordy-roy-fox.png',
  },
  {
    id: '11',
    name: 'Blossom Peach Bunny',
    position: 'HH',
    team: 'Garden',
    points: 256,
    tier: 'A',
    drafted: false,
    image: '/jellycats/blossom-peach-bunny.png',
  },
  {
    id: '12',
    name: 'Amuseable Taco',
    position: 'CH',
    team: 'Kitchen',
    points: 267,
    tier: 'A',
    drafted: false,
    image: '/jellycats/amuseable-taco.png',
  },
  {
    id: '13',
    name: 'Bashful Unicorn',
    position: 'CC',
    team: 'Fantasy',
    points: 278,
    tier: 'A',
    drafted: false,
    image: '/jellycats/bashful-unicorn.png',
  },
  {
    id: '14',
    name: 'Jellycat Penguin',
    position: 'SS',
    team: 'Arctic',
    points: 243,
    tier: 'B',
    drafted: false,
    image: '/jellycats/jellycat-penguin.png',
  },
  {
    id: '15',
    name: 'Amuseable Moon',
    position: 'HH',
    team: 'Space',
    points: 229,
    tier: 'B',
    drafted: false,
    image: '/jellycats/amuseable-moon.png',
  },
  {
    id: '16',
    name: 'Cordy Roy Pig',
    position: 'CH',
    team: 'Farm',
    points: 241,
    tier: 'B',
    drafted: false,
    image: '/jellycats/cordy-roy-pig.png',
  },
  {
    id: '17',
    name: 'Bashful Tiger',
    position: 'SS',
    team: 'Safari',
    points: 235,
    tier: 'B',
    drafted: false,
    image: '/jellycats/bashful-tiger.png',
  },
  {
    id: '18',
    name: 'Amuseable Donut',
    position: 'CC',
    team: 'Kitchen',
    points: 228,
    tier: 'B',
    drafted: false,
    image: '/jellycats/amuseable-donut.png',
  },
]

const defaultTeams: Team[] = [
  {
    id: '1',
    name: 'Fluffy Foxes',
    owner: 'Sarah',
    mascot: '🦊',
    color: 'bg-orange-100 border-orange-300',
    players: [],
  },
  {
    id: '2',
    name: 'Cuddly Bears',
    owner: 'Mike',
    mascot: '🐻',
    color: 'bg-amber-100 border-amber-300',
    players: [],
  },
  {
    id: '3',
    name: 'Snuggly Bunnies',
    owner: 'Emma',
    mascot: '🐰',
    color: 'bg-pink-100 border-pink-300',
    players: [],
  },
  {
    id: '4',
    name: 'Cozy Cats',
    owner: 'Alex',
    mascot: '🐱',
    color: 'bg-purple-100 border-purple-300',
    players: [],
  },
  {
    id: '5',
    name: 'Soft Sheep',
    owner: 'Jordan',
    mascot: '🐑',
    color: 'bg-blue-100 border-blue-300',
    players: [],
  },
  {
    id: '6',
    name: 'Gentle Giraffes',
    owner: 'Taylor',
    mascot: '🦒',
    color: 'bg-yellow-100 border-yellow-300',
    players: [],
  },
]

class DraftStore {
  private _players: Player[]
  private _teams: Team[]
  private _chat: ChatMessage[]
  // Track which users reacted (message -> emote -> set(userId)) to prevent duplicate increments
  private _reactionUsers: Map<string, Map<string, Set<string>>> = new Map()

  constructor() {
    this._players = structuredClone(defaultPlayers)
    this._teams = structuredClone(defaultTeams)
    this._chat = []
    // Seed some fun messages in dev when started under `bun dev`
    if (process.env.NODE_ENV === 'development') {
      const msgs = [
        'Welcome to the Jellycat Draft! 🎉',
        'Tip: Click a Jellycat card to draft it!',
        'Who will snag Bashful Bunny first? 🐰',
      ]
      for (const m of msgs) this.addChatMessage(m, 'system')
      // light mock chatter every 20s
      const lines = [
        'What a pick! 😮',
        'Cuddle points through the roof! 📈',
        'Team chemistry looking great ✨',
      ]
      // best-effort interval, ignore if serverless instance freezes
      setInterval(() => {
        const line = lines[Math.floor(Math.random() * lines.length)]
        try {
          this.addChatMessage(line, 'user')
        } catch {}
      }, 20000).unref?.()
    }
  }

  reset() {
    this._players = structuredClone(defaultPlayers)
    this._teams = structuredClone(defaultTeams)
    this._chat = []
  }

  getState() {
    return {
      players: this._players,
      teams: this._teams,
      chat: this._chat,
    }
  }

  addTeam(input: { name: string; owner?: string; mascot?: string; color?: string }) {
    const mascots = ['🦊', '🐻', '🐰', '🐱', '🐑', '🦒', '🐨', '🦁', '🐼', '🦄', '🐯', '🐶']
    const colors = [
      'bg-orange-100 border-orange-300',
      'bg-amber-100 border-amber-300',
      'bg-pink-100 border-pink-300',
      'bg-purple-100 border-purple-300',
      'bg-blue-100 border-blue-300',
      'bg-yellow-100 border-yellow-300',
      'bg-green-100 border-green-300',
    ]
    const id = genId('team')
    const team: Team = {
      id,
      name: input.name,
      owner: input.owner || 'Anonymous',
      mascot: input.mascot || mascots[this._teams.length % mascots.length],
      color: input.color || colors[this._teams.length % colors.length],
      players: [],
    }
    this._teams.push(team)
    this.addChatMessage(
      `New team joined the draft: ${team.mascot} ${team.name} (Owner: ${team.owner})`,
      'system',
    )
    return team
  }

  addPlayer(input: Omit<Player, 'id' | 'drafted'> & { id?: string; drafted?: boolean }) {
    const id = input.id ?? genId('player')
    const player: Player = {
      drafted: false,
      ...input,
      id,
    }
    this._players.push(player)
    return player
  }

  setPlayerPoints(id: string, points: number) {
    const p = this._players.find((pl) => pl.id === id)
    if (!p) throw new Error('Player not found')
    p.points = points
    // Update any drafted copy inside team rosters too
    for (const t of this._teams) {
      const tp = t.players.find((pl) => pl.id === id)
      if (tp) tp.points = points
    }
    return p
  }

  reorderTeams(order: string[]) {
    const idToTeam = new Map(this._teams.map((t) => [t.id, t]))
    const reordered: Team[] = []
    for (const id of order) {
      const team = idToTeam.get(id)
      if (team) reordered.push(team)
    }
    // Append any missing
    for (const t of this._teams) {
      if (!reordered.includes(t)) reordered.push(t)
    }
    this._teams = reordered
    return this._teams
  }

  draftPlayer(playerId: string, teamId: string) {
    const player = this._players.find((p) => p.id === playerId)
    const team = this._teams.find((t) => t.id === teamId)
    if (!player) throw new Error('Player not found')
    if (!team) throw new Error('Team not found')
    if (player.drafted) throw new Error('Player already drafted')
    player.drafted = true
    player.draftedBy = team.name
    team.players.push({ ...player })
    // System chat message
    this.addChatMessage(
      `${team.mascot} ${team.name} drafted ${player.name} (${player.team} • ${player.position})`,
      'system',
    )
    return { player, team }
  }

  addChatMessage(text: string, type: 'system' | 'user' = 'user') {
    const msg: ChatMessage = { id: genId('msg'), ts: Date.now(), type, text, emotes: {} }
    this._chat.push(msg)
    return msg
  }

  addReaction(messageId: string, emote: string, userId?: string) {
    const m = this._chat.find((x) => x.id === messageId)
    if (!m) throw new Error('Message not found')
    const uid = userId || 'anon'
    let emoteMap = this._reactionUsers.get(messageId)
    if (!emoteMap) {
      emoteMap = new Map()
      this._reactionUsers.set(messageId, emoteMap)
    }
    let users = emoteMap.get(emote)
    if (!users) {
      users = new Set()
      emoteMap.set(emote, users)
    }
    if (users.has(uid)) return m // already reacted; no increment
    users.add(uid)
    m.emotes[emote] = (m.emotes[emote] || 0) + 1
    return m
  }
}

// Singleton store instance
export const draftStore = new DraftStore()
