import { useState } from 'react'
import { usePopover } from '../../hooks/usePopover'

const RULES = [
  { value: '', label: 'No repeat' },
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
  { value: 'yearly', label: 'Yearly' },
]

interface RecurrencePickerProps {
  value: string
  onChange: (rule: string) => void
}

export function RecurrencePicker({ value, onChange }: RecurrencePickerProps) {
  const { open, pos, triggerRef, toggle, close } = usePopover({ contentWidth: 160 })
  const current = RULES.find((r) => r.value === value) || RULES[0]

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={toggle}
        className="flex items-center gap-2 min-h-[40px] px-3 rounded-lg hover:bg-gray-50 transition-colors text-sm"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke={value ? '#007aff' : '#86868b'} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M17 1l4 4-4 4" />
          <path d="M3 11V9a4 4 0 0 1 4-4h14" />
          <path d="M7 23l-4-4 4-4" />
          <path d="M21 13v2a4 4 0 0 1-4 4H3" />
        </svg>
        <span className={value ? 'text-text-primary' : 'text-text-secondary'}>{current.label}</span>
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-[60]" onClick={() => close()} />
          <div
            className="fixed bg-white rounded-xl shadow-xl border border-gray-200 py-1 z-[61] min-w-[160px]"
            style={{ top: pos.top, left: pos.left }}
          >
            {RULES.map((r) => (
              <button
                key={r.value}
                type="button"
                onClick={() => { onChange(r.value); close() }}
                className={`w-full text-left px-4 py-2.5 text-[15px] hover:bg-gray-50 transition-colors ${
                  value === r.value ? 'text-accent font-medium' : 'text-text-primary'
                }`}
              >
                {r.label}
              </button>
            ))}
          </div>
        </>
      )}
    </>
  )
}

export function recurrenceLabel(rule: string): string {
  return RULES.find((r) => r.value === rule)?.label || ''
}
