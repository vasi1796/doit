import { describe, it, expect } from 'vitest'
import { cursorSeedFrom } from '../db/initial-sync'

describe('cursorSeedFrom', () => {
  it('seeds one ms before the newest server row, ignoring missing and invalid timestamps', () => {
    const newest = Date.parse('2026-08-01T10:00:00Z')
    const seed = cursorSeedFrom([
      '2026-08-01T09:00:00Z',
      '2026-08-01T10:00:00Z',
      undefined,
      'not-a-date',
    ])
    // Server-derived, not device-clock-derived: a fast local clock must not
    // be able to push the cursor past events the server has yet to deliver.
    expect(seed).toBe(newest - 1)
  })

  it('seeds zero for an empty account so the first pull replays the whole log', () => {
    expect(cursorSeedFrom([])).toBe(0)
    expect(cursorSeedFrom([undefined, 'garbage'])).toBe(0)
  })
})
