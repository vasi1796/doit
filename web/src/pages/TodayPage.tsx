import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'

export function TodayPage() {
  const { tasks, loading } = useTasks({ is_completed: 'false' })
  const { selectTask } = useLayoutContext()

  const today = new Date().toISOString().split('T')[0]
  const overdueTasks = tasks.filter((t) => t.due_date && t.due_date < today)
  const todayTasks = tasks.filter((t) => t.due_date === today)

  const totalDue = todayTasks.length + overdueTasks.length

  return (
    <div>
      <div className="px-6 pt-6 pb-2">
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight">Today</h1>
        <p className="text-[14px] text-text-secondary mt-0.5">
          {new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })}
          {totalDue > 0 && (
            <span className="text-text-tertiary"> · {totalDue} {totalDue === 1 ? 'task' : 'tasks'}</span>
          )}
        </p>
      </div>
      {overdueTasks.length > 0 && (
        <>
          <div className="px-6 pt-5 pb-1 flex items-center gap-2">
            <h2 className="text-[11px] font-semibold text-danger uppercase tracking-wider">Overdue</h2>
            <span className="text-[11px] font-medium text-danger bg-danger/10 px-1.5 py-px rounded-full min-w-[18px] text-center">
              {overdueTasks.length}
            </span>
          </div>
          <TaskList
            tasks={overdueTasks}
            loading={false}
            emptyMessage=""
            onTaskSelect={selectTask}
          />
        </>
      )}
      <div className="px-6 pt-5 pb-1">
        <h2 className="text-[11px] font-semibold text-text-tertiary uppercase tracking-wider">Today</h2>
      </div>
      <TaskList
        tasks={todayTasks}
        loading={loading}
        emptyMessage="Nothing due today"
        emptyHint="Set due dates on tasks to see them here"
        onTaskSelect={selectTask}
      />
    </div>
  )
}
