import type { Task } from '../../api/types'
import { TaskItem } from './TaskItem'
import { EmptyState } from '../common/EmptyState'

interface TaskListProps {
  tasks: Task[]
  loading: boolean
  emptyMessage?: string
  emptyHint?: string
  emptyAction?: { label: string; onClick: () => void }
  onTaskChanged: () => void
  onTaskSelect: (id: string) => void
}

export function TaskList({ tasks, loading, emptyMessage, emptyHint, emptyAction, onTaskChanged, onTaskSelect }: TaskListProps) {
  if (loading) {
    return (
      <div className="space-y-1 px-4 py-2">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-[44px] bg-gray-100 rounded animate-pulse" />
        ))}
      </div>
    )
  }

  if (tasks.length === 0) {
    return <EmptyState message={emptyMessage} hint={emptyHint} action={emptyAction} />
  }

  return (
    <div className="divide-y divide-gray-100">
      {tasks.map((task) => (
        <TaskItem
          key={task.id}
          task={task}
          onChanged={onTaskChanged}
          onSelect={onTaskSelect}
        />
      ))}
    </div>
  )
}
