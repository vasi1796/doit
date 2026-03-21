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
