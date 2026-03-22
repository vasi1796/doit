import { db } from './database'
import { clock } from './clock'
import { compare, type HLCTimestamp } from '../hlc/hlc'
import { mergeLWW } from '../crdt/lww'
import type { Priority } from '../api/types'

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
    case 'TaskCreated':
      await db.tasks.put({
        id: aggId,
        title: data.title as string,
        description: data.description as string | undefined,
        priority: ((data.priority as number) ?? 0) as Priority,
        due_date: data.due_date ? (data.due_date as string).split('T')[0] : undefined,
        due_time: data.due_time as string | undefined,
        list_id: data.list_id as string | undefined,
        position: data.position as string,
        is_completed: false,
        is_deleted: false,
        created_at: updatedAt,
        updated_at: updatedAt,
        hlc_time: eventHLC.time,
        hlc_counter: eventHLC.counter,
      })
      break

    case 'TaskTitleUpdated':
      await mergeTaskField(aggId, eventHLC, { title: data.title as string })
      break

    case 'TaskDescriptionUpdated':
      await mergeTaskField(aggId, eventHLC, { description: data.description as string })
      break

    case 'TaskPriorityUpdated':
      await mergeTaskField(aggId, eventHLC, { priority: data.priority as number })
      break

    case 'TaskDueDateUpdated': {
      const dueDate = data.due_date ? (data.due_date as string).split('T')[0] : undefined
      await mergeTaskField(aggId, eventHLC, { due_date: dueDate })
      break
    }

    case 'TaskDueTimeUpdated':
      await mergeTaskField(aggId, eventHLC, { due_time: data.due_time as string | undefined })
      break

    case 'TaskRecurrenceUpdated':
      await mergeTaskField(aggId, eventHLC, { recurrence_rule: data.recurrence_rule as string | undefined })
      break

    case 'TaskCompleted':
      await mergeTaskField(aggId, eventHLC, {
        is_completed: true,
        completed_at: data.completed_at as string,
      })
      break

    case 'TaskUncompleted':
      await mergeTaskField(aggId, eventHLC, { is_completed: false, completed_at: undefined })
      break

    case 'TaskDeleted':
      await mergeTaskField(aggId, eventHLC, { is_deleted: true })
      break

    case 'TaskRestored':
      await mergeTaskField(aggId, eventHLC, { is_deleted: false })
      break

    case 'TaskMoved':
      await mergeTaskField(aggId, eventHLC, {
        list_id: data.list_id as string,
        position: data.position as string,
      })
      break

    case 'TaskReordered':
      await mergeTaskField(aggId, eventHLC, {
        position: data.position as string,
      })
      break

    // ---- Label-on-task events ----
    case 'LabelAdded':
      await db.taskLabels.put({ taskId: aggId, labelId: data.label_id as string })
      break

    case 'LabelRemoved':
      await db.taskLabels.where({ taskId: aggId, labelId: data.label_id as string }).delete()
      break

    // ---- Subtask events ----
    case 'SubtaskCreated':
      await db.subtasks.put({
        id: data.subtask_id as string,
        taskId: aggId,
        title: data.title as string,
        is_completed: false,
        position: data.position as string,
      })
      break

    case 'SubtaskCompleted':
      await db.subtasks.update(data.subtask_id as string, { is_completed: true })
      break

    case 'SubtaskUncompleted':
      await db.subtasks.update(data.subtask_id as string, { is_completed: false })
      break

    case 'SubtaskTitleUpdated':
      await db.subtasks.update(data.subtask_id as string, { title: data.title as string })
      break

    // ---- List events ----
    case 'ListCreated':
      await db.lists.put({
        id: aggId,
        name: data.name as string,
        colour: data.colour as string | undefined,
        icon: data.icon as string | undefined,
        position: data.position as string,
        created_at: updatedAt,
        updated_at: updatedAt,
      })
      break

    case 'ListDeleted':
      await db.lists.delete(aggId)
      break

    // ---- Label events ----
    case 'LabelCreated':
      await db.labels.put({
        id: aggId,
        name: data.name as string,
        colour: data.colour as string | undefined,
      })
      break

    case 'LabelDeleted':
      await db.labels.delete(aggId)
      break
  }
}

/**
 * LWW merge for a task field. Only applies the update if the event's
 * HLC timestamp is newer than the local record's HLC timestamp.
 * Uses the CRDT mergeLWW helper which compares full HLC (time + counter).
 */
async function mergeTaskField(
  taskId: string,
  eventHLC: HLCTimestamp,
  fields: Record<string, unknown>,
): Promise<void> {
  const local = await db.tasks.get(taskId)
  if (!local) {
    // Task doesn't exist locally — skip (TaskCreated event should arrive first)
    return
  }

  const localHLC: HLCTimestamp = {
    time: local.hlc_time ?? new Date(local.updated_at).getTime(),
    counter: local.hlc_counter ?? 0,
  }

  // mergeLWW returns [winningValue, winningHLC]; we only need to know if remote won
  const [, winnerHLC] = mergeLWW(null, localHLC, null, eventHLC)
  if (compare(winnerHLC, eventHLC) === 0) {
    await db.tasks.update(taskId, {
      ...fields,
      updated_at: new Date(eventHLC.time).toISOString(),
      hlc_time: eventHLC.time,
      hlc_counter: eventHLC.counter,
    })
  }
}
