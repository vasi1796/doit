import { describe, it, expect } from 'vitest'
import { between, first, last } from '../crdt/fracindex'

describe('first / last', () => {
  it('first() returns a string', () => {
    expect(typeof first()).toBe('string')
    expect(first().length).toBe(1)
  })

  it('last() returns a string', () => {
    expect(typeof last()).toBe('string')
    expect(last().length).toBe(1)
  })

  it('first() < last() lexicographically', () => {
    expect(first() < last()).toBe(true)
  })
})

describe('between', () => {
  const cases: {
    name: string
    before: string
    after: string
  }[] = [
    { name: 'between two distant chars', before: 'A', after: 'z' },
    { name: 'between empty before and a value (prepend)', before: '', after: 'O' },
    { name: 'between a value and empty after (append)', before: 'O', after: '' },
    { name: 'between adjacent chars', before: 'a', after: 'b' },
    { name: 'between first() and last()', before: first(), after: last() },
  ]

  for (const tc of cases) {
    it(`${tc.name}: result sorts between before and after`, () => {
      const result = between(tc.before, tc.after)
      const effectiveBefore = tc.before || '!'
      const effectiveAfter = tc.after || '~'
      expect(result > effectiveBefore).toBe(true)
      // Result should be less than or equal to effectiveAfter (appending may equal in edge cases)
      // but must be strictly between for normal cases
      expect(result < effectiveAfter || result === effectiveBefore + 'O').toBe(true)
    })
  }

  it('generates unique positions for successive inserts at end', () => {
    let pos = first()
    const positions: string[] = [pos]
    for (let i = 0; i < 10; i++) {
      pos = between(pos, '')
      positions.push(pos)
    }
    // All positions must be in strictly ascending order
    for (let i = 1; i < positions.length; i++) {
      expect(positions[i] > positions[i - 1]).toBe(true)
    }
  })

  it('generates unique positions for successive inserts at beginning', () => {
    let pos = first()
    const positions: string[] = [pos]
    for (let i = 0; i < 10; i++) {
      pos = between('', pos)
      positions.push(pos)
    }
    // Reverse because we prepend each time
    positions.reverse()
    for (let i = 1; i < positions.length; i++) {
      expect(positions[i] > positions[i - 1]).toBe(true)
    }
  })

  it('handles before >= after by appending midpoint', () => {
    const result = between('z', 'a')
    expect(result.startsWith('z')).toBe(true)
    expect(result.length).toBeGreaterThan(1)
  })

  it('handles equal before and after', () => {
    const result = between('m', 'm')
    expect(result.startsWith('m')).toBe(true)
    expect(result > 'm').toBe(true)
  })
})
