import Dexie, { type Table } from 'dexie'
import type { Task, Subtask, Label, List } from '../api/types'

/** Subtask stored in its own table with a foreign key to the parent task. */
export interface StoredSubtask extends Subtask {
  taskId: string
}

/** Join table for the task–label many-to-many relationship (OR-Set in Phase 2). */
export interface TaskLabel {
  taskId: string
  labelId: string
}

/** Queued operation awaiting sync to the server. */
export interface SyncOp {
  id?: number
  operationType: string
  aggregateId: string
  data: string
  hlcTime: number
  hlcCounter: number
  createdAt: number
  retryCount?: number
}

/** Tracks the last sync point for incremental pull. */
export interface SyncCursor {
  key: string  // always 'cursor'
  hlcTime: number
  hlcCounter: number
}

/** Key-value store for local-only user preferences (theme, future UI state).
 * These never sync to the server — they're device-local by design. */
export interface UserPreference {
  key: string
  value: string
}

/** Per-field HLC timestamps for fine-grained LWW merge. */
export interface FieldHLC {
  [field: string]: { time: number; counter: number }
}

/** Task record stored in IndexedDB — includes HLC fields for LWW merge. */
export type StoredTask = Omit<Task, 'subtasks' | 'labels'> & {
  hlc_time?: number      // keep for backward compat during migration
  hlc_counter?: number   // keep for backward compat during migration
  field_hlcs?: FieldHLC  // per-field HLC tracking
}

class DoItDB extends Dexie {
  tasks!: Table<StoredTask>
  lists!: Table<List>
  labels!: Table<Label>
  taskLabels!: Table<TaskLabel>
  subtasks!: Table<StoredSubtask>
  syncQueue!: Table<SyncOp>
  syncState!: Table<SyncCursor>
  userPreferences!: Table<UserPreference>

  constructor() {
    super('doit')

    this.version(1).stores({
      tasks: 'id, list_id, due_date, position, is_completed, is_deleted',
      lists: 'id, position',
      labels: 'id, name',
      taskLabels: '[taskId+labelId], taskId, labelId',
      subtasks: 'id, taskId, position',
      syncQueue: '++id, createdAt',
    })

    this.version(2).stores({
      syncState: '&key',
    })

    this.version(3).stores({
      syncQueue: '++id, createdAt',
    })

    // v4: local-only user preferences (theme, etc.) — device-local, never synced
    this.version(4).stores({
      userPreferences: '&key',
    })
  }
}

export const db = new DoItDB()
