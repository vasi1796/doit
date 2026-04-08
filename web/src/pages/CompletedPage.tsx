import { useMemo } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import type { Task } from '../api/types'

type TimeGroup = 'today' | 'yesterday' | 'week' | 'earlier'

function groupTasksByTime(tasks: Task[]): Record<TimeGroup, Task[]> {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  const weekAgo = new Date(today)
  weekAgo.setDate(weekAgo.getDate() - 7)

  const result: Record<TimeGroup, Task[]> = { today: [], yesterday: [], week: [], earlier: [] }

  for (const task of tasks) {
    if (!task.completed_at) {
      result.earlier.push(task)
      continue
    }
    const completed = new Date(task.completed_at)
    if (completed >= today) result.today.push(task)
    else if (completed >= yesterday) result.yesterday.push(task)
    else if (completed >= weekAgo) result.week.push(task)
    else result.earlier.push(task)
  }

  return result
}

export function CompletedPage() {
  const { tasks, loading } = useTasks({ is_completed: 'true' })
  const { selectTask } = useLayoutContext()

  const grouped = useMemo(() => groupTasksByTime(tasks), [tasks])
  const monthCount = useMemo(() => {
    const now = new Date()
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1)
    return tasks.filter((t) => t.completed_at && new Date(t.completed_at) >= monthStart).length
  }, [tasks])

  const sections: { key: TimeGroup; label: string }[] = [
    { key: 'today', label: 'Today' },
    { key: 'yesterday', label: 'Yesterday' },
    { key: 'week', label: 'This week' },
    { key: 'earlier', label: 'Earlier' },
  ]

  return (
    <div>
      <div className="px-6 pt-6 pb-2">
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight">Completed</h1>
      </div>
      {tasks.length > 0 && (
        <div className="mx-6 mt-2 mb-4 p-4 rounded-[14px] bg-accent-light border border-accent/10 flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-accent/20 flex items-center justify-center shrink-0">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-accent">
              <path d="M9 12l2 2 4-4m6 2a9 9 0 1 1-18 0 9 9 0 0 1 18 0z" />
            </svg>
          </div>
          <div>
            <p className="text-[15px] font-semibold text-text-primary">{monthCount} {monthCount === 1 ? 'task' : 'tasks'} completed this month</p>
            <p className="text-[12px] text-text-secondary">Keep up the momentum</p>
          </div>
        </div>
      )}
      {loading && <div className="px-4 py-8 text-center text-text-tertiary text-sm">Loading…</div>}
      {!loading && tasks.length === 0 && (
        <div className="px-4 py-16 text-center text-text-tertiary text-sm">No completed tasks yet</div>
      )}
      {sections.map((s) => {
        if (grouped[s.key].length === 0) return null
        return (
          <div key={s.key}>
            <div className="px-6 pt-5 pb-1">
              <h2 className="text-[11px] font-semibold text-text-tertiary uppercase tracking-wider">{s.label}</h2>
            </div>
            <TaskList
              tasks={grouped[s.key]}
              loading={false}
              emptyMessage=""
              onTaskSelect={selectTask}
            />
          </div>
        )
      })}
    </div>
  )
}
