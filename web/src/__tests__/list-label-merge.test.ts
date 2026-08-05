import { describe, it, expect, vi, beforeEach } from 'vitest'
import { applyFieldLWW, mergeRemoteEvents } from '../db/merge-events'
import type { FieldHLC } from '../db/database'

// In-memory stand-ins for the Dexie lists/labels tables. Hoisted because
// the vi.mock factory below runs before top-level test-file statements.
const { lists, labels } = vi.hoisted(() => {
  return {
    lists: new Map<string, Record<string, unknown>>(),
    labels: new Map<string, Record<string, unknown>>(),
  }
})

vi.mock('../db/database', async (importOriginal) => {
  const { TASK_LWW_FIELDS } = await importOriginal<typeof import('../db/database')>()
  function makeTable(store: Map<string, Record<string, unknown>>) {
    return {
      get: async (id: string) => store.get(id),
      put: async (record: { id: string }) => {
        store.set(record.id, record)
      },
      update: async (id: string, changes: Record<string, unknown>) => {
        const existing = store.get(id)
        if (existing) store.set(id, { ...existing, ...changes })
      },
      delete: async (id: string) => {
        store.delete(id)
      },
    }
  }
  return {
    TASK_LWW_FIELDS,
    db: {
      lists: makeTable(lists),
      labels: makeTable(labels),
    },
  }
})

function remoteEvent(
  eventType: string,
  aggId: string,
  data: Record<string, unknown>,
  timeMs: number,
  counter = 0,
) {
  return {
    id: `evt-${eventType}-${timeMs}`,
    aggregate_id: aggId,
    aggregate_type: eventType.startsWith('List') ? 'list' : 'label',
    event_type: eventType,
    user_id: 'user-1',
    data,
    timestamp: new Date(timeMs).toISOString(),
    counter,
    version: 1,
  }
}

const T0 = Date.parse('2026-07-24T10:00:00Z')

describe('applyFieldLWW', () => {
  it('remote wins when the field has no local tracking', () => {
    const { wins, fieldHlcs } = applyFieldLWW(undefined, 'name', { time: T0, counter: 0 })
    expect(wins).toBe(true)
    expect(fieldHlcs.name).toEqual({ time: T0, counter: 0 })
  })

  it('newer remote event wins over an older local edit', () => {
    const local: FieldHLC = { name: { time: T0, counter: 0 } }
    const { wins } = applyFieldLWW(local, 'name', { time: T0 + 1000, counter: 0 })
    expect(wins).toBe(true)
  })

  it('stale remote event loses to a newer local edit', () => {
    const local: FieldHLC = { name: { time: T0 + 1000, counter: 0 } }
    const { wins, fieldHlcs } = applyFieldLWW(local, 'name', { time: T0, counter: 0 })
    expect(wins).toBe(false)
    expect(fieldHlcs.name).toEqual({ time: T0 + 1000, counter: 0 })
  })

  it('equal HLC ties go to the remote for cross-device convergence', () => {
    const local: FieldHLC = { name: { time: T0, counter: 3 } }
    const { wins } = applyFieldLWW(local, 'name', { time: T0, counter: 3 })
    expect(wins).toBe(true)
  })

  it('tracking one field does not shield another', () => {
    const local: FieldHLC = { name: { time: T0 + 5000, counter: 0 } }
    const { wins } = applyFieldLWW(local, 'colour', { time: T0, counter: 0 })
    expect(wins).toBe(true)
  })
})

