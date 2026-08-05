// TZ must be fixed before the first Date call in this worker so todayStr()
// resolves in a non-UTC zone. Kiritimati (UTC+14) maximises the window in
// which local "today" differs from the UTC day.
declare const process: { env: Record<string, string | undefined> }
process.env.TZ = 'Pacific/Kiritimati'

import { describe, it, expect, vi, afterEach } from 'vitest'
import { todayStr } from '../utils/date'

describe('todayStr', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('uses the local calendar day, not the UTC day', () => {
    // 20:00 UTC = 10:00 the NEXT day in UTC+14 — a toISOString-based
    // implementation would report the UTC day and mis-bucket Today/Overdue.
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-01T20:00:00Z'))

    expect(new Date().toISOString().split('T')[0]).toBe('2026-08-01')
    expect(todayStr()).toBe('2026-08-02')
  })
})
