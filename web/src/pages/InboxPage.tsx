import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskList } from '../components/tasks/TaskList'
import { QuickAdd } from '../components/tasks/QuickAdd'

export function InboxPage() {
  const { tasks, loading } = useTasks({ inbox: 'true', is_completed: 'false' })
  const { lists, labels, quickAddRef, selectTask } = useLayoutContext()

  return (
    <div>
      <div className="px-6 pt-6 pb-2">
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight">Inbox</h1>
      </div>
      <QuickAdd ref={quickAddRef} lists={lists} labels={labels} />
      <TaskList
        tasks={tasks}
        loading={loading}
        emptyMessage="Your inbox is clear"
        emptyHint="Tap New task to get started"
        onTaskSelect={selectTask}
      />
    </div>
  )
}
