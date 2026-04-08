import { useState, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import * as operations from '../../db/operations'
import { useToast } from '../common/Toast'
import type { Subtask } from '../../api/types'

function SubtaskItem({ subtask, taskId }: { subtask: Subtask; taskId: string }) {
  const { toast } = useToast()
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState(subtask.title)
  const [completing, setCompleting] = useState(false)

  const handleToggle = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      if (subtask.is_completed) {
        await operations.uncompleteSubtask(taskId, subtask.id)
      } else {
        setCompleting(true)
        await operations.completeSubtask(taskId, subtask.id)
      }
    } catch (err) {
      setCompleting(false)
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

  const checked = subtask.is_completed || completing

  return (
    <div className="flex items-center gap-2 min-h-[36px] px-1 rounded-[8px] hover:bg-bg-secondary group transition-colors">
      <button
        type="button"
        onClick={handleToggle}
        aria-label={checked ? 'Mark subtask incomplete' : 'Mark subtask complete'}
        className={`w-4 h-4 rounded-[4px] border shrink-0 flex items-center justify-center transition-all duration-200 ${
          checked ? 'bg-accent border-accent scale-110' : 'border-text-quaternary hover:border-accent'
        }`}
      >
        {checked && (
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" className="animate-[check_0.2s_ease-out]">
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
          className={`flex-1 text-[14px] cursor-text transition-colors duration-200 ${checked ? 'line-through text-text-tertiary' : 'text-text-primary'}`}
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
      <h3 className="text-[11px] font-semibold text-text-tertiary uppercase tracking-wider mb-2">
        Subtasks{subtasks.length > 0 && ` · ${completed}/${subtasks.length}`}
      </h3>
      <div className="space-y-0.5">
        <AnimatePresence initial={false}>
          {subtasks.map((st) => (
            <motion.div
              key={st.id}
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.2, ease: 'easeOut' }}
              style={{ overflow: 'hidden' }}
            >
              <SubtaskItem subtask={st} taskId={taskId} />
            </motion.div>
          ))}
        </AnimatePresence>
        <form onSubmit={handleAdd} className="flex items-center gap-2 min-h-[36px] px-1">
          <span className="w-4 h-4 rounded-[4px] border border-separator shrink-0" aria-hidden="true" />
          <input
            ref={inputRef}
            type="text"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Add subtask…"
            aria-label="Add subtask"
            className="flex-1 text-[16px] outline-none bg-transparent text-text-primary placeholder:text-text-tertiary"
          />
          {newTitle.trim() && (
            <button type="submit" className="text-accent text-[13px] font-semibold">Add</button>
          )}
        </form>
      </div>
    </div>
  )
}
