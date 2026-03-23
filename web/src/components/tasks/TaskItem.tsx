import { useState } from 'react'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import * as operations from '../../db/operations'
import { useToast } from '../common/Toast'
import { InlineMarkdown } from '../common/InlineMarkdown'
import { PriorityFlag } from '../common/PriorityDot'
import { formatDueDate } from '../../utils/date'
import { PRIORITY_COLORS } from '../../constants'
import type { Task } from '../../api/types'

interface TaskItemProps {
  task: Task
  onSelect: (id: string) => void
}

export function SortableTaskItem({ task, onSelect }: TaskItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: task.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 50 : undefined,
    position: isDragging ? 'relative' as const : undefined,
  }

  return (
    <div ref={setNodeRef} style={style}>
      <TaskItem
        task={task}
        onSelect={onSelect}
        isDragging={isDragging}
        dragHandleProps={{ ...attributes, ...listeners }}
      />
    </div>
  )
}

interface TaskItemInternalProps {
  task: Task
  onSelect: (id: string) => void
  isDragging?: boolean
  dragHandleProps?: Record<string, unknown>
}

function TaskItem({ task, onSelect, isDragging, dragHandleProps }: TaskItemInternalProps) {
  const { toast } = useToast()
  const [completing, setCompleting] = useState(false)
  const [fading, setFading] = useState(false)

  const handleToggle = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (completing) return
    try {
      if (task.is_completed) {
        await operations.uncompleteTask(task.id)
      } else {
        setCompleting(true)
        await operations.completeTask(task.id)
        if (task.recurrence_rule) toast('Done! Next occurrence created', 'success')
        setTimeout(() => {
          setFading(true)
        }, 400)
      }
    } catch (err) {
      setCompleting(false)
      setFading(false)
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  const due = task.due_date ? formatDueDate(task.due_date, task.due_time) : null
  const checked = task.is_completed || completing
  const priorityColor = PRIORITY_COLORS[task.priority]
  const labels = task.labels || []
  const subtasks = task.subtasks || []
  const subtasksDone = subtasks.filter(s => s.is_completed).length

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onSelect(task.id)}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') onSelect(task.id) }}
      className={`w-full flex items-start gap-3 px-5 py-3 hover:bg-[#f8f8fa] text-left transition-all duration-300 relative cursor-pointer ${
        fading ? 'opacity-0 max-h-0 py-0 overflow-hidden' : 'opacity-100'
      } ${isDragging ? 'bg-white shadow-lg rounded-lg' : ''}`}
    >
      {/* Drag handle */}
      <button
        type="button"
        aria-label="Drag to reorder"
        className="w-[44px] min-h-[44px] -ml-5 -my-3 self-stretch flex items-center justify-center shrink-0 touch-none cursor-grab active:cursor-grabbing text-text-tertiary hover:text-text-secondary"
        {...dragHandleProps}
        onClick={(e) => e.stopPropagation()}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
          <circle cx="9" cy="6" r="1.5" />
          <circle cx="15" cy="6" r="1.5" />
          <circle cx="9" cy="12" r="1.5" />
          <circle cx="15" cy="12" r="1.5" />
          <circle cx="9" cy="18" r="1.5" />
          <circle cx="15" cy="18" r="1.5" />
        </svg>
      </button>

      {priorityColor && (
        <span className="absolute left-0 top-3 bottom-3 w-[4px] rounded-r-full" style={{ backgroundColor: priorityColor }} />
      )}

      <button
        type="button"
        onClick={handleToggle}
        aria-label={checked ? 'Mark incomplete' : 'Mark complete'}
        className={`w-[22px] h-[22px] mt-0.5 rounded-full border-2 shrink-0 flex items-center justify-center cursor-pointer transition-all duration-200 ${
          checked ? 'bg-accent border-accent scale-110' : 'border-[#c7c7cc] hover:border-accent'
        }`}
      >
        {checked && (
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" className="animate-[check_0.2s_ease-out]">
            <path d="m5 12 5 5L20 7" />
          </svg>
        )}
      </button>

      <div className="flex-1 min-w-0 py-0.5">
        <div className="flex items-center gap-2">
          <InlineMarkdown
            text={task.title}
            className={`flex-1 text-[15px] leading-snug truncate ${checked ? 'line-through text-text-secondary' : 'text-text-primary'}`}
          />
          <div className="flex items-center gap-1.5 shrink-0">
            <PriorityFlag priority={task.priority} size={13} />
            {due && (
              <span className={`text-[12px] whitespace-nowrap ${due.overdue ? 'text-danger font-medium' : 'text-text-secondary'}`}>
                {due.text}
              </span>
            )}
          </div>
        </div>

        {task.description && (
          <InlineMarkdown text={task.description} className="text-[12px] text-text-secondary block truncate mt-0.5" />
        )}

        {(labels.length > 0 || task.recurrence_rule || subtasks.length > 0) && (
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            {task.recurrence_rule && (
              <span className="text-[11px] text-text-secondary flex items-center gap-0.5">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M17 1l4 4-4 4" /><path d="M3 11V9a4 4 0 0 1 4-4h14" />
                  <path d="M7 23l-4-4 4-4" /><path d="M21 13v2a4 4 0 0 1-4 4H3" />
                </svg>
                {task.recurrence_rule}
              </span>
            )}
            {subtasks.length > 0 && (
              <span className="text-[11px] text-text-secondary flex items-center gap-1">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                  <path d="M9 11l3 3L22 4" /><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
                </svg>
                {subtasksDone}/{subtasks.length}
              </span>
            )}
            {labels.map((label) => (
              <span
                key={label.id}
                className="text-[11px] px-2 py-0.5 rounded-full font-medium"
                style={{
                  backgroundColor: (label.colour || '#86868b') + '18',
                  color: label.colour || '#86868b',
                }}
              >
                {label.name}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
