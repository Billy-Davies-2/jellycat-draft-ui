import { applyWSSHandler } from '@trpc/server/adapters/ws'
import { WebSocketServer } from 'ws'

import { appRouter } from './trpc-router'

declare global {
  var __TRPC_WS__: { wss?: WebSocketServer; port: number } | undefined
}

const isDev = process.env.NODE_ENV === 'development'
const port = Number(process.env.TRPC_WS_PORT || process.env.NEXT_PUBLIC_TRPC_WS_PORT || 3001)

if (isDev && typeof window === 'undefined') {
  try {
    if (!globalThis.__TRPC_WS__?.wss || globalThis.__TRPC_WS__?.port !== port) {
      // Close old one if port mismatch (rare on hot swap)
      try {
        globalThis.__TRPC_WS__?.wss?.close()
      } catch {}
      const wss = new WebSocketServer({ port })
      applyWSSHandler({ wss, router: appRouter, createContext: () => ({ isAdmin: false }) })
      globalThis.__TRPC_WS__ = { wss, port }
      if (!process.env.QUIET_WS && !globalThis.__TRPC_WS__?.wss)
        console.log(`[tRPC] WS server listening on ws://localhost:${port}`)
      const cleanup = () => {
        try {
          wss.close()
        } catch {}
        globalThis.__TRPC_WS__ = undefined
      }
      for (const sig of ['exit', 'SIGINT', 'SIGTERM']) process.on(sig as any, cleanup)
    }
  } catch (e: any) {
    const msg = String(e?.message || '')
    if (e?.code === 'EADDRINUSE' || msg.includes('in use')) {
      if (!process.env.QUIET_WS)
        console.warn(`[tRPC] WS port ${port} already in use; reusing existing listener`)
    } else {
      if (!process.env.QUIET_WS) console.warn('[tRPC] WS server init error suppressed:', e)
    }
  }
}
