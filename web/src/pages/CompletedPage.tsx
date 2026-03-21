import { useState, useCallback } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'

export function CompletedPage() {
  const { tasks, loading, refresh } = useTasks({ is_completed: 'true' })
  const { lists, refreshCounts } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const refreshAll = useCallback(() => { refresh(); refreshCounts() }, [refresh, refreshCounts])

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
