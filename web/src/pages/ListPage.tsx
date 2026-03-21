import { useParams, Navigate } from 'react-router'
import { usePageTasks } from '../hooks/usePageTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'
import { QuickAdd } from '../components/tasks/QuickAdd'

export function ListPage() {
  const { id } = useParams<{ id: string }>()
  if (!id) return <Navigate to="/inbox" replace />

  const { tasks, loading, refreshAll, selectedId, setSelectedId } = usePageTasks({ list_id: id, is_completed: 'false' })
  const { lists, labels, refreshLists, refreshLabels } = useLayoutContext()
  const list = lists.find((l) => l.id === id)

  return (
    <div>
      <div className="px-4 pt-6 pb-2 flex items-center gap-3">
        {list?.colour && (
          <span className="w-3 h-3 rounded-full" style={{ backgroundColor: list.colour }} />
        )}
        <h1 className="text-2xl font-semibold text-[#1d1d1f]">{list?.name || 'List'}</h1>
      </div>
      <QuickAdd listId={id} lists={lists} labels={labels} onCreated={refreshAll} onListsChanged={refreshLists} onLabelsChanged={refreshLabels} />
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="No tasks in this list"
        onTaskChanged={refreshAll}
        onTaskSelect={setSelectedId}
      />
      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} onChanged={refreshAll} onListsChanged={refreshLists} />
      )}
    </div>
  )
}
