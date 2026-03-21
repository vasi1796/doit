import { useState } from 'react'
import { useParams, Navigate } from 'react-router'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'

export function LabelPage() {
  const { id } = useParams<{ id: string }>()
  const { tasks, loading } = useTasks({ label_id: id ?? '', is_completed: 'false' })
  const { lists, labels } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  if (!id) return <Navigate to="/inbox" replace />

  const label = labels.find((l) => l.id === id)

  return (
    <div>
      <div className="px-4 pt-6 pb-2 flex items-center gap-3">
        {label?.colour && (
          <span className="w-3 h-3 rounded-sm" style={{ backgroundColor: label.colour }} />
        )}
        <h1 className="text-2xl font-semibold text-text-primary">{label?.name || 'Label'}</h1>
      </div>
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="No tasks with this label"
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
