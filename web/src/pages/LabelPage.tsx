import { useParams, Navigate } from 'react-router'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'

export function LabelPage() {
  const { id } = useParams<{ id: string }>()
  const { tasks, loading } = useTasks({ label_id: id ?? '', is_completed: 'false' })
  const { labels, selectTask } = useLayoutContext()

  if (!id) return <Navigate to="/inbox" replace />

  const label = labels.find((l) => l.id === id)
  const accentColor = label?.colour

  return (
    <div>
      {accentColor && (
        <div
          className="h-[3px] w-full"
          style={{ backgroundColor: accentColor }}
          aria-hidden="true"
        />
      )}
      <div className="px-6 pt-6 pb-2 flex items-center gap-3">
        {accentColor && (
          <span
            className="w-3.5 h-3.5 rounded-[4px] shrink-0"
            style={{ backgroundColor: accentColor }}
          />
        )}
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight truncate">{label?.name || 'Label'}</h1>
        <span className="ml-auto text-[13px] text-text-tertiary font-medium shrink-0">
          {tasks.length} {tasks.length === 1 ? 'task' : 'tasks'}
        </span>
      </div>
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="No tasks with this label"
        onTaskSelect={selectTask}
      />
    </div>
  )
}
