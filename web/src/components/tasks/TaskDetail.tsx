import { useState, useEffect, useRef, useCallback } from 'react'
import * as operations from '../../db/operations'
import { useTaskDetail } from '../../hooks/useTaskDetail'
import { useLabels } from '../../hooks/useLabels'
import { useToast } from '../common/Toast'
import { MarkdownEditor } from '../common/MarkdownEditor'
import { InlineMarkdown } from '../common/InlineMarkdown'
import { TaskProperties } from './TaskProperties'
import { SubtaskSection } from './SubtaskSection'
import { LabelsSection } from './LabelsSection'
import type { List } from '../../api/types'

interface TaskDetailProps {
  taskId: string
  lists?: List[]
  onClose: () => void
  /**
   * 'modal' (default) — centered fixed overlay with backdrop. Use on mobile/tablet.
   * 'panel' — fills its parent (no backdrop, no centering). Use as a third column
   * in a >=1024px desktop layout.
   */
  variant?: 'modal' | 'panel'
}

export function TaskDetail({ taskId, lists, onClose, variant = 'modal' }: TaskDetailProps) {
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

  const descTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const resolvedTaskId = task?.id
  const saveDescription = useCallback((val: string) => {
    if (descTimerRef.current) clearTimeout(descTimerRef.current)
    descTimerRef.current = setTimeout(() => {
      if (resolvedTaskId) operations.updateTask(resolvedTaskId, { description: val })
    }, 500)
  }, [resolvedTaskId])

  // Clear debounce timer on unmount
  useEffect(() => () => {
    if (descTimerRef.current) clearTimeout(descTimerRef.current)
  }, [])

  if (loading || !task) {
    if (variant === 'panel') {
      return (
        <div className="h-full bg-bg border-l border-separator p-6">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-bg-secondary rounded w-3/4" />
            <div className="h-20 bg-bg-secondary rounded" />
          </div>
        </div>
      )
    }
    return (
      <div className="fixed inset-0 bg-[rgba(0,0,0,0.35)] flex items-center justify-center z-50" onClick={onClose} aria-hidden="true">
        <div className="bg-bg-elevated rounded-[14px] p-6 w-full max-w-lg mx-4 shadow-modal border border-separator">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-bg-secondary rounded w-3/4" />
            <div className="h-20 bg-bg-secondary rounded" />
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
      toast('Task deleted', 'success', { label: 'Undo', onClick: () => operations.restoreTask(task.id) })
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

  const body = (
    <>
      {/* Header */}
      <div className="flex items-center gap-3 mb-4">
        <button
          onClick={handleToggleComplete}
          aria-label={task.is_completed ? 'Mark incomplete' : 'Mark complete'}
          className={`w-5 h-5 rounded-full border-2 shrink-0 flex items-center justify-center transition-all duration-200 ${
            task.is_completed
              ? 'bg-accent border-accent scale-110'
              : 'border-text-quaternary hover:border-accent'
          }`}
        >
          {task.is_completed && (
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" className="animate-[check_0.2s_ease-out]">
              <path d="m5 12 5 5L20 7" />
            </svg>
          )}
        </button>

        <div className="flex-1 min-w-0">
          {editingTitle ? (
            <input
              ref={titleRef}
              type="text"
              value={titleValue}
              onChange={(e) => setTitleValue(e.target.value)}
              onBlur={handleTitleSave}
              onKeyDown={(e) => { if (e.key === 'Enter') handleTitleSave() }}
              aria-label="Task title"
              className="text-lg font-semibold text-text-primary outline-none border-b-2 border-accent w-full py-0.5 bg-transparent"
              // eslint-disable-next-line jsx-a11y/no-autofocus
              autoFocus
            />
          ) : (
            // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-noninteractive-element-interactions
            <h2
              onClick={() => { setEditingTitle(true); setTimeout(() => titleRef.current?.select(), 10) }}
              className="text-lg font-semibold text-text-primary cursor-text hover:bg-bg-secondary rounded-[8px] px-1 -mx-1 py-0.5 transition-colors"
            >
              <InlineMarkdown text={task.title} />
            </h2>
          )}
        </div>

        <button onClick={onClose} aria-label="Close" className="text-text-secondary hover:text-text-primary min-w-[44px] min-h-[44px] flex items-center justify-center -m-2 rounded-[8px] hover:bg-bg-secondary transition-colors">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
            <path d="M18 6 6 18M6 6l12 12" />
          </svg>
        </button>
      </div>

      <TaskProperties task={task} lists={lists} onSave={save} />

      <div className="border border-separator rounded-[10px] p-3 mb-4 focus-within:border-accent transition-colors bg-bg">
        <MarkdownEditor
          value={task.description || ''}
          onChange={saveDescription}
          placeholder="Add notes…"
        />
      </div>

      <SubtaskSection taskId={task.id} subtasks={task.subtasks || []} />

      <LabelsSection
        taskId={task.id}
        taskLabels={task.labels || []}
        allLabels={allLabels}
      />

      <div className="flex justify-end pt-2 border-t border-separator">
        <button
          onClick={handleDelete}
          className="text-danger text-sm px-3 hover:bg-danger/10 rounded-[8px] transition-colors min-h-[44px]"
        >
          Delete
        </button>
      </div>
    </>
  )

  if (variant === 'panel') {
    return (
      <aside
        className="h-full w-full bg-bg border-l border-separator overflow-y-auto p-6"
        aria-label="Task detail"
      >
        {body}
      </aside>
    )
  }

  return (
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-noninteractive-element-interactions
    <div className="fixed inset-0 bg-[rgba(0,0,0,0.35)] flex items-center justify-center z-50 animate-[fade-in_0.15s_ease-out]" onClick={onClose} role="dialog" aria-modal="true" aria-label="Task detail">
      {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions */}
      <div
        className="bg-bg-elevated rounded-[14px] p-4 sm:p-6 w-full max-w-lg mx-3 sm:mx-4 shadow-modal border border-separator max-h-[85vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        {body}
      </div>
    </div>
  )
}
