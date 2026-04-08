import { useCallback } from 'react'
import { AnimatePresence } from 'framer-motion'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  TouchSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import type { Task } from '../../api/types'
import { SortableTaskItem } from './TaskItem'
import { EmptyState } from '../common/EmptyState'
import { between } from '../../crdt/fracindex'
import * as operations from '../../db/operations'

interface TaskListProps {
  tasks: Task[]
  loading: boolean
  emptyMessage?: string
  emptyHint?: string
  emptyAction?: { label: string; onClick: () => void }
  onTaskSelect: (id: string) => void
}

export function TaskList({ tasks, loading, emptyMessage, emptyHint, emptyAction, onTaskSelect }: TaskListProps) {
  const pointerSensor = useSensor(PointerSensor, {
    activationConstraint: { distance: 8 },
  })
  const touchSensor = useSensor(TouchSensor, {
    activationConstraint: { delay: 500, tolerance: 5 },
  })
  const keyboardSensor = useSensor(KeyboardSensor)
  const sensors = useSensors(pointerSensor, touchSensor, keyboardSensor)

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event
      if (!over || active.id === over.id) return

      const oldIndex = tasks.findIndex((t) => t.id === active.id)
      const newIndex = tasks.findIndex((t) => t.id === over.id)
      if (oldIndex === -1 || newIndex === -1) return

      // Compute a position between the new neighbors for the dragged task only.
      const reordered = tasks.filter((_, i) => i !== oldIndex)
      const insertAt = newIndex > oldIndex ? newIndex - 1 : newIndex

      const prevPos = insertAt > 0 ? reordered[insertAt - 1].position : ''
      const nextPos = insertAt < reordered.length ? reordered[insertAt].position : ''
      const newPosition = between(prevPos, nextPos)

      operations.updateTask(String(active.id), { position: newPosition })
    },
    [tasks],
  )

  if (loading) {
    return (
      <div className="space-y-1 px-4 py-2">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-[44px] bg-bg-secondary rounded-[8px] animate-pulse" />
        ))}
      </div>
    )
  }

  if (tasks.length === 0) {
    return <EmptyState message={emptyMessage} hint={emptyHint} action={emptyAction} />
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
        <div className="divide-y divide-separator">
          <AnimatePresence initial={false}>
            {tasks.map((task) => (
              <SortableTaskItem
                key={task.id}
                task={task}
                onSelect={onTaskSelect}
              />
            ))}
          </AnimatePresence>
        </div>
      </SortableContext>
    </DndContext>
  )
}
