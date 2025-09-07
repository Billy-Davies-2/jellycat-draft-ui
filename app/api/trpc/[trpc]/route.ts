import { fetchRequestHandler } from '@trpc/server/adapters/fetch'

// Start WebSocket server in development for subscriptions
import '@/lib/trpc-ws-server'
import type { AppRouter } from '@/lib/trpc-router'
import { appRouter } from '@/lib/trpc-router'

function getIsAdmin(req: Request): boolean {
  if (process.env.NODE_ENV !== 'development') return false
  const cookie = req.headers.get('cookie') || ''
  return /admin_token=dev-admin/.test(cookie)
}

const handler = (req: Request) =>
  fetchRequestHandler<AppRouter>({
    endpoint: '/api/trpc',
    req,
    router: appRouter,
    createContext: () => ({ isAdmin: getIsAdmin(req) }),
  })

export { handler as GET, handler as POST }
