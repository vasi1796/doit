import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mergeRemoteEvents } from '../db/merge-events'
import type { FieldHLC } from '../db/database'

const { tasks, lists, labels } = vi.hoisted(() => ({
  tasks: new Map<string, Record<string, unknown>>(),
  lists: new Map<string, Record<string, unknown>>(),
  labels: new Map<string, Record<string, unknown>>(),
}))

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
      tasks: makeTable(tasks),
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
    aggregate_type: 'task',
    event_type: eventType,
    user_id: 'user-1',
    data,
    timestamp: new Date(timeMs).toISOString(),
    counter,
    version: 1,
  }
}

const T0 = Date.parse('2026-07-26T10:00:00Z')

describe('mergeRemoteEvents — task per-field LWW', () => {
  beforeEach(() => {
    tasks.clear()
  })

  it('TaskCreated inserts a new task with per-field tracking', async () => {
    await mergeRemoteEvents([
      remoteEvent('TaskCreated', 't1', { title: 'Buy milk', priority: 2, position: 'a' }, T0),
    ])

    const task = tasks.get('t1')!
    expect(task.title).toBe('Buy milk')
    expect((task.field_hlcs as FieldHLC).title).toEqual({ time: T0, counter: 0 })
  })

  it('a redelivered TaskCreated does not clobber newer local edits or completion', async () => {
    tasks.set('t1', {
      id: 't1',
      title: 'Renamed locally',
      priority: 2,
      position: 'a',
      is_completed: true,
      completed_at: '2026-07-26T11:00:00Z',
      is_deleted: false,
      field_hlcs: {
        title: { time: T0 + 60_000, counter: 0 },
        is_completed: { time: T0 + 60_000, counter: 0 },
      },
    })

    await mergeRemoteEvents([
      remoteEvent('TaskCreated', 't1', { title: 'Original', priority: 1, position: 'a' }, T0),
    ])

    const task = tasks.get('t1')!
    expect(task.title).toBe('Renamed locally')
    expect(task.is_completed).toBe(true)
    // Untracked field takes the payload value, as with list/label redelivery
    expect(task.priority).toBe(1)
  })

  it('a redelivered TaskCreated with a newer HLC still cannot un-complete a task', async () => {
    // The server echoes the create back in the same pull that carried the op,
    // stamped with a server HLC newer than the client's completion.
    tasks.set('t1', {
      id: 't1',
      title: 'Original',
      priority: 1,
      position: 'a',
      is_completed: true,
      completed_at: '2026-07-26T10:00:30Z',
      is_deleted: false,
      field_hlcs: { is_completed: { time: T0 + 30_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('TaskCreated', 't1', { title: 'Original', priority: 1, position: 'a' }, T0 + 60_000),
    ])

    const task = tasks.get('t1')!
    expect(task.is_completed).toBe(true)
    expect(task.completed_at).toBe('2026-07-26T10:00:30Z')
    expect(task.is_deleted).toBe(false)
  })

  it('a stale remote field update loses to a newer tracked local edit', async () => {
    tasks.set('t1', {
      id: 't1',
      title: 'Newer local title',
      field_hlcs: { title: { time: T0 + 60_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('TaskTitleUpdated', 't1', { title: 'Old remote title' }, T0),
    ])

    expect(tasks.get('t1')!.title).toBe('Newer local title')
  })

  it('tracking one field does not shield another (per-field isolation)', async () => {
    tasks.set('t1', {
      id: 't1',
      title: 'Newer local title',
      priority: 0,
      field_hlcs: { title: { time: T0 + 60_000, counter: 0 } },
    })

    await mergeRemoteEvents([
      remoteEvent('TaskTitleUpdated', 't1', { title: 'Old remote title' }, T0),
      remoteEvent('TaskPriorityUpdated', 't1', { priority: 3 }, T0),
    ])

    const task = tasks.get('t1')!
    expect(task.title).toBe('Newer local title')
    expect(task.priority).toBe(3)
  })

  it('rows that predate per-field tracking fall back to the record-level HLC', async () => {
    tasks.set('t1', {
      id: 't1',
      title: 'Edited on old client',
      hlc_time: T0 + 60_000,
      hlc_counter: 0,
    })

    await mergeRemoteEvents([
      remoteEvent('TaskTitleUpdated', 't1', { title: 'Old remote title' }, T0),
    ])

    expect(tasks.get('t1')!.title).toBe('Edited on old client')
  })

  it('a winning completion applies its companion completed_at value', async () => {
    tasks.set('t1', { id: 't1', title: 'x', is_completed: false })

    await mergeRemoteEvents([
      remoteEvent('TaskCompleted', 't1', { completed_at: '2026-07-26T12:00:00Z' }, T0),
    ])

    const task = tasks.get('t1')!
    expect(task.is_completed).toBe(true)
    expect(task.completed_at).toBe('2026-07-26T12:00:00Z')
  })

  it('skips update events for tasks that do not exist locally', async () => {
    await mergeRemoteEvents([
      remoteEvent('TaskTitleUpdated', 'ghost', { title: 'Nope' }, T0),
    ])
    expect(tasks.has('ghost')).toBe(false)
  })
})
