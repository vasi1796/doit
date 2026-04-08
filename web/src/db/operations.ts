import { db } from './database'
import type { FieldHLC } from './database'
import { clock } from './clock'
import type { CreateTaskRequest, UpdateTaskRequest, CreateListRequest, CreateLabelRequest } from '../api/types'

export const SyncOpType = {
  CREATE_TASK: 'CreateTask',
  UPDATE_TASK: 'UpdateTask',
  COMPLETE_TASK: 'CompleteTask',
  UNCOMPLETE_TASK: 'UncompleteTask',
  DELETE_TASK: 'DeleteTask',
  RESTORE_TASK: 'RestoreTask',
  ADD_LABEL: 'AddLabel',
  REMOVE_LABEL: 'RemoveLabel',
  CREATE_SUBTASK: 'CreateSubtask',
  COMPLETE_SUBTASK: 'CompleteSubtask',
  UNCOMPLETE_SUBTASK: 'UncompleteSubtask',
  UPDATE_SUBTASK_TITLE: 'UpdateSubtaskTitle',
  CREATE_LIST: 'CreateList',
  DELETE_LIST: 'DeleteList',
  CREATE_LABEL: 'CreateLabel',
  DELETE_LABEL: 'DeleteLabel',
} as const

/** Generate a UUID v4. Falls back to manual generation on insecure contexts (HTTP). */
function uuid(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // Fallback for non-HTTPS contexts (e.g., local dev on phone over HTTP)
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16)
  })
}

/** HLC fields to write alongside every task mutation for proper LWW merge. */
function hlcFields() {
  const now = clock.now()
  return {
    hlc: now,
    fields: { updated_at: new Date().toISOString(), hlc_time: now.time, hlc_counter: now.counter },
  }
}

/** Build a FieldHLC map for the given field names using the provided HLC. */
function buildFieldHlcs(fieldNames: string[], hlc: { time: number; counter: number }): FieldHLC {
  const result: FieldHLC = {}
  for (const name of fieldNames) {
    result[name] = { time: hlc.time, counter: hlc.counter }
  }
  return result
}

/** Merge new per-field HLCs into an existing task's field_hlcs. */
async function mergeLocalFieldHlcs(taskId: string, newFieldHlcs: FieldHLC): Promise<FieldHLC> {
  const task = await db.tasks.get(taskId)
  const existing = task?.field_hlcs ?? {}
  return { ...existing, ...newFieldHlcs }
}

async function queueOp(operationType: string, aggregateId: string, hlc: { time: number; counter: number }, data: Record<string, unknown> = {}) {
  await db.syncQueue.add({
    operationType,
    aggregateId,
    data: JSON.stringify(data),
    hlcTime: hlc.time,
    hlcCounter: hlc.counter,
    createdAt: Date.now(),
  })
  // Nudge the sync engine to flush soon (500ms debounce)
  window.__syncEngine?.nudge()
}

// ---------------------------------------------------------------------------
// Task operations
// ---------------------------------------------------------------------------

export async function createTask(data: CreateTaskRequest): Promise<string> {
  const id = uuid()
  const { hlc, fields } = hlcFields()

  const allTaskFields = [
    'title', 'description', 'priority', 'due_date', 'due_time',
    'recurrence_rule', 'list_id', 'position', 'is_completed', 'is_deleted',
  ]
  const field_hlcs = buildFieldHlcs(allTaskFields, hlc)

  await db.tasks.put({
    id,
    title: data.title,
    description: data.description,
    priority: data.priority,
    due_date: data.due_date,
    due_time: data.due_time,
    list_id: data.list_id,
    position: data.position,
    is_completed: false,
    is_deleted: false,
    created_at: new Date().toISOString(),
    field_hlcs,
    ...fields,
  })

  await queueOp(SyncOpType.CREATE_TASK, id, hlc, data)
  return id
}

export async function updateTask(id: string, data: UpdateTaskRequest): Promise<void> {
  const { hlc, fields } = hlcFields()
  const changedFields = Object.keys(data)
  const newFieldHlcs = buildFieldHlcs(changedFields, hlc)
  const field_hlcs = await mergeLocalFieldHlcs(id, newFieldHlcs)
  await db.tasks.update(id, { ...data, ...fields, field_hlcs })
  await queueOp(SyncOpType.UPDATE_TASK, id, hlc, data)
}

export async function completeTask(id: string): Promise<void> {
  const { hlc, fields } = hlcFields()
  const field_hlcs = await mergeLocalFieldHlcs(id, buildFieldHlcs(['is_completed'], hlc))
  await db.tasks.update(id, {
    is_completed: true,
    completed_at: new Date().toISOString(),
    field_hlcs,
    ...fields,
  })
  await queueOp(SyncOpType.COMPLETE_TASK, id, hlc)
}

