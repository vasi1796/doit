import { useState, useCallback } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { TaskDetail } from '../components/tasks/TaskDetail'
import { QuickAdd } from '../components/tasks/QuickAdd'

export function InboxPage() {
  const { tasks, loading, refresh } = useTasks({ inbox: 'true', is_completed: 'false' })
  const { lists, labels, quickAddRef, refreshLists, refreshLabels, refreshCounts } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const refreshAll = useCallback(() => { refresh(); refreshCounts() }, [refresh, refreshCounts])

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-[#1d1d1f]">Inbox</h1>
      </div>
      <QuickAdd ref={quickAddRef} lists={lists} labels={labels} onCreated={refreshAll} onListsChanged={refreshLists} onLabelsChanged={refreshLabels} />
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="Your inbox is clear"
        emptyHint="Press Cmd+N to add a task"
        onTaskChanged={refreshAll}
        onTaskSelect={setSelectedId}
      />
      {selectedId && (
        <TaskDetail
          taskId={selectedId}
          lists={lists}
          onClose={() => setSelectedId(null)}
          onChanged={refreshAll}
          onListsChanged={refreshLists}
        />
      )}
    </div>
  )
}
