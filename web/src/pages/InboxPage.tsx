import { useState } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'
import { QuickAdd } from '../components/tasks/QuickAdd'

export function InboxPage() {
  const { tasks, loading } = useTasks({ inbox: 'true', is_completed: 'false' })
  const { lists, labels, quickAddRef } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-text-primary">Inbox</h1>
      </div>
      <QuickAdd ref={quickAddRef} lists={lists} labels={labels} />
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="Your inbox is clear"
        emptyHint="Tap New task to get started"
        onTaskSelect={setSelectedId}
      />
      {selectedId && (
        <TaskDetail
          taskId={selectedId}
          lists={lists}
          onClose={() => setSelectedId(null)}
        />
      )}
    </div>
  )
}
