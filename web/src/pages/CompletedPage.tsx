import { usePageTasks } from '../hooks/usePageTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'

export function CompletedPage() {
  const { tasks, loading, refreshAll, selectedId, setSelectedId } = usePageTasks({ is_completed: 'true' })
  const { lists } = useLayoutContext()

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-[#1d1d1f]">Completed</h1>
      </div>
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="No completed tasks"
        onTaskChanged={refreshAll}
        onTaskSelect={setSelectedId}
      />
      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} onChanged={refreshAll} />
      )}
    </div>
  )
}
