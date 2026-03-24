import { db } from './database'
import type { FieldHLC } from './database'
import { clock } from './clock'
import { compare, type HLCTimestamp } from '../hlc/hlc'
import { mergeLWW } from '../crdt/lww'
import type { Priority } from '../api/types'

// ---------------------------------------------------------------------------
// Typed payload interfaces for each event type
// ---------------------------------------------------------------------------

interface TaskCreatedPayload {
  title: string
  description?: string
  priority?: number
  due_date?: string
  due_time?: string
  list_id?: string
  position: string
}

interface TaskTitleUpdatedPayload {
  title: string
}

interface TaskDescriptionUpdatedPayload {
  description: string
}

interface TaskPriorityUpdatedPayload {
  priority: number
}

interface TaskDueDateUpdatedPayload {
  due_date?: string
}

interface TaskDueTimeUpdatedPayload {
  due_time?: string
}

interface TaskRecurrenceUpdatedPayload {
  recurrence_rule?: string
}

interface TaskCompletedPayload {
  completed_at: string
}

interface TaskMovedPayload {
  list_id: string
  position: string
}

interface TaskReorderedPayload {
  position: string
}

interface LabelOnTaskPayload {
  label_id: string
}

interface SubtaskCreatedPayload {
  subtask_id: string
  title: string
  position: string
}

interface SubtaskIdPayload {
  subtask_id: string
}

interface SubtaskTitleUpdatedPayload {
  subtask_id: string
  title: string
}

interface ListCreatedPayload {
  name: string
  colour?: string
  icon?: string
  position: string
}

interface LabelCreatedPayload {
  name: string
  colour?: string
}

/**
 * A remote event received from the sync response.
 * Matches the Go eventstore.Event JSON shape.
 */
interface RemoteEvent {
  id: string
  aggregate_id: string
  aggregate_type: string
  event_type: string
  user_id: string
  data: Record<string, unknown>
  timestamp: string  // ISO 8601
  counter: number
  version: number
}

/**
 * Merge remote events from the server into local IndexedDB.
 * Uses LWW (Last-Writer-Wins) — the event is applied only if its HLC
 * timestamp is newer than the local record's updated_at.
 */
export async function mergeRemoteEvents(events: RemoteEvent[]): Promise<void> {
  for (const event of events) {
    // Update client HLC so future local ops are causally after these events
    const eventHLC: HLCTimestamp = {
      time: new Date(event.timestamp).getTime(),
      counter: event.counter,
    }
    clock.update(eventHLC)

    try {
      await applyEvent(event)
    } catch (err) {
      console.warn('merge-events: failed to apply event', event.event_type, event.id, err)
    }
  }
}

