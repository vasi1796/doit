import { describe, it, expect } from 'vitest'
import { computeDropPosition, healMissingPositions } from '../utils/reorder'

const items = (positions: Array<string | undefined>) =>
  positions.map((position, i) => ({ id: `i${i}`, position }))

describe('computeDropPosition', () => {
  it('dragging down one slot lands after the target, not back where it started', () => {
    const list = items(['1', '2'])
    const pos = computeDropPosition(list, 'i0', 'i1')!
    expect(pos > '2').toBe(true)
  })

  it('dragging up one slot lands before the target', () => {
    const list = items(['1', '2'])
    const pos = computeDropPosition(list, 'i1', 'i0')!
    expect(pos < '1').toBe(true)
  })

  it('dragging down multiple slots lands after the target', () => {
    const list = items(['1', '2', '3'])
    const pos = computeDropPosition(list, 'i0', 'i2')!
    expect(pos > '3').toBe(true)
  })

  it('dropping onto a position-less row keeps the dragged row after all positioned rows', () => {
    // Regression: a missing neighbour position must not map to
    // "before everything" — that sent rows to the top of the list.
    const list = items(['a', 'b', undefined])
    const pos = computeDropPosition(list, 'i1', 'i2')!
    expect(pos > 'a').toBe(true)
  })

  it('skips position-less predecessors when finding the prev key', () => {
    const list = items(['a', undefined, undefined])
    const pos = computeDropPosition(list, 'i2', 'i1')!
    expect(pos > 'a').toBe(true)
  })

  it('returns null for unknown ids', () => {
    expect(computeDropPosition(items(['a']), 'i0', 'ghost')).toBeNull()
    expect(computeDropPosition(items(['a']), 'ghost', 'i0')).toBeNull()
  })
})

describe('healMissingPositions', () => {
  it('assigns keys preserving display order and persists each', () => {
    const assigned: Array<[string, string]> = []
    const healed = healMissingPositions(
      items(['a', undefined, undefined]),
      (id, position) => assigned.push([id, position]),
    )

    expect(assigned.map(([id]) => id)).toEqual(['i1', 'i2'])
    const positions = healed.map((i) => i.position!)
    expect(positions[1] > positions[0]).toBe(true)
    expect(positions[2] > positions[1]).toBe(true)
  })

  it('leaves positioned rows and the skipped row untouched', () => {
    const assigned: Array<[string, string]> = []
    const healed = healMissingPositions(
      items(['a', undefined]),
      (id, position) => assigned.push([id, position]),
      'i1',
    )
    expect(assigned).toEqual([])
    expect(healed[1].position).toBeUndefined()
  })
})
