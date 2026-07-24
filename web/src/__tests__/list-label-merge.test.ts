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

vi.mock('../db/database', () => {
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

  it('skips update events for lists that do not exist locally', async () => {
    await mergeRemoteEvents([
      remoteEvent('ListNameUpdated', 'ghost', { name: 'Nope' }, T0),
    ])
    expect(lists.has('ghost')).toBe(false)
  })
})
