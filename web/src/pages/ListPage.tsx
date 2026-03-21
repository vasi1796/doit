import { useState } from 'react'
import { useParams, Navigate } from 'react-router'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'
import { QuickAdd } from '../components/tasks/QuickAdd'

export function ListPage() {
  const { id } = useParams<{ id: string }>()
  const { tasks, loading } = useTasks({ list_id: id ?? '', is_completed: 'false' })
  const { lists, labels } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  if (!id) return <Navigate to="/inbox" replace />

  const list = lists.find((l) => l.id === id)

  return (
    <div>
      <div className="px-4 pt-6 pb-2 flex items-center gap-3">
        {list?.colour && (
          <span className="w-3 h-3 rounded-full" style={{ backgroundColor: list.colour }} />
        )}
        <h1 className="text-2xl font-semibold text-text-primary">{list?.name || 'List'}</h1>
      </div>
      <QuickAdd listId={id} lists={lists} labels={labels} />
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="No tasks in this list"
        onTaskSelect={setSelectedId}
      />
      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} />
      )}
    </div>
  )
}
