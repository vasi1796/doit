import * as operations from '../db/operations'
import { useTasks } from '../hooks/useTasks'
import { useToast } from '../components/common/Toast'
import { EmptyState } from '../components/common/EmptyState'
import { PriorityFlag } from '../components/common/PriorityDot'
import type { Task } from '../api/types'

function TrashItem({ task }: { task: Task }) {
  const { toast } = useToast()

  const handleRestore = async () => {
    try {
      await operations.restoreTask(task.id)
      toast('Task restored', 'success')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to restore', 'error')
    }
  }

  return (
    <div className="flex items-center gap-3 px-4 min-h-[44px] py-2 border-b border-gray-100">
      <PriorityFlag priority={task.priority} size={12} />
      <span className="flex-1 text-[15px] text-text-secondary line-through truncate">{task.title}</span>
      <button
        onClick={handleRestore}
        className="text-accent text-sm font-medium px-2 min-h-[44px]"
      >
        Restore
      </button>
    </div>
  )
}

export function TrashPage() {
  const { tasks, loading } = useTasks({ is_deleted: 'true' })

  return (
    <div>
      <div className="px-4 pt-6 pb-2">
        <h1 className="text-2xl font-semibold text-text-primary">Trash</h1>
        <p className="text-sm text-text-secondary">Tasks are permanently deleted after 30 days</p>
      </div>
      {loading ? (
        <div className="space-y-1 px-4 py-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-[44px] bg-gray-100 rounded animate-pulse" />
          ))}
        </div>
      ) : tasks.length === 0 ? (
        <EmptyState message="Trash is empty" hint="Deleted tasks will appear here" />
      ) : (
        tasks.map((task) => (
          <TrashItem key={task.id} task={task} />
        ))
      )}
    </div>
  )
}