async function applyEvent(event: RemoteEvent): Promise<void> {
  const aggId = event.aggregate_id
  const data = event.data
  const updatedAt = event.timestamp
  const eventHLC: HLCTimestamp = {
    time: new Date(event.timestamp).getTime(),
    counter: event.counter,
  }

  switch (event.event_type) {
    // ---- Task events ----
    case 'TaskCreated': {
      const p = data as unknown as TaskCreatedPayload
      const hlcEntry = { time: eventHLC.time, counter: eventHLC.counter }
      const field_hlcs: FieldHLC = {
        title: hlcEntry,
        description: hlcEntry,
        priority: hlcEntry,
        due_date: hlcEntry,
        due_time: hlcEntry,
        recurrence_rule: hlcEntry,
        list_id: hlcEntry,
        position: hlcEntry,
        is_completed: hlcEntry,
        is_deleted: hlcEntry,
      }
      await db.tasks.put({
        id: aggId,
        title: p.title,
        description: p.description,
        priority: (p.priority ?? 0) as Priority,
        due_date: p.due_date ? p.due_date.split('T')[0] : undefined,
        due_time: p.due_time,
        list_id: p.list_id,
        position: p.position,
        is_completed: false,
        is_deleted: false,
        created_at: updatedAt,
        updated_at: updatedAt,
        hlc_time: eventHLC.time,
        hlc_counter: eventHLC.counter,
        field_hlcs,
      })
      break
    }

    case 'TaskTitleUpdated': {
      const p = data as unknown as TaskTitleUpdatedPayload
      await mergeTaskField(aggId, eventHLC, { title: p.title }, ['title'])
      break
    }

    case 'TaskDescriptionUpdated': {
      const p = data as unknown as TaskDescriptionUpdatedPayload
      await mergeTaskField(aggId, eventHLC, { description: p.description }, ['description'])
      break
    }

    case 'TaskPriorityUpdated': {
      const p = data as unknown as TaskPriorityUpdatedPayload
      await mergeTaskField(aggId, eventHLC, { priority: p.priority }, ['priority'])
      break
    }

    case 'TaskDueDateUpdated': {
      const p = data as unknown as TaskDueDateUpdatedPayload
      const dueDate = p.due_date ? p.due_date.split('T')[0] : undefined
      await mergeTaskField(aggId, eventHLC, { due_date: dueDate }, ['due_date'])
      break
    }

    case 'TaskDueTimeUpdated': {
      const p = data as unknown as TaskDueTimeUpdatedPayload
      await mergeTaskField(aggId, eventHLC, { due_time: p.due_time }, ['due_time'])
      break
    }

    case 'TaskRecurrenceUpdated': {
      const p = data as unknown as TaskRecurrenceUpdatedPayload
      await mergeTaskField(aggId, eventHLC, { recurrence_rule: p.recurrence_rule }, ['recurrence_rule'])
      break
    }

    case 'TaskCompleted': {
      const p = data as unknown as TaskCompletedPayload
      await mergeTaskField(aggId, eventHLC, {
        is_completed: true,
        completed_at: p.completed_at,
      }, ['is_completed'])
      break
    }

    case 'TaskUncompleted':
      await mergeTaskField(aggId, eventHLC, { is_completed: false, completed_at: undefined }, ['is_completed'])
      break

    case 'TaskDeleted':
      await mergeTaskField(aggId, eventHLC, { is_deleted: true }, ['is_deleted'])
      break

    case 'TaskRestored':
      await mergeTaskField(aggId, eventHLC, { is_deleted: false }, ['is_deleted'])
      break

    case 'TaskMoved': {
      const p = data as unknown as TaskMovedPayload
      await mergeTaskField(aggId, eventHLC, {
        list_id: p.list_id,
        position: p.position,
      }, ['list_id', 'position'])
      break
    }

    case 'TaskReordered': {
      const p = data as unknown as TaskReorderedPayload
      await mergeTaskField(aggId, eventHLC, {
        position: p.position,
      }, ['position'])
      break
    }

    // ---- Label-on-task events ----
    case 'LabelAdded': {
      const p = data as unknown as LabelOnTaskPayload
      await db.taskLabels.put({ taskId: aggId, labelId: p.label_id })
      break
    }

    case 'LabelRemoved': {
      const p = data as unknown as LabelOnTaskPayload
      await db.taskLabels.where({ taskId: aggId, labelId: p.label_id }).delete()
      break
    }

    // ---- Subtask events ----
    case 'SubtaskCreated': {
      const p = data as unknown as SubtaskCreatedPayload
      await db.subtasks.put({
        id: p.subtask_id,
        taskId: aggId,
        title: p.title,
        is_completed: false,
        position: p.position,
      })
      break
    }

    case 'SubtaskCompleted': {
      const p = data as unknown as SubtaskIdPayload
      await db.subtasks.update(p.subtask_id, { is_completed: true })
      break
    }

    case 'SubtaskUncompleted': {
      const p = data as unknown as SubtaskIdPayload
      await db.subtasks.update(p.subtask_id, { is_completed: false })
      break
    }

    case 'SubtaskTitleUpdated': {
      const p = data as unknown as SubtaskTitleUpdatedPayload
      await db.subtasks.update(p.subtask_id, { title: p.title })
      break
    }

    // ---- List events ----
    case 'ListCreated': {
      const p = data as unknown as ListCreatedPayload
      await db.lists.put({
        id: aggId,
        name: p.name,
        colour: p.colour,
        icon: p.icon,
        position: p.position,
        created_at: updatedAt,
        updated_at: updatedAt,
      })
      break
    }

    case 'ListDeleted':
      await db.lists.delete(aggId)
      break

    // ---- Label events ----
    case 'LabelCreated': {
      const p = data as unknown as LabelCreatedPayload
      await db.labels.put({
        id: aggId,
        name: p.name,
        colour: p.colour,
      })
      break
    }

    case 'LabelDeleted':
      await db.labels.delete(aggId)
      break
  }
}