export async function uncompleteTask(id: string): Promise<void> {
  const { hlc, fields } = hlcFields()
  const field_hlcs = await mergeLocalFieldHlcs(id, buildFieldHlcs(['is_completed'], hlc))
  await db.tasks.update(id, {
    is_completed: false,
    completed_at: undefined,
    field_hlcs,
    ...fields,
  })
  await queueOp(SyncOpType.UNCOMPLETE_TASK, id, hlc)
}

export async function deleteTask(id: string): Promise<void> {
  const { hlc, fields } = hlcFields()
  const field_hlcs = await mergeLocalFieldHlcs(id, buildFieldHlcs(['is_deleted'], hlc))
  await db.tasks.update(id, { is_deleted: true, field_hlcs, ...fields })
  await queueOp(SyncOpType.DELETE_TASK, id, hlc)
}

export async function restoreTask(id: string): Promise<void> {
  const { hlc, fields } = hlcFields()
  const field_hlcs = await mergeLocalFieldHlcs(id, buildFieldHlcs(['is_deleted'], hlc))
  await db.tasks.update(id, { is_deleted: false, field_hlcs, ...fields })
  await queueOp(SyncOpType.RESTORE_TASK, id, hlc)
}

// ---------------------------------------------------------------------------
// Label-on-task operations
// ---------------------------------------------------------------------------

export async function addLabel(taskId: string, labelId: string): Promise<void> {
  const { hlc } = hlcFields()
  await db.taskLabels.put({ taskId, labelId })
  await queueOp(SyncOpType.ADD_LABEL, taskId, hlc, { label_id: labelId })
}

export async function removeLabel(taskId: string, labelId: string): Promise<void> {
  const { hlc } = hlcFields()
  await db.taskLabels.where({ taskId, labelId }).delete()
  await queueOp(SyncOpType.REMOVE_LABEL, taskId, hlc, { label_id: labelId })
}

// ---------------------------------------------------------------------------
// Subtask operations
// ---------------------------------------------------------------------------

export async function createSubtask(taskId: string, data: { title: string; position: string }): Promise<string> {
  const id = uuid()
  const { hlc } = hlcFields()
  await db.subtasks.put({ id, taskId, title: data.title, is_completed: false, position: data.position })
  await queueOp(SyncOpType.CREATE_SUBTASK, taskId, hlc, { subtask_id: id, ...data })
  return id
}

export async function completeSubtask(taskId: string, subtaskId: string): Promise<void> {
  const { hlc } = hlcFields()
  await db.subtasks.update(subtaskId, { is_completed: true })
  await queueOp(SyncOpType.COMPLETE_SUBTASK, taskId, hlc, { subtask_id: subtaskId })
}

export async function uncompleteSubtask(taskId: string, subtaskId: string): Promise<void> {
  const { hlc } = hlcFields()
  await db.subtasks.update(subtaskId, { is_completed: false })
  await queueOp(SyncOpType.UNCOMPLETE_SUBTASK, taskId, hlc, { subtask_id: subtaskId })
}

export async function updateSubtaskTitle(taskId: string, subtaskId: string, title: string): Promise<void> {
  const { hlc } = hlcFields()
  await db.subtasks.update(subtaskId, { title })
  await queueOp(SyncOpType.UPDATE_SUBTASK_TITLE, taskId, hlc, { subtask_id: subtaskId, title })
}

// ---------------------------------------------------------------------------
// List operations
// ---------------------------------------------------------------------------

export async function createList(data: CreateListRequest): Promise<string> {
  const id = uuid()
  const { hlc } = hlcFields()
  await db.lists.put({
    id,
    name: data.name,
    colour: data.colour,
    icon: data.icon,
    position: data.position,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  })
  await queueOp(SyncOpType.CREATE_LIST, id, hlc, data)
  return id
}

export async function deleteList(id: string): Promise<void> {
  const { hlc } = hlcFields()
  await db.lists.delete(id)
  await queueOp(SyncOpType.DELETE_LIST, id, hlc)
}

// ---------------------------------------------------------------------------
// Label operations
// ---------------------------------------------------------------------------

export async function createLabel(data: CreateLabelRequest): Promise<string> {
  const id = uuid()
  const { hlc } = hlcFields()
  await db.labels.put({ id, name: data.name, colour: data.colour })
  await queueOp(SyncOpType.CREATE_LABEL, id, hlc, data)
  return id
}

export async function deleteLabel(id: string): Promise<void> {
  const { hlc } = hlcFields()
  await db.labels.delete(id)
  await queueOp(SyncOpType.DELETE_LABEL, id, hlc)
}

// ---------------------------------------------------------------------------
// User preferences (device-local, never synced)
// ---------------------------------------------------------------------------

/**
 * Write a local-only user preference. These are intentionally not queued
 * for sync — each device has its own UI state (theme, etc.).
 */
export async function setUserPreference(key: string, value: string): Promise<void> {
  await db.userPreferences.put({ key, value })
}
