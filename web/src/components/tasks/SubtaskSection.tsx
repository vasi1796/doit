import { useState, useRef } from 'react'
import * as operations from '../../db/operations'
import { useToast } from '../common/Toast'
import type { Subtask } from '../../api/types'

function SubtaskItem({ subtask, taskId }: { subtask: Subtask; taskId: string }) {
  const { toast } = useToast()
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState(subtask.title)

  const handleToggle = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      if (subtask.is_completed) {
        await operations.uncompleteSubtask(taskId, subtask.id)
      } else {
        await operations.completeSubtask(taskId, subtask.id)
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed', 'error')
    }
  }

  const handleSave = async () => {
    setEditing(false)
    const trimmed = editValue.trim()
    if (!trimmed || trimmed === subtask.title) {
      setEditValue(subtask.title)
      return
    }
    try {
      await operations.updateSubtaskTitle(taskId, subtask.id, trimmed)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to update', 'error')
      setEditValue(subtask.title)
    }
  }

  return (
    <div className="flex items-center gap-2 min-h-[36px] px-1 rounded hover:bg-gray-50 group">
      <button
        type="button"
        onClick={handleToggle}
        className={`w-4 h-4 rounded border shrink-0 flex items-center justify-center transition-colors ${
          subtask.is_completed ? 'bg-accent border-accent' : 'border-gray-300 hover:border-accent'
        }`}
      >
        {subtask.is_completed && (
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round">
            <path d="m5 12 5 5L20 7" />
          </svg>
        )}
      </button>
      {editing ? (
        <input
          type="text"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onBlur={handleSave}
          onKeyDown={(e) => { if (e.key === 'Enter') handleSave(); if (e.key === 'Escape') setEditing(false) }}
          className="flex-1 text-base outline-none border-b border-accent py-0.5"
          // eslint-disable-next-line jsx-a11y/no-autofocus
          autoFocus
        />
      ) : (
        <span
          onClick={() => !subtask.is_completed && setEditing(true)}
          onKeyDown={(e) => { if (e.key === 'Enter' && !subtask.is_completed) setEditing(true) }}
          role="button"
          tabIndex={subtask.is_completed ? -1 : 0}
          className={`flex-1 text-sm cursor-text ${subtask.is_completed ? 'line-through text-text-secondary' : ''}`}
        >
          {subtask.title}
        </span>
      )}
    </div>
  )
}

interface SubtaskSectionProps {
  taskId: string
  subtasks: Subtask[]
}

export function SubtaskSection({ taskId, subtasks }: SubtaskSectionProps) {
  const { toast } = useToast()
  const [newTitle, setNewTitle] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTitle.trim()) return
    try {
      await operations.createSubtask(taskId, {
        title: newTitle.trim(),
        position: Date.now().toString(),
      })
      setNewTitle('')
      setTimeout(() => inputRef.current?.focus(), 50)
    } catch (err) {
      toast(err instanceof Error ? err.message : 'Failed to add subtask', 'error')
    }
  }

  const completed = subtasks.filter(s => s.is_completed).length

  return (
    <div className="mb-4">
      <h3 className="text-xs font-medium text-text-secondary uppercase tracking-wide mb-2">
        Subtasks{subtasks.length > 0 && ` (${completed}/${subtasks.length})`}
      </h3>
      <div className="space-y-0.5">
        {subtasks.map((st) => (
          <SubtaskItem key={st.id} subtask={st} taskId={taskId} />
        ))}
        <form onSubmit={handleAdd} className="flex items-center gap-2 min-h-[36px] px-1">
          <span className="w-4 h-4 rounded border border-gray-200 shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Add subtask..."
            className="flex-1 text-base outline-none placeholder:text-text-secondary"
          />
          {newTitle.trim() && (
            <button type="submit" className="text-accent text-xs font-medium">Add</button>
          )}
        </form>
      </div>
    </div>
  )
}
