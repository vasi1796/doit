import * as operations from '../db/operations'
import { useTasks } from '../hooks/useTasks'
import { useToast } from '../components/common/Toast'
import { EmptyState } from '../components/common/EmptyState'
import { PriorityFlag } from '../components/common/PriorityDot'
import { daysUntilTTL } from '../utils/date'
import type { Task } from '../api/types'

const TRASH_TTL_DAYS = 30

function TrashItem({ task }: { task: Task }) {
  const { toast } = useToast()
  // Use `updated_at` as a proxy for the deletion timestamp: tasks in trash
  // are only modified via the delete action, so the two are equivalent until
  // the API ships a dedicated `deleted_at` field.
  const days = daysUntilTTL(task.updated_at, TRASH_TTL_DAYS)
  const isUrgent = days !== null && days < 7

  const handleRestore = async () => {
    try {
      await operations.restoreTask(task.id)
      toast('Task restored', 'success')
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to restore', 'error')
    }
  }

  return (
    <div className="flex items-center gap-3 px-6 min-h-[44px] py-2 border-b border-separator">
      <PriorityFlag priority={task.priority} size={12} />
      <span className="flex-1 text-[15px] text-text-tertiary line-through truncate">{task.title}</span>
      {days !== null && (
        <span
          className={`text-[11px] font-medium px-2 py-0.5 rounded-full shrink-0 ${
            isUrgent ? 'text-danger bg-danger/10' : 'text-text-tertiary bg-bg-secondary'
          }`}
          title={`Auto-delete in ${days} ${days === 1 ? 'day' : 'days'}`}
        >
          {days}d
        </span>
      )}
      <button
        onClick={handleRestore}
        className="text-accent text-sm font-semibold px-2 min-h-[44px] hover:text-accent-hover transition-colors"
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
      <div className="px-6 pt-6 pb-2">
        <h1 className="text-[28px] font-semibold text-text-primary tracking-tight">Trash</h1>
      </div>
      {tasks.length > 0 && (
        <div className="mx-6 my-3 p-3 rounded-[10px] bg-warning/10 border border-warning/20 flex items-start gap-3">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-warning shrink-0 mt-px">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <p className="text-[13px] text-text-primary">
            Items in trash are permanently deleted after 30 days.
          </p>
        </div>
      )}
      {loading ? (
        <div className="space-y-1 px-6 py-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-[44px] bg-bg-secondary rounded-[10px] animate-pulse" />
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
