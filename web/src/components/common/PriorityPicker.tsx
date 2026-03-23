import type { Priority } from '../../api/types'
import { COLORS, PRIORITY_COLORS } from '../../constants'

const PRIORITIES: { value: Priority; label: string; color: string }[] = [
  { value: 0, label: 'None', color: COLORS.gray },
  { value: 1, label: 'Low', color: PRIORITY_COLORS[1]! },
  { value: 2, label: 'Medium', color: PRIORITY_COLORS[2]! },
  { value: 3, label: 'High', color: PRIORITY_COLORS[3]! },
]

interface PriorityPickerProps {
  value: Priority
  onChange: (priority: Priority) => void
  compact?: boolean
}

export function PriorityPicker({ value, onChange, compact = false }: PriorityPickerProps) {
  if (compact) {
    return (
      <div className="flex items-center gap-1 flex-wrap">
        {PRIORITIES.map((p) => (
          <button
            key={p.value}
            type="button"
            onClick={() => onChange(p.value)}
            aria-label={`Priority: ${p.label}`}
            className={`min-h-[36px] flex items-center justify-center rounded-lg transition-all text-[13px] font-medium gap-1 px-2 ${
              value === p.value
                ? 'bg-gray-100 ring-1 ring-gray-200'
                : 'hover:bg-gray-50 opacity-40 hover:opacity-70'
            }`}
          >
            {p.value === 0 ? (
              <span className="text-text-secondary">—</span>
            ) : (
              <svg width="14" height="14" viewBox="0 0 24 24" fill={p.color} stroke={p.color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
                <line x1="4" y1="22" x2="4" y2="15" />
              </svg>
            )}
            <span className="hidden sm:inline" style={{ color: p.color }}>{p.label}</span>
          </button>
        ))}
      </div>
    )
  }

  return (
    <div className="flex items-center gap-1 flex-wrap">
      {PRIORITIES.map((p) => (
        <button
          key={p.value}
          type="button"
          onClick={() => onChange(p.value)}
          aria-label={`Priority: ${p.label}`}
          className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-[13px] font-medium transition-all min-h-[36px] ${
            value === p.value
              ? 'bg-gray-100 ring-1 ring-gray-200'
              : 'opacity-40 hover:opacity-80 hover:bg-gray-50'
          }`}
        >
          {p.value === 0 ? (
            <span className="text-text-secondary">—</span>
          ) : (
            <svg width="14" height="14" viewBox="0 0 24 24" fill={p.color} stroke={p.color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
              <line x1="4" y1="22" x2="4" y2="15" />
            </svg>
          )}
          <span className="hidden sm:inline" style={{ color: p.value === 0 ? COLORS.gray : p.color }}>{p.label}</span>
        </button>
      ))}
    </div>
  )
}

export function priorityLabel(value: Priority): string {
  return PRIORITIES[value]?.label || 'None'
}
