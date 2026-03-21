import { api } from '../api/client'
import { db } from './database'
import type { Task, Priority } from '../api/types'
import type { StoredSubtask, TaskLabel } from './database'

/**
 * Populate IndexedDB from the server API.
 * Falls back to snapshot endpoint if REST endpoints fail (Safari eviction recovery).
 * Called once on app launch (after authentication).
 *
 * If a sync cursor already exists, the database has been populated before
 * and incremental sync will handle catching up — skip the destructive reload.
 */
export async function initialSync(): Promise<void> {
  const existingCursor = await db.syncState.get('cursor')
  if (existingCursor) {
    // Database already seeded — incremental sync will catch up
    return
  }

  let activeTasks: Task[], completedTasks: Task[], deletedTasks: Task[]
  let lists: Awaited<ReturnType<typeof api.listLists>>
  let labels: Awaited<ReturnType<typeof api.listLabels>>

  try {
    ;[activeTasks, completedTasks, deletedTasks, lists, labels] = await Promise.all([
      api.listTasks({ is_completed: 'false' }),
      api.listTasks({ is_completed: 'true' }),
      api.listTasks({ is_deleted: 'true' }),
      api.listLists(),
      api.listLabels(),
    ])
  } catch {
    // REST endpoints failed — try snapshot recovery
    return rehydrateFromSnapshots()
  }

  const allTasks = [...activeTasks, ...completedTasks, ...deletedTasks]

  // Extract embedded subtasks and labels into their own tables
  const subtasks: StoredSubtask[] = []
  const taskLabels: TaskLabel[] = []

  const flatTasks = allTasks.map((task: Task) => {
    if (task.subtasks) {
      for (const st of task.subtasks) {
        subtasks.push({ ...st, taskId: task.id })
      }
    }
    if (task.labels) {
      for (const l of task.labels) {
        taskLabels.push({ taskId: task.id, labelId: l.id })
      }
    }
    // Strip embedded arrays — they live in separate tables
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { subtasks: _s, labels: _l, ...flat } = task
    return flat
  })

  await db.transaction('rw', [db.tasks, db.lists, db.labels, db.subtasks, db.taskLabels], async () => {
    await Promise.all([
      db.tasks.clear(),
      db.lists.clear(),
      db.labels.clear(),
      db.subtasks.clear(),
      db.taskLabels.clear(),
    ])
    await Promise.all([
      db.tasks.bulkPut(flatTasks),
      db.lists.bulkPut(lists),
      db.labels.bulkPut(labels),
      db.subtasks.bulkPut(subtasks),
      db.taskLabels.bulkPut(taskLabels),
    ])
  })

  // Set sync cursor to "now" so the first incremental sync doesn't re-fetch everything
  await db.syncState.put({
    key: 'cursor',
    hlcTime: Date.now(),
    hlcCounter: 0,
  })
}

/**
 * Recovery path: rehydrate IndexedDB from server-side aggregate snapshots.
 * Used when REST endpoints fail (e.g., after Safari IndexedDB eviction).
 */
async function rehydrateFromSnapshots(): Promise<void> {
  const res = await fetch('/api/v1/snapshots', { credentials: 'include' })
  if (!res.ok) return

  const snapshots: { aggregate_id: string; aggregate_type: string; data: Record<string, unknown> }[] = await res.json()

  await db.transaction('rw', [db.tasks, db.lists, db.labels], async () => {
    await Promise.all([db.tasks.clear(), db.lists.clear(), db.labels.clear()])

    for (const snap of snapshots) {
      switch (snap.aggregate_type) {
        case 'task':
          await db.tasks.put({
            id: snap.data.id as string,
            title: snap.data.title as string,
            description: snap.data.description as string | undefined,
            priority: (snap.data.priority as number ?? 0) as Priority,
            due_date: snap.data.due_date as string | undefined,
            due_time: snap.data.due_time as string | undefined,
            list_id: snap.data.list_id as string | undefined,
            position: snap.data.position as string,
            is_completed: snap.data.is_completed as boolean,
            completed_at: snap.data.completed_at as string | undefined,
            is_deleted: snap.data.is_deleted as boolean,
            recurrence_rule: snap.data.recurrence_rule as string | undefined,
            created_at: snap.data.created_at as string,
            updated_at: snap.data.updated_at as string,
          })
          break
        case 'list':
          await db.lists.put({
            id: snap.data.id as string,
            name: snap.data.name as string,
            colour: snap.data.colour as string | undefined,
            icon: snap.data.icon as string | undefined,
            position: snap.data.position as string,
            created_at: snap.data.created_at as string,
            updated_at: snap.data.updated_at as string,
          })
          break
        case 'label':
          await db.labels.put({
            id: snap.data.id as string,
            name: snap.data.name as string,
            colour: snap.data.colour as string | undefined,
          })
          break
      }
    }
  })

  await db.syncState.put({
    key: 'cursor',
    hlcTime: Date.now(),
    hlcCounter: 0,
  })
}
