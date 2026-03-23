import { useState, useMemo } from 'react'
import { useTasks } from '../hooks/useTasks'
import { useLayoutContext } from '../components/layout/AppLayout'
import { TaskDetail } from '../components/tasks/TaskDetail'
import { InlineMarkdown } from '../components/common/InlineMarkdown'
import { PRIORITY_COLORS } from '../constants'
import type { Task } from '../api/types'

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

function toDateStr(d: Date): string {
  return `${d.getFullYear()}-${(d.getMonth() + 1).toString().padStart(2, '0')}-${d.getDate().toString().padStart(2, '0')}`
}

function getCalendarDays(year: number, month: number): { date: string; inMonth: boolean }[] {
  const first = new Date(year, month, 1)
  const last = new Date(year, month + 1, 0)
  const days: { date: string; inMonth: boolean }[] = []

  // Fill leading days from previous month
  for (let i = 0; i < first.getDay(); i++) {
    const d = new Date(year, month, -first.getDay() + i + 1)
    days.push({ date: toDateStr(d), inMonth: false })
  }

  // Days in current month
  for (let d = 1; d <= last.getDate(); d++) {
    days.push({ date: toDateStr(new Date(year, month, d)), inMonth: true })
  }

  // Fill trailing days to complete the grid (always 6 rows = 42 cells)
  while (days.length < 42) {
    const d = new Date(year, month + 1, days.length - (first.getDay() + last.getDate()) + 1)
    days.push({ date: toDateStr(d), inMonth: false })
  }

  return days
}

interface CalendarEntry {
  task: Task
  isPhantom: boolean
}

function buildEntriesByDate(tasks: Task[], rangeStart: string, rangeEnd: string): Record<string, CalendarEntry[]> {
  const map: Record<string, CalendarEntry[]> = {}

  const addEntry = (dateStr: string, entry: CalendarEntry) => {
    if (!map[dateStr]) map[dateStr] = []
    map[dateStr].push(entry)
  }

  for (const task of tasks) {
    // Add the actual task on its due date
    if (task.due_date) {
      addEntry(task.due_date, { task, isPhantom: false })
    }

    // Expand recurring tasks into phantom future entries
    if (!task.recurrence_rule || !task.due_date) continue

    const limit = new Date(rangeEnd + 'T00:00:00')
    const cursor = new Date(task.due_date + 'T00:00:00')
    advanceCursor(cursor, task.recurrence_rule)

    let safety = 0
    while (cursor <= limit && safety < 60) {
      const dateStr = toDateStr(cursor)
      if (dateStr >= rangeStart && dateStr <= rangeEnd) {
        addEntry(dateStr, { task, isPhantom: true })
      }
      advanceCursor(cursor, task.recurrence_rule)
      safety++
    }
  }

  return map
}

function advanceCursor(cursor: Date, rule: string) {
  switch (rule) {
    case 'daily': cursor.setDate(cursor.getDate() + 1); break
    case 'weekly': cursor.setDate(cursor.getDate() + 7); break
    case 'monthly': cursor.setMonth(cursor.getMonth() + 1); break
    case 'yearly': cursor.setFullYear(cursor.getFullYear() + 1); break
  }
}

function CalendarTask({ entry, onSelect }: { entry: CalendarEntry; onSelect: (id: string) => void }) {
  const dotColor = PRIORITY_COLORS[entry.task.priority]

  return (
    <button
      type="button"
      onClick={(e) => { e.stopPropagation(); onSelect(entry.task.id) }}
      aria-label={entry.task.title}
      className={`w-full text-left text-[11px] leading-tight px-1 py-1 rounded hover:bg-black/5 truncate flex items-center gap-1 min-h-[44px] ${
        entry.isPhantom ? 'opacity-50' : ''
      }`}
    >
      {dotColor && (
        <span className="w-1.5 h-1.5 rounded-full shrink-0" style={{ backgroundColor: dotColor }} />
      )}
      {entry.isPhantom && (
        <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" className="shrink-0 text-text-secondary">
          <path d="M17 1l4 4-4 4" /><path d="M3 11V9a4 4 0 0 1 4-4h14" />
          <path d="M7 23l-4-4 4-4" /><path d="M21 13v2a4 4 0 0 1-4 4H3" />
        </svg>
      )}
      <InlineMarkdown text={entry.task.title} className="truncate" />
    </button>
  )
}

