import { describe, it, expect } from 'bun:test'
import { dbHealthDetails } from '../../lib/db'

// Simple unit-level validation of health details shape

describe('dbHealthDetails', () => {
  it('returns shape with driver and timestamp', async () => {
    process.env.DB_DRIVER = 'memory'
    const d = await dbHealthDetails()
    expect(d.ok).toBe(true)
    expect(typeof d.driver).toBe('string')
    expect(typeof d.timestamp).toBe('number')
    expect(typeof d.version).toBe('string')
    expect(d.uptimeMs).toBeGreaterThanOrEqual(0)
  })
})
