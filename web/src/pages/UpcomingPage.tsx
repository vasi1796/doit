import { useState, useCallback } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'

function getUpcomingDays(): string[] {
  const days: string[] = []
  const today = new Date()
  for (let i = 1; i <= 7; i++) {
    const d = new Date(today)
    d.setDate(d.getDate() + i)
    days.push(d.toISOString().split('T')[0])
  }
  return days
}

function formatDayHeader(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00')
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)

  if (date.getTime() === tomorrow.getTime()) return 'Tomorrow'
  return date.toLocaleDateString('en-US', { weekday: 'long', month: 'short', day: 'numeric' })
}

export function UpcomingPage() {
  const { tasks, loading, refresh } = useTasks({ is_completed: 'false' })
  const { lists, refreshCounts } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const refreshAll = useCallback(() => { refresh(); refreshCounts() }, [refresh, refreshCounts])

  const days = getUpcomingDays()
  const grouped = days
    .map((day) => ({
      date: day,
      label: formatDayHeader(day),
      tasks: tasks.filter((t) => t.due_date === day),
    }))
    .filter((g) => g.tasks.length > 0)

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-[#1d1d1f]">Upcoming</h1>
      </div>
      {loading ? (
        <div className="px-4 py-8 text-center text-[#86868b] text-sm">Loading...</div>
      ) : grouped.length === 0 ? (
        <div className="px-4 py-16 text-center text-[#86868b] text-sm">Nothing coming up this week</div>
      ) : (
        grouped.map((group) => (
          <div key={group.date}>
            <div className="px-4 pt-4 pb-1">
              <h2 className="text-sm font-semibold text-[#86868b]">{group.label}</h2>
            </div>
            <TaskList
              tasks={group.tasks}
              loading={false}
              onTaskChanged={refreshAll}
              onTaskSelect={setSelectedId}
            />
          </div>
        ))
      )}
      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} onChanged={refreshAll} />
      )}
    </div>
  )
}
