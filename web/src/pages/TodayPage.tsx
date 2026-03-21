import { usePageTasks } from '../hooks/usePageTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'

export function TodayPage() {
  const { tasks, loading, refreshAll, selectedId, setSelectedId } = usePageTasks({ is_completed: 'false' })
  const { lists } = useLayoutContext()

  const today = new Date().toISOString().split('T')[0]
  const todayTasks = tasks.filter((t) => t.due_date === today)

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-text-primary">Today</h1>
        <p className="text-sm text-text-secondary">
          {new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })}
        </p>
      </div>
      <TaskList
        tasks={todayTasks}
        loading={loading}
        emptyMessage="Nothing due today"
        emptyHint="Set due dates on tasks to see them here"
        onTaskChanged={refreshAll}
        onTaskSelect={setSelectedId}
      />
      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} onChanged={refreshAll} />
      )}
    </div>
  )
}
