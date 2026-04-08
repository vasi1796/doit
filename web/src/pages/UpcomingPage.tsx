import { useMemo } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { nextNDays, formatDayGroupHeader } from '../utils/date'

export function UpcomingPage() {
  const { tasks, loading } = useTasks({ is_completed: 'false' })
  const { selectTask } = useLayoutContext()

  // `days` is stable for the lifetime of the page mount; `grouped` re-derives
  // only when the task list changes.
  const days = useMemo(() => nextNDays(7), [])
  const grouped = useMemo(
    () =>
      days
        .map((day) => ({
          date: day,
          header: formatDayGroupHeader(day),
          tasks: tasks.filter((t) => t.due_date === day),
        }))
        .filter((g) => g.tasks.length > 0),
    [days, tasks],
  )

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
