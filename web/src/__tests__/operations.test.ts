import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import type { FieldHLC } from '../db/database'

const { tasks, labels, queued } = vi.hoisted(() => ({
  tasks: new Map<string, Record<string, unknown>>(),
  labels: new Map<string, Record<string, unknown>>(),
  queued: [] as Record<string, unknown>[],
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
    }
  }
  return {
    db: {
      tasks: makeTable(tasks),
      labels: makeTable(labels),
      syncQueue: {
        add: async (op: Record<string, unknown>) => {
          queued.push(op)
        },
      },
    },
  }
})

const T0 = Date.parse('2026-07-26T10:00:00Z')
vi.mock('../db/clock', () => ({
  clock: { now: () => ({ time: T0, counter: 7 }) },
}))

import * as operations from '../db/operations'

describe('operations — optimistic writes and sync queueing', () => {
  const nudge = vi.fn()

  beforeEach(() => {
    tasks.clear()
    labels.clear()
    queued.length = 0
    nudge.mockClear()
    vi.stubGlobal('window', { __syncEngine: { nudge } })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('updateTask stamps field_hlcs for exactly the changed fields, with the queued HLC', async () => {
    tasks.set('t1', {
      id: 't1',
      title: 'x',
      field_hlcs: { title: { time: 1, counter: 0 } },
    })

    await operations.updateTask('t1', { position: 'aO' })

    const task = tasks.get('t1')!
    const hlcs = task.field_hlcs as FieldHLC
    expect(hlcs.position).toEqual({ time: T0, counter: 7 })
    expect(hlcs.title).toEqual({ time: 1, counter: 0 })

    expect(queued).toHaveLength(1)
    expect(queued[0]).toMatchObject({
      operationType: 'UpdateTask',
      aggregateId: 't1',
      hlcTime: T0,
      hlcCounter: 7,
    })
    expect(nudge).toHaveBeenCalledOnce()
  })

  it('createTask seeds per-field tracking for every merge-protected field', async () => {
    const id = await operations.createTask({ title: 'New', priority: 1, position: 'a' })

    const task = tasks.get(id)!
    const hlcs = task.field_hlcs as FieldHLC
    const allTaskFields = [
      'title', 'description', 'priority', 'due_date', 'due_time',
      'recurrence_rule', 'list_id', 'position', 'is_completed', 'is_deleted',
    ]
    for (const field of allTaskFields) {
      expect(hlcs[field]).toEqual({ time: T0, counter: 7 })
    }
    expect(queued[0]).toMatchObject({ operationType: 'CreateTask', aggregateId: id })
  })

  it('an update with only undefined fields queues nothing', async () => {
    labels.set('la1', { id: 'la1', name: 'Urgent' })

    await operations.updateLabel('la1', { name: undefined, colour: undefined })

    expect(queued).toHaveLength(0)
    expect(nudge).not.toHaveBeenCalled()
  })
})
