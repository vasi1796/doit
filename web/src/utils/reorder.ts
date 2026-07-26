import { between } from '../crdt/fracindex'

export interface Orderable {
  id: string
  position?: string
}

/**
 * Fractional-index position for dropping `activeId` at `overId`'s slot,
 * matching dnd-kit's arrayMove semantics: the dragged item ends up at
 * `overId`'s index in the final array. `items` must be in display order.
 *
 * Rows without a position sort after all positioned rows (mirroring
 * useLabels), so a position-less `next` neighbour means "end of the
 * positioned range" and a position-less `prev` falls back to the nearest
 * positioned predecessor.
 */
export function computeDropPosition(
  items: ReadonlyArray<Orderable>,
  activeId: string,
  overId: string,
): string | null {
  const oldIndex = items.findIndex((i) => i.id === activeId)
  const newIndex = items.findIndex((i) => i.id === overId)
  if (oldIndex === -1 || newIndex === -1) return null

  const reordered = items.filter((_, i) => i !== oldIndex)

  let prevPos = ''
  for (let i = newIndex - 1; i >= 0; i--) {
    const p = reordered[i]?.position
    if (p != null) {
      prevPos = p
      break
    }
  }
  const nextPos = newIndex < reordered.length ? (reordered[newIndex].position ?? '') : ''
  return between(prevPos, nextPos)
}

/**
 * Assign fractional-index keys to position-less rows (legacy data, e.g. from
 * a projection rebuild) in their current display order, so drop computations
 * and the nulls-last sort agree. Returns the patched array; `assign` is
 * called once per healed row to persist the key. `skipId` (the actively
 * dragged row) is left unhealed — its position is about to be overwritten.
 */
export function healMissingPositions<T extends Orderable>(
  items: ReadonlyArray<T>,
  assign: (id: string, position: string) => void,
  skipId?: string,
): T[] {
  let lastPos = ''
  return items.map((item) => {
    if (item.position != null) {
      lastPos = item.position
      return item
    }
    if (item.id === skipId) return item
    const position = between(lastPos, '')
    lastPos = position
    assign(item.id, position)
    return { ...item, position }
  })
}
