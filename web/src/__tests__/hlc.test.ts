import { describe, it, expect } from 'vitest'
import { HLCClock, compare } from '../hlc/hlc'

describe('compare', () => {
  const cases: {
    name: string
    a: { time: number; counter: number }
    b: { time: number; counter: number }
    expected: number
  }[] = [
    { name: 'a < b by time', a: { time: 100, counter: 0 }, b: { time: 200, counter: 0 }, expected: -1 },
    { name: 'a > b by time', a: { time: 200, counter: 0 }, b: { time: 100, counter: 0 }, expected: 1 },
    { name: 'a < b by counter', a: { time: 100, counter: 1 }, b: { time: 100, counter: 2 }, expected: -1 },
    { name: 'a > b by counter', a: { time: 100, counter: 5 }, b: { time: 100, counter: 3 }, expected: 1 },
    { name: 'equal timestamps', a: { time: 100, counter: 3 }, b: { time: 100, counter: 3 }, expected: 0 },
  ]

  for (const tc of cases) {
    it(tc.name, () => {
      expect(compare(tc.a, tc.b)).toBe(tc.expected)
    })
  }
})

describe('HLCClock', () => {
  describe('now()', () => {
    it('returns a timestamp with time and counter', () => {
      const clock = new HLCClock(() => 1000)
      const ts = clock.now()
      expect(ts).toHaveProperty('time')
      expect(ts).toHaveProperty('counter')
    })

    it('uses physical time when it advances', () => {
      let pt = 1000
      const clock = new HLCClock(() => pt)
      const ts1 = clock.now()
      expect(ts1).toEqual({ time: 1000, counter: 0 })

      pt = 2000
      const ts2 = clock.now()
      expect(ts2).toEqual({ time: 2000, counter: 0 })
    })

    it('increments counter when physical time does not advance', () => {
      const clock = new HLCClock(() => 1000)
      const ts1 = clock.now()
      expect(ts1).toEqual({ time: 1000, counter: 0 })

      const ts2 = clock.now()
      expect(ts2).toEqual({ time: 1000, counter: 1 })

      const ts3 = clock.now()
      expect(ts3).toEqual({ time: 1000, counter: 2 })
    })

    it('monotonicity: successive calls never go backwards', () => {
      let pt = 1000
      const clock = new HLCClock(() => pt)
      const timestamps = []
      for (let i = 0; i < 20; i++) {
        // Sometimes advance time, sometimes not
        if (i % 3 === 0) pt += 100
        timestamps.push(clock.now())
      }
      for (let i = 1; i < timestamps.length; i++) {
        expect(compare(timestamps[i], timestamps[i - 1])).toBe(1)
      }
    })

    it('returns a copy, not a reference to internal state', () => {
      const clock = new HLCClock(() => 1000)
      const ts1 = clock.now()
      ts1.time = 9999
      const ts2 = clock.now()
      expect(ts2.time).toBe(1000)
    })
  })

  describe('update()', () => {
    it('advances to remote time when remote is ahead', () => {
      const clock = new HLCClock(() => 1000)
      clock.now() // initialize
      const ts = clock.update({ time: 5000, counter: 3 })
      expect(ts.time).toBe(5000)
      expect(ts.counter).toBe(4)
    })

    it('stays on local time when local is ahead of remote', () => {
      let pt = 5000
      const clock = new HLCClock(() => pt)
      clock.now() // initialize at 5000
      pt = 5000 // keep same
      const ts = clock.update({ time: 1000, counter: 0 })
      expect(ts.time).toBe(5000)
      expect(ts.counter).toBe(1)
    })

    it('uses physical time when it is ahead of both', () => {
      let pt = 1000
      const clock = new HLCClock(() => pt)
      clock.now() // initialize at 1000
      pt = 20000 // physical time jumps ahead of both local (1000) and remote (5000)
      const ts = clock.update({ time: 5000, counter: 0 })
      expect(ts.time).toBe(20000)
      expect(ts.counter).toBe(0)
    })

    it('picks max counter when local and remote times are equal', () => {
      const clock = new HLCClock(() => 1000)
      clock.now() // { time: 1000, counter: 0 }
      clock.now() // { time: 1000, counter: 1 }
      clock.now() // { time: 1000, counter: 2 }
      // local is { time: 1000, counter: 2 }, remote has higher counter
      const ts = clock.update({ time: 1000, counter: 10 })
      expect(ts.time).toBe(1000)
      expect(ts.counter).toBe(11) // max(2, 10) + 1
    })

    it('returns a copy, not internal state', () => {
      const clock = new HLCClock(() => 1000)
      clock.now()
      const ts = clock.update({ time: 2000, counter: 0 })
      ts.time = 9999
      const ts2 = clock.now()
      expect(ts2.time).not.toBe(9999)
    })
  })
})
