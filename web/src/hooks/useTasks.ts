import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'
import type { Task } from '../api/types'

export function useTasks(params?: Record<string, string>) {
  const tasks = useLiveQuery(async () => {
    let collection = db.tasks.toCollection()

    if (params?.is_deleted === 'true') {
      collection = db.tasks.filter((t) => t.is_deleted === true)
    } else {
      collection = db.tasks.filter((t) => t.is_deleted === false)

      if (params?.is_completed === 'true') {
        collection = collection.and((t) => t.is_completed === true)
      } else if (params?.is_completed === 'false') {
        collection = collection.and((t) => t.is_completed === false)
      }
    }

    if (params?.inbox === 'true') {
      collection = collection.and((t) => !t.list_id)
    }

    if (params?.list_id) {
      const listId = params.list_id
      collection = collection.and((t) => t.list_id === listId)
    }

    if (params?.label_id) {
      const labelId = params.label_id
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
  }, [params ? JSON.stringify(params) : ''])

  return {
    tasks: tasks ?? [],
    loading: tasks === undefined,
  }
}
