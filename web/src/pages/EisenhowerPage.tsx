import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
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
  const { selectTask } = useLayoutContext()

  const classified = classifyTasks(tasks)

  return (
    <div className="flex flex-col h-full">
      <div className="px-6 pt-6 pb-2">
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight">Eisenhower Matrix</h1>
        <p className="text-[14px] text-text-secondary mt-0.5">
          Priority = importance · due today/overdue = urgent
        </p>
      </div>

      {/* Summary pills */}
      <div className="px-6 flex flex-wrap gap-2 mb-3">
        {QUADRANTS.map((q) => (
          <span
            key={q.key}
            className="inline-flex items-center gap-1.5 text-[12px] font-medium px-2.5 py-1 rounded-full"
            style={{
              backgroundColor: q.color + '1A',
              color: q.color,
            }}
          >
            <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: q.color }} />
            {q.title} · {classified[q.key].length}
          </span>
        ))}
      </div>

      {loading ? (
        <div className="px-4 py-8 text-center text-text-tertiary text-sm">Loading…</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 px-4 pb-4 flex-1 min-h-0">
          {QUADRANTS.map((q) => (
            <div key={q.key} className="flex flex-col min-h-[200px] rounded-[14px] border border-separator overflow-hidden bg-bg-elevated">
              <div
                className="px-4 py-2.5 flex items-center gap-2 border-b border-separator"
                style={{ backgroundColor: q.color + '0F' }}
              >
                <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: q.color }} />
                <div className="min-w-0">
                  <h2 className="text-[14px] font-semibold" style={{ color: q.color }}>{q.title}</h2>
                  <p className="text-[11px] text-text-tertiary">{q.subtitle}</p>
                </div>
                <span
                  className="ml-auto text-[11px] font-semibold shrink-0 min-w-[20px] h-5 px-1.5 rounded-full flex items-center justify-center"
                  style={{ backgroundColor: q.color + '1F', color: q.color }}
                >
                  {classified[q.key].length}
                </span>
              </div>
              <div className="flex-1 overflow-y-auto">
                <TaskList
                  tasks={classified[q.key]}
                  loading={false}
                  emptyMessage="No tasks"
                  onTaskSelect={selectTask}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
