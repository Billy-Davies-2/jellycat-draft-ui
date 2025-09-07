import { initTRPC } from '@trpc/server'
import { z } from 'zod'
import { getDAL } from './db'
import { pubsub } from './pubsub'
import { observable } from '@trpc/server/observable'

export interface Context {
  isAdmin: boolean
}
export const tctx = initTRPC.context<Context>()
const t = tctx.create()

function secureProcedure() {
  return t.procedure.use(({ ctx, next }) => {
    if (!ctx.isAdmin) {
      throw new Error('UNAUTHORIZED')
    }
    return next()
  })
}

export const appRouter = t.router({
  debug: t.procedure.query(() => {
    const dal = getDAL() as any
    const driver =
      process.env.DB_DRIVER ||
      process.env.NEXT_PUBLIC_DB_DRIVER ||
      (process.env.NODE_ENV === 'development' ? 'mega-mock' : 'memory')
    return {
      driver,
      dalClass: dal?.constructor?.name,
    }
  }),
  // server-sent events via websocket/subscription
  events: t.procedure.subscription(() => {
    return observable<{ type: string; payload?: any }>((emit) => {
      const off = pubsub.onEvent((e) => emit.next(e as any))
      return () => off()
    })
  }),

  chat: t.router({
    list: t.procedure.query(async () => {
      const dal = getDAL()
      const state = await dal.getState()
      return state.chat
    }),
    send: t.procedure.input(z.object({ text: z.string().min(1) })).mutation(async ({ input }) => {
      const dal = getDAL()
      const msg = await dal.addChatMessage(input.text, 'user')
      pubsub.emitEvent({ type: 'chat:add', payload: { id: msg.id } })
      return msg
    }),
    react: t.procedure
      .input(
        z.object({ messageId: z.string(), emote: z.string().min(1), user: z.string().optional() }),
      )
      .mutation(async ({ input }) => {
        const dal = getDAL()
        const msg = await dal.addReaction(input.messageId, input.emote, input.user)
        pubsub.emitEvent({ type: 'chat:react', payload: { id: msg.id, emote: input.emote } })
        return msg
      }),
  }),

  draft: t.router({
    state: t.procedure.query(async () => {
      const dal = getDAL()
      const s = await dal.getState()
      if (process.env.NODE_ENV === 'development' && !process.env.QUIET_DB_INIT) {
        try {
          console.log(`[trpc][draft.state] teams=${s.teams?.length} players=${s.players?.length}`)
        } catch {}
      }
      return s
    }),
    reset: t.procedure.mutation(async () => {
      const dal = getDAL()
      await dal.reset()
      pubsub.emitEvent({ type: 'draft:pick', payload: { playerId: '', teamId: '' } })
      return { ok: true }
    }),
    pick: t.procedure
      .input(z.object({ playerId: z.string(), teamId: z.string() }))
      .mutation(async ({ input }) => {
        const dal = getDAL()
        const res = await dal.draftPlayer(input.playerId, input.teamId)
        pubsub.emitEvent({
          type: 'draft:pick',
          payload: { playerId: input.playerId, teamId: input.teamId },
        })
        return res
      }),
  }),

  teams: t.router({
    list: t.procedure.query(async () => {
      const dal = getDAL()
      const state = await dal.getState()
      return state.teams
    }),
    add: t.procedure
      .input(
        z.object({
          name: z.string().min(1),
          owner: z.string().optional(),
          mascot: z.string().optional(),
          color: z.string().optional(),
        }),
      )
      .mutation(async ({ input }) => {
        const dal = getDAL()
        const team = await dal.addTeam(input)
        pubsub.emitEvent({ type: 'teams:add', payload: { id: team.id } })
        return team
      }),
    reorder: t.procedure
      .input(z.object({ order: z.array(z.string()) }))
      .mutation(async ({ input }) => {
        const dal = getDAL()
        const teams = await dal.reorderTeams(input.order)
        pubsub.emitEvent({ type: 'teams:reorder' })
        return teams
      }),
  }),

  players: t.router({
    add: t.procedure
      .input(
        z.object({
          name: z.string(),
          position: z.string(),
          team: z.string(),
          points: z.number(),
          tier: z.enum(['S', 'A', 'B', 'C']),
          image: z.string().optional(),
        }),
      )
      .mutation(async ({ input }) => {
        const dal = getDAL()
        const player = await dal.addPlayer({ ...input, drafted: false })
        pubsub.emitEvent({
          type: 'players:updatePoints',
          payload: { id: player.id, points: player.points },
        })
        return player
      }),
    setPoints: t.procedure
      .input(z.object({ id: z.string(), points: z.number() }))
      .mutation(async ({ input }) => {
        const dal = getDAL()
        const p = await dal.setPlayerPoints(input.id, input.points)
        pubsub.emitEvent({ type: 'players:updatePoints', payload: { id: p.id, points: p.points } })
        return p
      }),
    profile: t.procedure.input(z.object({ id: z.string() })).query(async ({ input }) => {
      const dal = getDAL()
      const state = await dal.getState()
      const player = state.players.find((p) => String(p.id) === input.id)
      if (!player) throw new Error('NOT_FOUND')
      // Derive mock metrics (would come from ClickHouse aggregates in real setup)
      // Use points value + id hash to generate stable pseudo-randoms for demo
      const seed =
        player.points +
        Array.from(player.id).reduce<number>((a, c) => a + (c as string).charCodeAt(0), 0)
      function norm(x: number) {
        return Math.max(0, Math.min(100, x))
      }
      const consistency = norm((seed * 13) % 101)
      const popularity = norm((seed * 29) % 101)
      const efficiency = norm((seed * 47) % 101)
      const trendDelta = (((seed % 15) - 7) / 7).toFixed(2) // -1 to +1-ish
      return { ...player, metrics: { consistency, popularity, efficiency, trendDelta } }
    }),
  }),
  admin: t.router({
    login: t.procedure
      .input(z.object({ username: z.string(), password: z.string() }))
      .mutation(async ({ input }) => {
        const ok =
          process.env.NODE_ENV === 'development' &&
          input.username === 'admin' &&
          input.password === 'password'
        if (!ok) throw new Error('INVALID_CREDENTIALS')
        // Return a token caller can set as cookie/localStorage (simple dev mode token)
        const token = 'dev-admin'
        return { token }
      }),
    addJellycat: secureProcedure()
      .input(
        z.object({
          name: z.string(),
          position: z.string(),
          team: z.string(),
          points: z.number().min(0),
          tier: z.enum(['S', 'A', 'B', 'C']),
          image: z.string().optional(),
        }),
      )
      .mutation(async ({ input }) => {
        const dal = getDAL()
        const player = await dal.addPlayer({ ...input, drafted: false })
        pubsub.emitEvent({
          type: 'players:updatePoints',
          payload: { id: player.id, points: player.points },
        })
        return player
      }),
  }),
})

export type AppRouter = typeof appRouter
