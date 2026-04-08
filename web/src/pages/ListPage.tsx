import { useParams, Navigate } from 'react-router'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { QuickAdd } from '../components/tasks/QuickAdd'

export function ListPage() {
  const { id } = useParams<{ id: string }>()
  const { tasks, loading } = useTasks({ list_id: id ?? '', is_completed: 'false' })
  const { lists, labels, selectTask } = useLayoutContext()

  if (!id) return <Navigate to="/inbox" replace />

  const list = lists.find((l) => l.id === id)
  const accentColor = list?.colour

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
            className="w-3.5 h-3.5 rounded-full shrink-0"
            style={{ backgroundColor: accentColor }}
          />
        )}
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight truncate">{list?.name || 'List'}</h1>
        <span className="ml-auto text-[13px] text-text-tertiary font-medium shrink-0">
          {tasks.length} {tasks.length === 1 ? 'task' : 'tasks'}
        </span>
      </div>
      <QuickAdd listId={id} lists={lists} labels={labels} />
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="No tasks in this list"
        onTaskSelect={selectTask}
      />
    </div>
  )
}
