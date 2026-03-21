import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '../db/database'
import type { Task } from '../api/types'

export function useTaskDetail(id: string | null) {
  const task = useLiveQuery(async (): Promise<Task | null> => {
    if (!id) return null

    const t = await db.tasks.get(id)
    if (!t) return null

    const [subtasks, taskLabelLinks] = await Promise.all([
      db.subtasks.where('taskId').equals(id).sortBy('position'),
      db.taskLabels.where('taskId').equals(id).toArray(),
    ])
    const labels = taskLabelLinks.length > 0
      ? await db.labels.where('id').anyOf(taskLabelLinks.map((tl) => tl.labelId)).toArray()
      : []

    return { ...t, subtasks, labels } as Task
  }, [id])

  return {
    task: task ?? null,
    loading: task === undefined,
  }
}
