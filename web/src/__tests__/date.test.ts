import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { toDateStr, formatDisplayDate, formatDueDate, formatDisplayTime } from '../utils/date'

describe('toDateStr', () => {
  const cases: { name: string; date: Date; expected: string }[] = [
    { name: 'formats a normal date', date: new Date(2025, 0, 15), expected: '2025-01-15' },
    { name: 'pads single-digit month', date: new Date(2025, 2, 5), expected: '2025-03-05' },
    { name: 'pads single-digit day', date: new Date(2025, 11, 1), expected: '2025-12-01' },
    { name: 'handles Dec 31', date: new Date(2025, 11, 31), expected: '2025-12-31' },
    { name: 'handles Jan 1', date: new Date(2026, 0, 1), expected: '2026-01-01' },
  ]

  for (const tc of cases) {
    it(tc.name, () => {
      expect(toDateStr(tc.date)).toBe(tc.expected)
    })
  }
})

// Helper: freeze "today" for deterministic date-relative tests
function freezeDate(dateStr: string) {
  const fakeNow = new Date(dateStr + 'T12:00:00').getTime()
  vi.useFakeTimers()
  vi.setSystemTime(fakeNow)
}

describe('formatDisplayDate', () => {
  beforeEach(() => freezeDate('2026-03-23'))
  afterEach(() => vi.useRealTimers())

  const cases: { name: string; input: string; expected: string }[] = [
    { name: 'returns "Today" for today', input: '2026-03-23', expected: 'Today' },
    { name: 'returns "Tomorrow" for tomorrow', input: '2026-03-24', expected: 'Tomorrow' },
    { name: 'returns "Yesterday" for yesterday', input: '2026-03-22', expected: 'Yesterday' },
    { name: 'returns weekday name for 2-6 days ahead', input: '2026-03-26', expected: 'Thursday' },
    { name: 'returns short date for 7+ days ahead', input: '2026-04-15', expected: 'Apr 15' },
    { name: 'returns short date for dates in the past beyond yesterday', input: '2026-03-10', expected: 'Mar 10' },
  ]

  for (const tc of cases) {
    it(tc.name, () => {
      expect(formatDisplayDate(tc.input)).toBe(tc.expected)
    })
  }
})

describe('formatDueDate', () => {
  beforeEach(() => freezeDate('2026-03-23'))
  afterEach(() => vi.useRealTimers())

  const cases: { name: string; dateStr: string; timeStr?: string; expectedText: string; expectedOverdue: boolean }[] = [
    { name: 'today without time', dateStr: '2026-03-23', expectedText: 'Today', expectedOverdue: false },
    { name: 'tomorrow without time', dateStr: '2026-03-24', expectedText: 'Tomorrow', expectedOverdue: false },
    { name: 'overdue date', dateStr: '2026-03-20', expectedText: 'Overdue', expectedOverdue: true },
    { name: 'weekday for 2-6 days out', dateStr: '2026-03-27', expectedText: 'Fri', expectedOverdue: false },
    { name: 'short date for 7+ days', dateStr: '2026-05-01', expectedText: 'May 1', expectedOverdue: false },
    { name: 'today with time', dateStr: '2026-03-23', timeStr: '14:30', expectedText: 'Today 2:30 PM', expectedOverdue: false },
    { name: 'overdue with time', dateStr: '2026-03-20', timeStr: '09:00', expectedText: 'Overdue 9:00 AM', expectedOverdue: true },
  ]

  for (const tc of cases) {
    it(tc.name, () => {
      const result = formatDueDate(tc.dateStr, tc.timeStr)
      expect(result.text).toBe(tc.expectedText)
      expect(result.overdue).toBe(tc.expectedOverdue)
    })
  }
})

describe('formatDisplayTime', () => {
  const cases: { name: string; input: string; expected: string }[] = [
    { name: 'converts morning time', input: '09:05', expected: '9:05 AM' },
    { name: 'converts noon', input: '12:00', expected: '12:00 PM' },
    { name: 'converts afternoon time', input: '14:30', expected: '2:30 PM' },
    { name: 'converts midnight', input: '00:00', expected: '12:00 AM' },
    { name: 'converts 1 AM', input: '01:00', expected: '1:00 AM' },
    { name: 'converts 11 PM', input: '23:59', expected: '11:59 PM' },
  ]

  for (const tc of cases) {
    it(tc.name, () => {
      expect(formatDisplayTime(tc.input)).toBe(tc.expected)
    })
  }
})