describe('mergeRemoteEvents — list/label field LWW', () => {
  beforeEach(() => {
    lists.clear()
    labels.clear()
  })

  it('applies rename and recolour events to an existing list', async () => {
    lists.set('l1', { id: 'l1', name: 'Work', colour: '#ff0000' })

    await mergeRemoteEvents([
      remoteEvent('ListNameUpdated', 'l1', { name: 'Office' }, T0),
      remoteEvent('ListColourUpdated', 'l1', { colour: '#00ff00' }, T0 + 1),
    ])

    const list = lists.get('l1')!
    expect(list.name).toBe('Office')
    expect(list.colour).toBe('#00ff00')
  })

  it('a stale remote rename does not clobber a newer local rename', async () => {
    lists.set('l1', {
      id: 'l1',
      name: 'Renamed locally',
      colour: '#ff0000',
      field_hlcs: { name: { time: T0 + 60_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('ListNameUpdated', 'l1', { name: 'Old remote name' }, T0),
    ])

    expect(lists.get('l1')!.name).toBe('Renamed locally')
  })

  it('concurrent rename and recolour from different devices both survive', async () => {
    labels.set('la1', {
      id: 'la1',
      name: 'Urgent',
      colour: '#ff0000',
      field_hlcs: { name: { time: T0 + 60_000, counter: 0 } },
    })

    // Remote device recoloured before our local rename — colour applies,
    // stale remote name is rejected, local rename survives.
    await mergeRemoteEvents([
      remoteEvent('LabelNameUpdated', 'la1', { name: 'Old remote name' }, T0),
      remoteEvent('LabelColourUpdated', 'la1', { colour: '#0000ff' }, T0),
    ])

    const label = labels.get('la1')!
    expect(label.name).toBe('Urgent')
    expect(label.colour).toBe('#0000ff')
  })

  it('ListCreated inserts a new list when none exists locally', async () => {
    await mergeRemoteEvents([
      remoteEvent('ListCreated', 'l1', { name: 'Work', colour: '#ff0000', position: 'a' }, T0),
    ])
    expect(lists.get('l1')!.name).toBe('Work')
  })

  it('a redelivered ListCreated does not clobber a newer local rename', async () => {
    lists.set('l1', {
      id: 'l1',
      name: 'Renamed locally',
      colour: '#ff0000',
      field_hlcs: { name: { time: T0 + 60_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('ListCreated', 'l1', { name: 'Original', colour: '#123456', position: 'a' }, T0),
    ])

    const list = lists.get('l1')!
    // Tracked rename (newer HLC) survives; untracked colour takes the payload value
    expect(list.name).toBe('Renamed locally')
    expect(list.colour).toBe('#123456')
    expect((list.field_hlcs as FieldHLC).name).toEqual({ time: T0 + 60_000, counter: 0 })
  })

  it('skips update events for lists that do not exist locally', async () => {
    await mergeRemoteEvents([
      remoteEvent('ListNameUpdated', 'ghost', { name: 'Nope' }, T0),
    ])
    expect(lists.has('ghost')).toBe(false)
  })

  it('applies reorder events to existing lists and labels', async () => {
    lists.set('l1', { id: 'l1', name: 'Work', position: 'a' })
    labels.set('la1', { id: 'la1', name: 'Urgent', position: 'b' })

    await mergeRemoteEvents([
      remoteEvent('ListReordered', 'l1', { position: 'aO' }, T0),
      remoteEvent('LabelReordered', 'la1', { position: 'bO' }, T0),
    ])

    expect(lists.get('l1')!.position).toBe('aO')
    expect(labels.get('la1')!.position).toBe('bO')
  })

  it('a stale remote reorder does not clobber a newer local reorder', async () => {
    labels.set('la1', {
      id: 'la1',
      name: 'Urgent',
      position: 'z',
      field_hlcs: { position: { time: T0 + 60_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('LabelReordered', 'la1', { position: 'a' }, T0),
    ])

    expect(labels.get('la1')!.position).toBe('z')
  })

  it('concurrent local rename and remote reorder both survive', async () => {
    lists.set('l1', {
      id: 'l1',
      name: 'Renamed locally',
      position: 'a',
      field_hlcs: { name: { time: T0 + 60_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('ListReordered', 'l1', { position: 'aO' }, T0),
    ])

    const list = lists.get('l1')!
    expect(list.name).toBe('Renamed locally')
    expect(list.position).toBe('aO')
  })

  it('LabelCreated inserts with position, and a redelivery does not clobber a newer local reorder', async () => {
    await mergeRemoteEvents([
      remoteEvent('LabelCreated', 'la1', { name: 'Urgent', colour: '#ff0000', position: 'b' }, T0),
    ])
    expect(labels.get('la1')!.position).toBe('b')

    labels.set('la1', {
      ...labels.get('la1')!,
      position: 'bO',
      field_hlcs: { position: { time: T0 + 60_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('LabelCreated', 'la1', { name: 'Urgent', colour: '#ff0000', position: 'b' }, T0),
    ])

    expect(labels.get('la1')!.position).toBe('bO')
  })
})
