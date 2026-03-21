/** A single add or remove operation in an Observed-Remove Set. */
export interface ORSetOp {
  value: string
  tag: string   // unique per operation (UUID)
  op: 'add' | 'remove'
}

/** Merge two OR-Set operation logs by taking their union. Deduplicates by tag. */
export function mergeORSet(local: ORSetOp[], remote: ORSetOp[]): ORSetOp[] {
  // Deduplicate by (tag, op) pair — the same tag can appear as both add and remove
  const seen = new Set<string>()
  const merged: ORSetOp[] = []

  for (const op of local) {
    const key = `${op.tag}:${op.op}`
    if (!seen.has(key)) {
      seen.add(key)
      merged.push(op)
    }
  }
  for (const op of remote) {
    const key = `${op.tag}:${op.op}`
    if (!seen.has(key)) {
      seen.add(key)
      merged.push(op)
    }
  }

  return merged
}

/** Compute current set members from an OR-Set operation log. */
export function materialize(ops: ORSetOp[]): string[] {
  const byValue = new Map<string, { addTags: Set<string>; removeTags: Set<string> }>()

  for (const op of ops) {
    let state = byValue.get(op.value)
    if (!state) {
      state = { addTags: new Set(), removeTags: new Set() }
      byValue.set(op.value, state)
    }
    if (op.op === 'add') {
      state.addTags.add(op.tag)
    } else {
      state.removeTags.add(op.tag)
    }
  }

  const result: string[] = []
  for (const [value, state] of byValue) {
    for (const addTag of state.addTags) {
      if (!state.removeTags.has(addTag)) {
        result.push(value)
        break
      }
    }
  }

  return result
}
