// Next.js instrumentation hook (runs on server start & during build)
export async function register() {
  if (process.env.NODE_ENV === 'development') {
    await import('./lib/init-dev-db')
  }
}
