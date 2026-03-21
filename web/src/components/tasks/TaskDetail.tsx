import { useState, useEffect, useRef } from 'react'
import * as operations from '../../db/operations'
import { useTaskDetail } from '../../hooks/useTaskDetail'
import { useLabels } from '../../hooks/useLabels'
import { useToast } from '../common/Toast'
import { TaskProperties } from './TaskProperties'
import { SubtaskSection } from './SubtaskSection'
import { LabelsSection } from './LabelsSection'
import type { List } from '../../api/types'

interface TaskDetailProps {
  taskId: string
  lists?: List[]
  onClose: () => void
}

export function TaskDetail({ taskId, lists, onClose }: TaskDetailProps) {
  const { task, loading } = useTaskDetail(taskId)
  const { labels: allLabels } = useLabels()
  const { toast } = useToast()
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleValue, setTitleValue] = useState('')
  const titleRef = useRef<HTMLInputElement>(null)

  const [prevTask, setPrevTask] = useState(task)
  if (task && task !== prevTask) {
    setPrevTask(task)
    setTitleValue(task.title)
  }

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onClose])

  if (loading || !task) {
    return (
      <div className="fixed inset-0 bg-black/20 flex items-center justify-center z-50" onClick={onClose} aria-hidden="true">
        <div className="bg-white rounded-xl p-6 w-full max-w-lg mx-4 shadow-xl">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-gray-100 rounded w-3/4" />
            <div className="h-20 bg-gray-100 rounded" />
          </div>
        </div>
      </div>
    )
  }

  const save = async (field: string, value: unknown) => {
    try {
      await operations.updateTask(task.id, { [field]: value })
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to save', 'error')
    }
  }

  const handleTitleSave = () => {
    setEditingTitle(false)
    const trimmed = titleValue.trim()
    if (trimmed && trimmed !== task.title) {
      save('title', trimmed)
    } else {
      setTitleValue(task.title)
    }
  }

  const handleDelete = async () => {
    try {
      await operations.deleteTask(task.id)
      toast('Task deleted', 'success')
      onClose()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to delete', 'error')
    }
  }

  const handleToggleComplete = async () => {
    try {
      if (task.is_completed) {
        await operations.uncompleteTask(task.id)
      } else {
        const openSubtasks = (task.subtasks || []).filter(s => !s.is_completed)
        await Promise.allSettled(openSubtasks.map(st => operations.completeSubtask(task.id, st.id)))
        await operations.completeTask(task.id)
        toast(task.recurrence_rule ? 'Done! Next occurrence created' : 'Task completed', 'success')
      }
      onClose()
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to update', 'error')
    }
  }

  return (
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-noninteractive-element-interactions
    <div className="fixed inset-0 bg-black/20 flex items-center justify-center z-50 animate-[fade-in_0.15s_ease-out]" onClick={onClose} role="dialog" aria-modal="true" aria-label="Task detail">
      {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
      <div
        className="bg-white rounded-xl p-6 w-full max-w-lg mx-4 shadow-xl max-h-[85vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center gap-3 mb-4">
          <button
            onClick={handleToggleComplete}
            aria-label={task.is_completed ? 'Mark incomplete' : 'Mark complete'}
            className={`w-5 h-5 rounded-full border-2 shrink-0 flex items-center justify-center transition-all duration-200 ${
              task.is_completed
                ? 'bg-accent border-accent scale-110'
                : 'border-gray-300 hover:border-accent'
            }`}
          >
            {task.is_completed && (
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" className="animate-[check_0.2s_ease-out]">
                <path d="m5 12 5 5L20 7" />
              </svg>
            )}
          </button>

          <div className="flex-1">
            {editingTitle ? (
              <input
                ref={titleRef}
                type="text"
                value={titleValue}
                onChange={(e) => setTitleValue(e.target.value)}
                onBlur={handleTitleSave}
                onKeyDown={(e) => { if (e.key === 'Enter') handleTitleSave() }}
                aria-label="Task title"
                className="text-lg font-medium text-text-primary outline-none border-b-2 border-accent w-full py-0.5"
                // eslint-disable-next-line jsx-a11y/no-autofocus
                autoFocus
              />
            ) : (
              // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-noninteractive-element-interactions
              <h2
                onClick={() => { setEditingTitle(true); setTimeout(() => titleRef.current?.select(), 10) }}
                className="text-lg font-medium text-text-primary cursor-text hover:bg-gray-50 rounded px-1 -mx-1 py-0.5"
              >
                {task.title}
              </h2>
            )}
          </div>

          <button onClick={onClose} aria-label="Close" className="text-text-secondary hover:text-text-primary min-w-[44px] min-h-[44px] flex items-center justify-center -m-2">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        <TaskProperties task={task} lists={lists} onSave={save} />

        <textarea
          defaultValue={task.description || ''}
          onBlur={(e) => {
            if (e.target.value !== (task.description || '')) save('description', e.target.value)
          }}
          placeholder="Add notes..."
          aria-label="Task notes"
          className="w-full min-h-[80px] text-base text-text-primary placeholder:text-text-secondary outline-none resize-none border border-gray-200 rounded-lg p-3 mb-4 focus:border-accent"
        />

        <SubtaskSection taskId={task.id} subtasks={task.subtasks || []} />

        <LabelsSection
          taskId={task.id}
          taskLabels={task.labels || []}
          allLabels={allLabels}
        />

        <div className="flex justify-end pt-2 border-t border-gray-100">
          <button
            onClick={handleDelete}
            className="text-danger text-sm px-3 hover:bg-red-50 rounded-lg transition-colors min-h-[44px]"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}
