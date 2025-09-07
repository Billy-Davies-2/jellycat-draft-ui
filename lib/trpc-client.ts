'use client'
import {
  createTRPCProxyClient,
  httpBatchLink,
  splitLink,
  createWSClient,
  wsLink,
} from '@trpc/client'
import type { AppRouter } from './trpc-router'

const http = httpBatchLink({ url: '/api/trpc' })

// In dev, use a WebSocket link for subscriptions
const ws =
  typeof window !== 'undefined' && process.env.NODE_ENV === 'development'
    ? wsLink({
        client: createWSClient({
          url: `ws://localhost:${process.env.NEXT_PUBLIC_TRPC_WS_PORT || process.env.TRPC_WS_PORT || 3001}`,
        }),
      })
    : null

export const trpc = createTRPCProxyClient<AppRouter>({
  links: ws
    ? [
        // route subscriptions over ws, rest over http
        splitLink({
          condition: (op) => op.type === 'subscription',
          true: ws,
          false: http,
        }),
      ]
    : [http],
})
