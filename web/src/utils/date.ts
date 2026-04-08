/** Convert a Date to YYYY-MM-DD string */
export function toDateStr(d: Date): string {
  return `${d.getFullYear()}-${(d.getMonth() + 1).toString().padStart(2, '0')}-${d.getDate().toString().padStart(2, '0')}`
}

/** Calculate day difference from today (negative = past) */
function dayDiff(dateStr: string): number {
  const date = new Date(dateStr + 'T00:00:00')
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Math.floor((date.getTime() - today.getTime()) / (1000 * 60 * 60 * 24))
}

/** Format a date string for display in pickers (Today, Tomorrow, weekday, or short date) */
export function formatDisplayDate(dateStr: string): string {
  const diff = dayDiff(dateStr)
  const date = new Date(dateStr + 'T00:00:00')

  if (diff === 0) return 'Today'
  if (diff === 1) return 'Tomorrow'
  if (diff === -1) return 'Yesterday'
  if (diff > 1 && diff < 7) return date.toLocaleDateString('en-US', { weekday: 'long' })
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

/** Format a due date with optional time for task list display, including overdue status */
export function formatDueDate(dateStr: string, timeStr?: string): { text: string; overdue: boolean } {
  const diff = dayDiff(dateStr)
  const date = new Date(dateStr + 'T00:00:00')

  let dateText: string
  if (diff < 0) dateText = 'Overdue'
  else if (diff === 0) dateText = 'Today'
  else if (diff === 1) dateText = 'Tomorrow'
  else if (diff < 7) dateText = date.toLocaleDateString('en-US', { weekday: 'short' })
  else dateText = date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })

  if (timeStr) {
    const [h, m] = timeStr.split(':').map(Number)
    const ampm = h >= 12 ? 'PM' : 'AM'
    const hour = h % 12 || 12
    dateText += ` ${hour}:${m.toString().padStart(2, '0')} ${ampm}`
  }

  return { text: dateText, overdue: diff < 0 }
}

/** Format a 24h time string (HH:MM) for display (12h with AM/PM) */
export function formatDisplayTime(timeStr: string): string {
  const [h, m] = timeStr.split(':').map(Number)
  const ampm = h >= 12 ? 'PM' : 'AM'
  const hour = h % 12 || 12
  return `${hour}:${m.toString().padStart(2, '0')} ${ampm}`
}

/** Whole days between an ISO timestamp and now, floored at 0. Used by the trash
 *  auto-delete countdown — pass an `updated_at` (or a future `deleted_at`)
 *  and the 30-day TTL window. Returns null when the input is missing. */
export function daysUntilTTL(fromTimestamp: string | undefined, ttlDays: number): number | null {
  if (!fromTimestamp) return null
  const deleted = new Date(fromTimestamp)
  const expires = new Date(deleted)
  expires.setDate(expires.getDate() + ttlDays)
  const days = Math.ceil((expires.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
  return Math.max(0, days)
}

/** Returns YYYY-MM-DD strings for the next N days starting from tomorrow
 *  (excludes today). Used by the Upcoming page. */
export function nextNDays(n: number): string[] {
  const days: string[] = []
  const base = new Date()
  for (let i = 1; i <= n; i++) {
    const d = new Date(base)
    d.setDate(d.getDate() + i)
    days.push(toDateStr(d))
  }
  return days
}

/** Format a YYYY-MM-DD into {primary, secondary, isTomorrow} for day-group
 *  headers on the Upcoming page. Tomorrow gets the accent treatment in the UI. */
export function formatDayGroupHeader(dateStr: string): { primary: string; secondary: string; isTomorrow: boolean } {
  const date = new Date(dateStr + 'T00:00:00')
  const secondary = date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  const isTomorrow = dayDiff(dateStr) === 1
  return {
    primary: isTomorrow ? 'Tomorrow' : date.toLocaleDateString('en-US', { weekday: 'long' }),
    secondary,
    isTomorrow,
  }
}

/** Buckets for grouping completed tasks by when they were completed. */
export type CompletedTimeGroup = 'today' | 'yesterday' | 'week' | 'earlier'

/** Group items by completion timestamp into today/yesterday/this-week/earlier.
 *  The grouping boundaries are computed once per call.
 *  Also returns the count completed in the current calendar month, so the
 *  caller doesn't need a second pass over the list. */
export function groupByCompletion<T extends { completed_at?: string }>(
  items: readonly T[],
): { grouped: Record<CompletedTimeGroup, T[]>; monthCount: number } {
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  const weekAgo = new Date(today)
  weekAgo.setDate(weekAgo.getDate() - 7)
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1)

  const grouped: Record<CompletedTimeGroup, T[]> = { today: [], yesterday: [], week: [], earlier: [] }
  let monthCount = 0

  for (const item of items) {
    if (!item.completed_at) {
      grouped.earlier.push(item)
      continue
    }
    const completed = new Date(item.completed_at)
    if (completed >= monthStart) monthCount++
    if (completed >= today) grouped.today.push(item)
    else if (completed >= yesterday) grouped.yesterday.push(item)
    else if (completed >= weekAgo) grouped.week.push(item)
    else grouped.earlier.push(item)
  }

  return { grouped, monthCount }
}
