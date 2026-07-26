import Dexie, { type Table } from 'dexie'
import type { Task, Subtask, Label, List } from '../api/types'

/** Subtask stored in its own table with a foreign key to the parent task. */
export interface StoredSubtask extends Subtask {
  taskId: string
}

/** Join table for the task–label many-to-many relationship. Plain add/delete
 * rows — sync pulls deliver events in HLC order, so no tags/tombstones are
 * needed (see ADR-002 addendum). */
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

/** List/label records carry per-field HLCs so concurrent rename/recolour
 * edits from different devices are both preserved. Absent on rows that
 * have never been edited — any remote update then wins. */
export type StoredList = List & { field_hlcs?: FieldHLC }
export type StoredLabel = Label & { field_hlcs?: FieldHLC }

class DoItDB extends Dexie {
  tasks!: Table<StoredTask>
  lists!: Table<StoredList>
  labels!: Table<StoredLabel>
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

    // v5: user-orderable labels. Backfill mirrors migrations 009/010 exactly
    // (alphabetical by code units, same key formula) so devices that never
    // sync the backfill still agree with the server's positions.
    this.version(5)
      .stores({
        labels: 'id, name, position',
      })
      .upgrade(async (tx) => {
        const labels = await tx.table('labels').toArray()
        labels.sort((a, b) =>
          a.name < b.name ? -1 : a.name > b.name ? 1 : a.id < b.id ? -1 : 1,
        )
        await Promise.all(
          labels.map((label, i) =>
            tx.table('labels').update(label.id, { position: backfillPosition(i) }),
          ),
        )
      })
  }
}

/** Mirrors the SQL backfill in api/migrations/010_label_position_collate.sql. */
export function backfillPosition(n: number): string {
  return String.fromCharCode(35 + Math.floor(n / 90)) + String.fromCharCode(33 + (n % 90))
}

export const db = new DoItDB()