export function CalendarPage() {
  const { tasks, loading } = useTasks({ is_completed: 'false' })
  const { lists } = useLayoutContext()
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth())

  const todayStr = toDateStr(now)

  const days = useMemo(() => getCalendarDays(year, month), [year, month])

  const entriesByDate = useMemo(() => {
    const rangeStart = days[0].date
    const rangeEnd = days[days.length - 1].date
    return buildEntriesByDate(tasks, rangeStart, rangeEnd)
  }, [tasks, days])

  const goToPrev = () => {
    if (month === 0) { setYear(year - 1); setMonth(11) }
    else setMonth(month - 1)
  }
  const goToNext = () => {
    if (month === 11) { setYear(year + 1); setMonth(0) }
    else setMonth(month + 1)
  }
  const goToToday = () => {
    setYear(now.getFullYear())
    setMonth(now.getMonth())
  }

  const monthLabel = new Date(year, month).toLocaleDateString('en-US', { month: 'long', year: 'numeric' })

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 pt-6 pb-2 flex items-center gap-3">
        <h1 className="text-2xl font-semibold text-text-primary">Calendar</h1>
        <div className="ml-auto flex items-center gap-1">
          <button
            type="button"
            onClick={goToToday}
            className="text-xs text-accent font-medium px-3 min-h-[44px] min-w-[44px]"
          >
            Today
          </button>
          <button
            type="button"
            onClick={goToPrev}
            aria-label="Previous month"
            className="w-[44px] h-[44px] flex items-center justify-center text-text-secondary hover:text-text-primary"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M15 18l-6-6 6-6" />
            </svg>
          </button>
          <span className="text-sm font-medium text-text-primary min-w-[140px] text-center">{monthLabel}</span>
          <button
            type="button"
            onClick={goToNext}
            aria-label="Next month"
            className="w-[44px] h-[44px] flex items-center justify-center text-text-secondary hover:text-text-primary"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M9 18l6-6-6-6" />
            </svg>
          </button>
        </div>
      </div>

      {loading ? (
        <div className="px-4 py-8 text-center text-text-secondary text-sm">Loading...</div>
      ) : (
        <div className="flex-1 overflow-y-auto px-4 pb-4">
          {/* Weekday headers */}
          <div className="grid grid-cols-7 mb-1">
            {WEEKDAYS.map((d) => (
              <div key={d} className="text-center text-[11px] font-medium text-text-secondary py-1">{d}</div>
            ))}
          </div>

          {/* Calendar grid */}
          <div className="grid grid-cols-7 border-t border-l border-gray-200">
            {days.map((day) => {
              const dayTasks = entriesByDate[day.date] || []
              const isToday = day.date === todayStr
              const dayNum = parseInt(day.date.split('-')[2], 10)

              return (
                <div
                  key={day.date}
                  className={`border-r border-b border-gray-200 min-h-[80px] md:min-h-[100px] p-1 ${
                    day.inMonth ? 'bg-white' : 'bg-gray-50'
                  }`}
                >
                  <div className="flex justify-end mb-0.5">
                    <span className={`text-[12px] w-6 h-6 flex items-center justify-center rounded-full ${
                      isToday
                        ? 'bg-accent text-white font-semibold'
                        : day.inMonth
                          ? 'text-text-primary'
                          : 'text-text-tertiary'
                    }`}>
                      {dayNum}
                    </span>
                  </div>
                  <div className="space-y-0.5">
                    {dayTasks.slice(0, 3).map((entry, i) => (
                      <CalendarTask key={`${entry.task.id}-${entry.isPhantom ? 'p' : 'r'}-${i}`} entry={entry} onSelect={setSelectedId} />
                    ))}
                    {dayTasks.length > 3 && (
                      <span className="text-[10px] text-text-secondary pl-1">+{dayTasks.length - 3} more</span>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {selectedId && (
        <TaskDetail taskId={selectedId} lists={lists} onClose={() => setSelectedId(null)} />
      )}
    </div>
  )
}
