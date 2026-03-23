import { describe, it, expect } from 'vitest'
import { mergeORSet, materialize, type ORSetOp } from '../crdt/orset'

describe('mergeORSet', () => {
  it('returns empty array when both inputs are empty', () => {
    expect(mergeORSet([], [])).toEqual([])
  })

  it('returns local ops when remote is empty', () => {
    const local: ORSetOp[] = [{ value: 'a', tag: 't1', op: 'add' }]
    expect(mergeORSet(local, [])).toEqual(local)
  })

  it('returns remote ops when local is empty', () => {
    const remote: ORSetOp[] = [{ value: 'a', tag: 't1', op: 'add' }]
    expect(mergeORSet([], remote)).toEqual(remote)
  })

  it('deduplicates ops with the same tag and op type', () => {
    const op: ORSetOp = { value: 'a', tag: 't1', op: 'add' }
    const result = mergeORSet([op], [op])
    expect(result).toEqual([op])
  })

  it('keeps both add and remove with the same tag', () => {
    const add: ORSetOp = { value: 'a', tag: 't1', op: 'add' }
    const remove: ORSetOp = { value: 'a', tag: 't1', op: 'remove' }
    const result = mergeORSet([add], [remove])
    expect(result).toHaveLength(2)
    expect(result).toContainEqual(add)
    expect(result).toContainEqual(remove)
  })

  it('merges disjoint sets', () => {
    const local: ORSetOp[] = [{ value: 'a', tag: 't1', op: 'add' }]
    const remote: ORSetOp[] = [{ value: 'b', tag: 't2', op: 'add' }]
    const result = mergeORSet(local, remote)
    expect(result).toHaveLength(2)
  })
})

describe('materialize', () => {
  it('returns empty array for no ops', () => {
    expect(materialize([])).toEqual([])
  })

  it('returns value for a single add', () => {
    const ops: ORSetOp[] = [{ value: 'a', tag: 't1', op: 'add' }]
    expect(materialize(ops)).toEqual(['a'])
  })

  it('excludes value when add tag is removed', () => {
    const ops: ORSetOp[] = [
      { value: 'a', tag: 't1', op: 'add' },
      { value: 'a', tag: 't1', op: 'remove' },
    ]
    expect(materialize(ops)).toEqual([])
  })

  it('keeps value when only some add tags are removed', () => {
    const ops: ORSetOp[] = [
      { value: 'a', tag: 't1', op: 'add' },
      { value: 'a', tag: 't2', op: 'add' },
      { value: 'a', tag: 't1', op: 'remove' },
    ]
    expect(materialize(ops)).toEqual(['a'])
  })

  it('handles multiple values', () => {
    const ops: ORSetOp[] = [
      { value: 'a', tag: 't1', op: 'add' },
      { value: 'b', tag: 't2', op: 'add' },
      { value: 'c', tag: 't3', op: 'add' },
    ]
    const result = materialize(ops)
    expect(result).toHaveLength(3)
    expect(result).toContain('a')
    expect(result).toContain('b')
    expect(result).toContain('c')
  })

  it('concurrent add-remove: re-add with new tag survives', () => {
    const ops: ORSetOp[] = [
      { value: 'a', tag: 't1', op: 'add' },
      { value: 'a', tag: 't1', op: 'remove' },
      { value: 'a', tag: 't2', op: 'add' },
    ]
    expect(materialize(ops)).toEqual(['a'])
  })

  it('value removed when all add tags have matching removes', () => {
    const ops: ORSetOp[] = [
      { value: 'a', tag: 't1', op: 'add' },
      { value: 'a', tag: 't2', op: 'add' },
      { value: 'a', tag: 't1', op: 'remove' },
      { value: 'a', tag: 't2', op: 'remove' },
    ]
    expect(materialize(ops)).toEqual([])
  })
})
