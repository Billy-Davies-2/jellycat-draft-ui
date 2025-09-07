import { NextResponse } from 'next/server'
import { dbHealthDetails } from '@/lib/db'

export async function GET() {
  const details = await dbHealthDetails()
  const { ok } = details
  const body = details
  return new NextResponse(JSON.stringify(body), {
    status: ok ? 200 : 503,
    headers: { 'content-type': 'application/json' },
  })
}
