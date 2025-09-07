import { NextResponse } from 'next/server'

const headers = {
  'content-type': 'application/json',
  deprecation: 'true',
  link: '</api/trpc>; rel="service"',
}
const body = { error: 'REST endpoints removed. Use tRPC at /api/trpc.' }

export function POST() {
  return new NextResponse(JSON.stringify(body), { status: 410, headers })
}
