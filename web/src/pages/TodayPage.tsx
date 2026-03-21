import { useState, useCallback } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'

export function TodayPage() {
  const { tasks, loading, refresh } = useTasks({ is_completed: 'false' })
  const { lists, refreshCounts } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const refreshAll = useCallback(() => { refresh(); refreshCounts() }, [refresh, refreshCounts])

  const today = new Date().toISOString().split('T')[0]
  const todayTasks = tasks.filter((t) => t.due_date === today)

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-[#1d1d1f]">Today</h1>
        <p className="text-sm text-[#86868b]">
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
