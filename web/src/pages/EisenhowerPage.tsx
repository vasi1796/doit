import { useState } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskDetail } from '../components/tasks/TaskDetail'
import { TaskList } from '../components/tasks/TaskList'
import { COLORS } from '../constants'
import type { Task } from '../api/types'

interface Quadrant {
  key: string
  title: string
  subtitle: string
  color: string
}

const QUADRANTS: Quadrant[] = [
  { key: 'do', title: 'Do', subtitle: 'Important & Urgent', color: COLORS.red },
  { key: 'schedule', title: 'Schedule', subtitle: 'Important & Not Urgent', color: COLORS.blue },
  { key: 'delegate', title: 'Delegate', subtitle: 'Not Important & Urgent', color: COLORS.orange },
  { key: 'eliminate', title: 'Eliminate', subtitle: 'Not Important & Not Urgent', color: COLORS.gray },
]

function classifyTasks(tasks: Task[]): Record<string, Task[]> {
  const today = new Date().toISOString().split('T')[0]
  const result: Record<string, Task[]> = { do: [], schedule: [], delegate: [], eliminate: [] }

  for (const task of tasks) {
    const isImportant = task.priority >= 2 // high (3) or medium (2)
    const isUrgent = task.due_date != null && task.due_date <= today

    if (isImportant && isUrgent) result.do.push(task)
    else if (isImportant && !isUrgent) result.schedule.push(task)
    else if (!isImportant && isUrgent) result.delegate.push(task)
    else result.eliminate.push(task)
  }

  return result
}

export function EisenhowerPage() {
  const { tasks, loading } = useTasks({ is_completed: 'false' })
  const { lists } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const classified = classifyTasks(tasks)

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-text-primary">Eisenhower Matrix</h1>
        <p className="text-sm text-text-secondary">
          Priority = importance, due today/overdue = urgent
        </p>
      </div>
      {loading ? (
        <div className="px-4 py-8 text-center text-text-secondary text-sm">Loading...</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 p-4">
          {QUADRANTS.map((q) => (
            <div key={q.key} className="flex flex-col min-h-[200px] rounded-xl border border-gray-200 overflow-hidden">
              <div className="px-3 py-2 flex items-center gap-2" style={{ backgroundColor: q.color + '0d' }}>
                <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: q.color }} />
                <div className="min-w-0">
                  <h2 className="text-sm font-semibold" style={{ color: q.color }}>{q.title}</h2>
                  <p className="text-[11px] text-text-secondary">{q.subtitle}</p>
                </div>
                <span className="ml-auto text-[11px] text-text-secondary font-medium shrink-0">{classified[q.key].length}</span>
              </div>
              <div className="flex-1 overflow-y-auto">
                <TaskList
                  tasks={classified[q.key]}
                  loading={false}
                  emptyMessage="No tasks"
                  onTaskSelect={setSelectedId}
                />
              </div>
            </div>
          ))}
        </div>
      )}
      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} />
      )}
    </div>
  )
}
