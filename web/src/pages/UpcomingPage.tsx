import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'

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

function formatDayHeader(dateStr: string): { primary: string; secondary: string; isTomorrow: boolean } {
  const date = new Date(dateStr + 'T00:00:00')
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)

  const isTomorrow = date.getTime() === tomorrow.getTime()
  if (isTomorrow) {
    return {
      primary: 'Tomorrow',
      secondary: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
      isTomorrow: true,
    }
  }
  return {
    primary: date.toLocaleDateString('en-US', { weekday: 'long' }),
    secondary: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    isTomorrow: false,
  }
}

export function UpcomingPage() {
  const { tasks, loading } = useTasks({ is_completed: 'false' })
  const { selectTask } = useLayoutContext()

  const days = getUpcomingDays()
  const grouped = days
    .map((day) => ({
      date: day,
      header: formatDayHeader(day),
      tasks: tasks.filter((t) => t.due_date === day),
    }))
    .filter((g) => g.tasks.length > 0)

  return (
    <div>
      <div className="px-6 pt-6 pb-2">
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight">Upcoming</h1>
      </div>
      {loading ? (
        <div className="px-4 py-8 text-center text-text-tertiary text-sm">Loading…</div>
      ) : grouped.length === 0 ? (
        <div className="px-4 py-16 text-center text-text-tertiary text-sm">Nothing coming up this week</div>
      ) : (
        grouped.map((group) => (
          <div key={group.date}>
            <div className="px-6 pt-5 pb-1 flex items-baseline gap-2">
              {group.header.isTomorrow ? (
                <>
                  <span className="w-1.5 h-1.5 rounded-full bg-accent shrink-0" aria-hidden="true" />
                  <h2 className="text-[17px] font-semibold text-accent">{group.header.primary}</h2>
                  <span className="text-[12px] text-text-tertiary">{group.header.secondary}</span>
                </>
              ) : (
                <>
                  <h2 className="text-[11px] font-semibold text-text-tertiary uppercase tracking-wider">{group.header.primary}</h2>
                  <span className="text-[11px] text-text-tertiary">· {group.header.secondary}</span>
                </>
              )}
            </div>
            <TaskList
              tasks={group.tasks}
              loading={false}
              onTaskSelect={selectTask}
            />
          </div>
        ))
      )}
    </div>
  )
}