/**
 * LWW merge for task fields using per-field HLC tracking.
 * Each field has its own HLC timestamp, so concurrent updates to different
 * fields never conflict. Only applies each field if the event's HLC is newer
 * than that specific field's HLC.
 *
 * @param fieldNames - the field(s) whose HLC should be checked/updated
 */
async function mergeTaskField(
  taskId: string,
  eventHLC: HLCTimestamp,
  fields: Record<string, unknown>,
  fieldNames: string[],
): Promise<void> {
  const local = await db.tasks.get(taskId)
  if (!local) {
    // Task doesn't exist locally — skip (TaskCreated event should arrive first)
    return
  }

  const fieldHlcs: FieldHLC = local.field_hlcs ? { ...local.field_hlcs } : {}

  // Fallback HLC for tasks that predate per-field tracking
  const fallbackHLC: HLCTimestamp = {
    time: local.hlc_time ?? new Date(local.updated_at).getTime(),
    counter: local.hlc_counter ?? 0,
  }

  // Check if the remote event wins for ALL specified fields
  // (fields from the same event share the same HLC, so they win or lose together)
  let anyFieldWins = false
  for (const fieldName of fieldNames) {
    const localFieldHLC: HLCTimestamp = fieldHlcs[fieldName]
      ? { time: fieldHlcs[fieldName].time, counter: fieldHlcs[fieldName].counter }
      : fallbackHLC

    const [, winnerHLC] = mergeLWW(null, localFieldHLC, null, eventHLC)
    if (compare(winnerHLC, eventHLC) === 0) {
      anyFieldWins = true
      fieldHlcs[fieldName] = { time: eventHLC.time, counter: eventHLC.counter }
    }
  }

  if (anyFieldWins) {
    // Only apply fields that actually won their per-field comparison
    const winningFields: Record<string, unknown> = {}
    for (const fieldName of fieldNames) {
      if (
        fieldHlcs[fieldName] &&
        fieldHlcs[fieldName].time === eventHLC.time &&
        fieldHlcs[fieldName].counter === eventHLC.counter
      ) {
        // This field's HLC was updated to the event's HLC — it won
        if (fieldName in fields) {
          winningFields[fieldName] = fields[fieldName]
        }
      }
    }

    // Also include non-field-tracked companion values (e.g., completed_at alongside is_completed)
    for (const key of Object.keys(fields)) {
      if (!fieldNames.includes(key)) {
        winningFields[key] = fields[key]
      }
    }

    // Update task-level HLC if event is newer (backward compat)
    const taskLevelHLC: HLCTimestamp = {
      time: local.hlc_time ?? new Date(local.updated_at).getTime(),
      counter: local.hlc_counter ?? 0,
    }
    const [, taskWinnerHLC] = mergeLWW(null, taskLevelHLC, null, eventHLC)
    const taskHlcUpdate = compare(taskWinnerHLC, eventHLC) === 0
      ? { hlc_time: eventHLC.time, hlc_counter: eventHLC.counter }
      : {}

    await db.tasks.update(taskId, {
      ...winningFields,
      updated_at: new Date(eventHLC.time).toISOString(),
      field_hlcs: fieldHlcs,
      ...taskHlcUpdate,
    })
  }
}
