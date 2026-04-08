import { useState } from 'react'
import { usePopover } from '../../hooks/usePopover'
import { formatDisplayTime } from '../../utils/date'

const HOURS = Array.from({ length: 12 }, (_, i) => i + 1)
const MINUTES = [0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55]

interface TimePickerProps {
  value: string
  onChange: (time: string) => void
  onClear?: () => void
}

export function TimePicker({ value, onChange, onClear }: TimePickerProps) {
  const { open, pos, triggerRef, toggle, close } = usePopover({ contentWidth: 220 })

  const parsed = value ? value.split(':').map(Number) : [9, 0]
  const [selHour, setSelHour] = useState(parsed[0] > 12 ? parsed[0] - 12 : parsed[0] || 12)
  const [selMin, setSelMin] = useState(parsed[1] || 0)
  const [selAmPm, setSelAmPm] = useState(parsed[0] >= 12 ? 'PM' : 'AM')
  const [prevValue, setPrevValue] = useState(value)

  if (value !== prevValue) {
    setPrevValue(value)
    const p = value ? value.split(':').map(Number) : [9, 0]
    setSelHour(p[0] > 12 ? p[0] - 12 : p[0] || 12)
    setSelMin(p[1] || 0)
    setSelAmPm(p[0] >= 12 ? 'PM' : 'AM')
  }

  const applyTime = (h: number, m: number, ampm: string) => {
    let hour24 = h
    if (ampm === 'PM' && h !== 12) hour24 = h + 12
    if (ampm === 'AM' && h === 12) hour24 = 0
    const timeStr = `${hour24.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`
    onChange(timeStr)
  }

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={toggle}
        className={`flex items-center gap-2 min-h-[40px] px-3 rounded-[10px] hover:bg-bg-secondary transition-colors text-sm ${value ? 'text-accent' : 'text-text-secondary'}`}
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
          <circle cx="12" cy="12" r="10" />
          <polyline points="12 6 12 12 16 14" />
        </svg>
        <span className={value ? 'text-text-primary' : 'text-text-secondary'}>
          {value ? formatDisplayTime(value) : 'Time'}
        </span>
        {value && onClear && (
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onClear() }}
            className="text-text-secondary hover:text-danger"
            aria-label="Clear time"
          >
            ×
          </button>
        )}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-[60]" onClick={() => close()} aria-hidden="true" />
          <div
            className="fixed bg-bg-elevated rounded-[14px] shadow-popover border border-separator p-3 z-[61] w-[210px]"
            style={{ top: pos.top, left: pos.left }}
          >
            <div className="flex gap-2 mb-3">
              <div className="flex-1">
                <p className="text-[10px] text-text-tertiary font-semibold uppercase tracking-wider mb-1">Hour</p>
                <div className="grid grid-cols-4 gap-1">
                  {HOURS.map(h => (
                    <button
                      key={h}
                      type="button"
                      onClick={() => { setSelHour(h); applyTime(h, selMin, selAmPm) }}
                      className={`py-1.5 rounded-[8px] text-[13px] font-medium transition-colors ${
                        selHour === h ? 'bg-accent text-white' : 'hover:bg-bg-secondary text-text-primary'
                      }`}
                    >
                      {h}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <div className="mb-3">
              <p className="text-[10px] text-text-tertiary font-semibold uppercase tracking-wider mb-1">Minute</p>
              <div className="grid grid-cols-6 gap-1">
                {MINUTES.map(m => (
                  <button
                    key={m}
                    type="button"
                    onClick={() => { setSelMin(m); applyTime(selHour, m, selAmPm) }}
                    className={`py-1.5 rounded-[8px] text-[13px] font-medium transition-colors ${
                      selMin === m ? 'bg-accent text-white' : 'hover:bg-bg-secondary text-text-primary'
                    }`}
                  >
                    {m.toString().padStart(2, '0')}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex gap-1">
              {['AM', 'PM'].map(p => (
                <button
                  key={p}
                  type="button"
                  onClick={() => { setSelAmPm(p); applyTime(selHour, selMin, p) }}
                  className={`flex-1 py-2 rounded-[8px] text-[13px] font-semibold transition-colors ${
                    selAmPm === p ? 'bg-accent text-white' : 'hover:bg-bg-secondary text-text-primary'
                  }`}
                >
                  {p}
                </button>
              ))}
            </div>

            <button
              type="button"
              onClick={() => close()}
              className="w-full mt-2 py-2 text-[13px] font-semibold text-accent hover:bg-accent-light rounded-[8px] transition-colors"
            >
              Done
            </button>
          </div>
        </>
      )}
    </>
  )
}
