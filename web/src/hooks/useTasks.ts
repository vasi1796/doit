import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'
import type { Task } from '../api/types'

export interface TaskFilter {
  /** true → only completed, false → only incomplete, undefined → both */
  completed?: boolean
  /** true → trash view (deleted only); otherwise deleted tasks are excluded */
  deleted?: boolean
  /** true → only tasks with no list */
  inbox?: boolean
  listId?: string
  labelId?: string
}

export function useTasks(filter: TaskFilter = {}) {
  const { completed, deleted, inbox, listId, labelId } = filter

  const tasks = useLiveQuery(async () => {
    let collection = db.tasks.filter((t) => t.is_deleted === (deleted === true))
    if (deleted !== true && completed !== undefined) {
      collection = collection.and((t) => t.is_completed === completed)
    }

    if (inbox) {
      collection = collection.and((t) => !t.list_id)
    }

    if (listId) {
      collection = collection.and((t) => t.list_id === listId)
    }

    if (labelId) {
      const taskIds = await db.taskLabels.where('labelId').equals(labelId).toArray()
      const taskIdSet = new Set(taskIds.map((tl) => tl.taskId))
      collection = collection.and((t) => taskIdSet.has(t.id))
    }

    const results = await collection.sortBy('position')

    // Attach embedded subtasks and labels for compatibility with existing components
    return Promise.all(
      results.map(async (task) => {
        const [subtasks, taskLabelLinks] = await Promise.all([
          db.subtasks.where('taskId').equals(task.id).sortBy('position'),
          db.taskLabels.where('taskId').equals(task.id).toArray(),
        ])
        const labels = taskLabelLinks.length > 0
          ? await db.labels.where('id').anyOf(taskLabelLinks.map((tl) => tl.labelId)).toArray()
          : []
        return { ...task, subtasks, labels } as Task
      })
    )
  }, [completed, deleted, inbox, listId, labelId])

  return {
    tasks: tasks ?? [],
    loading: tasks === undefined,
  }
}
