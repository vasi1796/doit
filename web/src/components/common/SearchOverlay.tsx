import { useState, useEffect, useRef, useMemo } from 'react'
import { useTasks } from '../../hooks/useTasks'
import { InlineMarkdown } from './InlineMarkdown'
import { PriorityFlag } from './PriorityDot'
import { formatDueDate } from '../../utils/date'
import { COLORS, PRIORITY_COLORS } from '../../constants'
import type { Task } from '../../api/types'

interface SearchOverlayProps {
  onClose: () => void
  onSelectTask: (id: string) => void
}

export function SearchOverlay({ onClose, onSelectTask }: SearchOverlayProps) {
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const { tasks } = useTasks({ is_completed: 'false' })
  const { tasks: completedTasks } = useTasks({ is_completed: 'true' })

  const allTasks = useMemo(() => [...tasks, ...completedTasks], [tasks, completedTasks])

  const results = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return []

    return allTasks.filter((t) => {
      if (t.title.toLowerCase().includes(q)) return true
      if (t.description?.toLowerCase().includes(q)) return true
      if (t.labels?.some((l) => l.name.toLowerCase().includes(q))) return true
      return false
    }).slice(0, 20)
  }, [query, allTasks])

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  // Reset active index when query changes
  const handleQueryChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(e.target.value)
    setActiveIndex(0)
  }

  // Keyboard navigation
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIndex((i) => Math.min(i + 1, results.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIndex((i) => Math.max(i - 1, 0))
      } else if (e.key === 'Enter' && results.length > 0) {
        e.preventDefault()
        onSelectTask(results[activeIndex].id)
        onClose()
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onClose, onSelectTask, results, activeIndex])

  return (
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-noninteractive-element-interactions
    <div
      className="fixed inset-0 bg-[rgba(0,0,0,0.35)] z-[70] flex items-start justify-center pt-[12vh] animate-[fade-in_0.1s_ease-out]"
      role="presentation"
      onClick={onClose}
    >
      {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-noninteractive-element-interactions */}
      <div
        className="w-full max-w-[560px] mx-4 bg-bg-elevated rounded-[14px] shadow-modal border border-separator overflow-hidden max-h-[70vh] flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-label="Search tasks"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 border-b border-separator shrink-0">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="shrink-0 text-text-tertiary">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={handleQueryChange}
            placeholder="Search tasks, lists, and labels…"
            className="flex-1 text-[16px] py-3.5 outline-none bg-transparent text-text-primary placeholder:text-text-tertiary"
            aria-label="Search tasks"
          />
          <kbd className="hidden md:inline-block font-mono text-[11px] text-text-quaternary bg-bg-secondary border border-separator rounded px-1.5 py-0.5">
            ⌘K
          </kbd>
        </div>

        {/* Results */}
        {query.trim() && (
          <div className="flex-1 overflow-y-auto">
            {results.length === 0 ? (
              <div className="px-4 py-8 text-center text-text-tertiary text-sm">
                No tasks found for "{query.trim()}"
              </div>
            ) : (
              <div>
                <div className="px-4 pt-3 pb-1 text-[11px] font-semibold text-text-tertiary uppercase tracking-wider">
                  Tasks · {results.length}
                </div>
                {results.map((task, i) => (
                  <SearchResult
                    key={task.id}
                    task={task}
                    isActive={i === activeIndex}
                    onSelect={() => { onSelectTask(task.id); onClose() }}
                  />
                ))}
              </div>
            )}
          </div>
        )}

        {!query.trim() && (
          <div className="px-4 py-8 text-center text-text-tertiary text-[13px]">
            Type to search across titles, descriptions, and labels
          </div>
        )}

        {/* Keyboard hints footer */}
        <div className="hidden md:flex items-center gap-4 px-4 py-2 border-t border-separator text-[11px] text-text-tertiary shrink-0">
          <span className="flex items-center gap-1.5">
            <kbd className="font-mono bg-bg-secondary border border-separator rounded px-1 py-px">↑↓</kbd>
            Navigate
          </span>
          <span className="flex items-center gap-1.5">
            <kbd className="font-mono bg-bg-secondary border border-separator rounded px-1 py-px">Enter</kbd>
            Open
          </span>
          <span className="flex items-center gap-1.5">
            <kbd className="font-mono bg-bg-secondary border border-separator rounded px-1 py-px">Esc</kbd>
            Close
          </span>
        </div>
      </div>
    </div>
  )
}

function SearchResult({ task, isActive, onSelect }: { task: Task; isActive: boolean; onSelect: () => void }) {
  const ref = useRef<HTMLButtonElement>(null)
  const due = task.due_date ? formatDueDate(task.due_date, task.due_time) : null
  const priorityColor = PRIORITY_COLORS[task.priority]
  const labels = task.labels || []

  // Scroll active result into view
  useEffect(() => {
    if (isActive) ref.current?.scrollIntoView({ block: 'nearest' })
  }, [isActive])

  return (
    <button
      ref={ref}
      type="button"
      onClick={onSelect}
      className={`w-full text-left px-4 py-2.5 min-h-[44px] flex items-start gap-3 transition-colors relative ${
        isActive ? 'bg-accent-light' : 'hover:bg-bg-secondary'
      }`}
    >
      {priorityColor && (
        <span className="absolute left-0 top-2.5 bottom-2.5 w-[3px] rounded-r-full" style={{ backgroundColor: priorityColor }} aria-hidden="true" />
      )}

      <div className={`w-[18px] h-[18px] mt-0.5 rounded-full border-2 shrink-0 flex items-center justify-center ${
        task.is_completed ? 'bg-accent border-accent' : 'border-text-quaternary'
      }`}>
        {task.is_completed && (
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
            <path d="m5 12 5 5L20 7" />
          </svg>
        )}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <InlineMarkdown
            text={task.title}
            className={`flex-1 text-[14px] leading-snug truncate ${task.is_completed ? 'line-through text-text-tertiary' : 'text-text-primary'}`}
          />
          <div className="flex items-center gap-1.5 shrink-0">
            <PriorityFlag priority={task.priority} size={12} />
            {due && (
              <span className={`text-[11px] whitespace-nowrap ${due.overdue ? 'text-danger font-medium' : 'text-text-tertiary'}`}>
                {due.text}
              </span>
            )}
          </div>
        </div>

        {task.description && (
          <p className="text-[12px] text-text-tertiary truncate mt-0.5">{task.description}</p>
        )}

        {labels.length > 0 && (
          <div className="flex items-center gap-1.5 mt-1 flex-wrap">
            {labels.map((label) => (
              <span
                key={label.id}
                className="text-[10px] px-1.5 py-0.5 rounded-full font-medium"
                style={{
                  backgroundColor: (label.colour || COLORS.gray) + '1F',
                  color: label.colour || COLORS.gray,
                }}
              >
                {label.name}
              </span>
            ))}
          </div>
        )}
      </div>
    </button>
  )
}
