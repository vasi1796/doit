import { useState } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'

export function TodayPage() {
  const { tasks, loading } = useTasks({ is_completed: 'false' })
  const { lists } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const today = new Date().toISOString().split('T')[0]
  const overdueTasks = tasks.filter((t) => t.due_date && t.due_date < today)
  const todayTasks = tasks.filter((t) => t.due_date === today)

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-text-primary">Today</h1>
        <p className="text-sm text-text-secondary">
          {new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })}
        </p>
      </div>
      {overdueTasks.length > 0 && (
        <>
          <div className="px-5 pt-4 pb-1">
            <h2 className="text-sm font-semibold text-danger">Overdue</h2>
          </div>
          <TaskList
            tasks={overdueTasks}
            loading={false}
            emptyMessage=""
            onTaskSelect={setSelectedId}
          />
        </>
      )}
      <div className="px-5 pt-4 pb-1">
        <h2 className="text-sm font-semibold text-text-secondary">Today</h2>
      </div>
      <TaskList
        tasks={todayTasks}
        loading={loading}
        emptyMessage="Nothing due today"
        emptyHint="Set due dates on tasks to see them here"
        onTaskSelect={setSelectedId}
      />
      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} />
      )}
    </div>
  )
}
