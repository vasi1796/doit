import { useState } from 'react'
import { usePopover } from '../../hooks/usePopover'
import { toDateStr, formatDisplayDate } from '../../utils/date'

const DAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']

function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate()
}

function getFirstDayOfWeek(year: number, month: number): number {
  return new Date(year, month, 1).getDay()
}

interface DatePickerProps {
  value: string
  onChange: (date: string) => void
  onClear?: () => void
}

export function DatePicker({ value, onChange, onClear }: DatePickerProps) {
  const { open, pos, triggerRef, toggle, close } = usePopover({ contentWidth: 280, contentHeight: 340 })

  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const todayStr = toDateStr(today)

  const initial = value ? new Date(value + 'T00:00:00') : today
  const [viewYear, setViewYear] = useState(initial.getFullYear())
  const [viewMonth, setViewMonth] = useState(initial.getMonth())
  const [prevValue, setPrevValue] = useState(value)

  if (value !== prevValue) {
    setPrevValue(value)
    const d = value ? new Date(value + 'T00:00:00') : new Date()
    setViewYear(d.getFullYear())
    setViewMonth(d.getMonth())
  }

  const prevMonth = () => {
    if (viewMonth === 0) { setViewMonth(11); setViewYear(viewYear - 1) }
    else setViewMonth(viewMonth - 1)
  }

  const nextMonth = () => {
    if (viewMonth === 11) { setViewMonth(0); setViewYear(viewYear + 1) }
    else setViewMonth(viewMonth + 1)
  }

  const selectDate = (day: number) => {
    const d = new Date(viewYear, viewMonth, day)
    onChange(toDateStr(d))
    close()
  }

  const daysInMonth = getDaysInMonth(viewYear, viewMonth)
  const firstDay = getFirstDayOfWeek(viewYear, viewMonth)

  // Quick shortcuts
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)
  const nextWeek = new Date(today)
  nextWeek.setDate(nextWeek.getDate() + 7)

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={toggle}
        className="flex items-center gap-2 min-h-[40px] px-3 rounded-lg hover:bg-gray-50 transition-colors text-sm"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke={value ? '#007aff' : '#86868b'} strokeWidth="1.5" strokeLinecap="round">
          <rect x="3" y="4" width="18" height="18" rx="2" />
          <line x1="16" y1="2" x2="16" y2="6" />
          <line x1="8" y1="2" x2="8" y2="6" />
          <line x1="3" y1="10" x2="21" y2="10" />
        </svg>
        <span className={value ? 'text-text-primary' : 'text-text-secondary'}>
          {value ? formatDisplayDate(value) : 'Date'}
        </span>
        {value && onClear && (
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onClear() }}
            className="text-text-secondary hover:text-danger"
            aria-label="Clear date"
          >
            ×
          </button>
        )}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-[60]" onClick={close} aria-hidden="true" />
          <div
            className="fixed bg-white rounded-xl shadow-xl border border-gray-200 p-3 z-[61] w-[270px]"
            style={{ top: pos.top, left: pos.left }}
          >
            {/* Quick shortcuts */}
            <div className="flex gap-1.5 mb-3">
              <button
                type="button"
                onClick={() => { onChange(todayStr); close() }}
                className="flex-1 py-1.5 text-[12px] font-medium rounded-lg bg-accent/8 text-accent hover:bg-accent/15"
              >
                Today
              </button>
              <button
                type="button"
                onClick={() => { onChange(toDateStr(tomorrow)); close() }}
                className="flex-1 py-1.5 text-[12px] font-medium rounded-lg bg-gray-100 text-text-primary hover:bg-gray-200"
              >
                Tomorrow
              </button>
              <button
                type="button"
                onClick={() => { onChange(toDateStr(nextWeek)); close() }}
                className="flex-1 py-1.5 text-[12px] font-medium rounded-lg bg-gray-100 text-text-primary hover:bg-gray-200"
              >
                Next week
              </button>
            </div>

            {/* Month navigation */}
            <div className="flex items-center justify-between mb-2">
              <button type="button" onClick={prevMonth} className="p-1 hover:bg-gray-100 rounded-lg">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#1d1d1f" strokeWidth="2" strokeLinecap="round">
                  <path d="m15 18-6-6 6-6" />
                </svg>
              </button>
              <span className="text-[13px] font-semibold text-text-primary">
                {MONTHS[viewMonth]} {viewYear}
              </span>
              <button type="button" onClick={nextMonth} className="p-1 hover:bg-gray-100 rounded-lg">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#1d1d1f" strokeWidth="2" strokeLinecap="round">
                  <path d="m9 18 6-6-6-6" />
                </svg>
              </button>
            </div>

            {/* Day headers */}
            <div className="grid grid-cols-7 mb-1">
              {DAYS.map(d => (
                <span key={d} className="text-center text-[10px] font-medium text-text-secondary py-1">{d}</span>
              ))}
            </div>

            {/* Calendar grid */}
            <div className="grid grid-cols-7">
              {/* Empty cells for days before the 1st */}
              {Array.from({ length: firstDay }, (_, i) => (
                <span key={`empty-${i}`} />
              ))}
              {/* Day cells */}
              {Array.from({ length: daysInMonth }, (_, i) => {
                const day = i + 1
                const dateStr = toDateStr(new Date(viewYear, viewMonth, day))
                const isSelected = dateStr === value
                const isToday = dateStr === todayStr
                const isPast = new Date(viewYear, viewMonth, day) < today

                return (
                  <button
                    key={day}
                    type="button"
                    onClick={() => selectDate(day)}
                    className={`py-1.5 text-[13px] rounded-lg transition-colors ${
                      isSelected
                        ? 'bg-accent text-white font-semibold'
                        : isToday
                          ? 'bg-accent/10 text-accent font-semibold'
                          : isPast
                            ? 'text-text-tertiary hover:bg-gray-100'
                            : 'text-text-primary hover:bg-gray-100'
                    }`}
                  >
                    {day}
                  </button>
                )
              })}
            </div>
          </div>
        </>
      )}
    </>
  )
}
