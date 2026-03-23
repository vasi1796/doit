import { describe, it, expect } from 'vitest'
import { mergeLWW } from '../crdt/lww'
import type { HLCTimestamp } from '../hlc/hlc'

describe('mergeLWW', () => {
  const cases: {
    name: string
    localVal: string
    localHLC: HLCTimestamp
    remoteVal: string
    remoteHLC: HLCTimestamp
    expectedVal: string
    expectedHLC: HLCTimestamp
  }[] = [
    {
      name: 'remote wins when remote time is later',
      localVal: 'local',
      localHLC: { time: 100, counter: 0 },
      remoteVal: 'remote',
      remoteHLC: { time: 200, counter: 0 },
      expectedVal: 'remote',
      expectedHLC: { time: 200, counter: 0 },
    },
    {
      name: 'local wins when local time is later',
      localVal: 'local',
      localHLC: { time: 200, counter: 0 },
      remoteVal: 'remote',
      remoteHLC: { time: 100, counter: 0 },
      expectedVal: 'local',
      expectedHLC: { time: 200, counter: 0 },
    },
    {
      name: 'remote wins when times equal but remote counter is higher',
      localVal: 'local',
      localHLC: { time: 100, counter: 1 },
      remoteVal: 'remote',
      remoteHLC: { time: 100, counter: 5 },
      expectedVal: 'remote',
      expectedHLC: { time: 100, counter: 5 },
    },
    {
      name: 'local wins when times equal but local counter is higher',
      localVal: 'local',
      localHLC: { time: 100, counter: 5 },
      remoteVal: 'remote',
      remoteHLC: { time: 100, counter: 1 },
      expectedVal: 'local',
      expectedHLC: { time: 100, counter: 5 },
    },
    {
      name: 'remote wins on exact tie (deterministic tiebreaker)',
      localVal: 'local',
      localHLC: { time: 100, counter: 3 },
      remoteVal: 'remote',
      remoteHLC: { time: 100, counter: 3 },
      expectedVal: 'remote',
      expectedHLC: { time: 100, counter: 3 },
    },
  ]

  for (const tc of cases) {
    it(tc.name, () => {
      const [val, hlc] = mergeLWW(tc.localVal, tc.localHLC, tc.remoteVal, tc.remoteHLC)
      expect(val).toBe(tc.expectedVal)
      expect(hlc).toEqual(tc.expectedHLC)
    })
  }

  it('works with non-string types (numbers)', () => {
    const [val, hlc] = mergeLWW(10, { time: 50, counter: 0 }, 20, { time: 100, counter: 0 })
    expect(val).toBe(20)
    expect(hlc).toEqual({ time: 100, counter: 0 })
  })
})
