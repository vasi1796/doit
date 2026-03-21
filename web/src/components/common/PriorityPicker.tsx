import type { Priority } from '../../api/types'

const PRIORITIES: { value: Priority; label: string; color: string }[] = [
  { value: 0, label: 'None', color: '#86868b' },
  { value: 1, label: 'Low', color: '#4cd964' },
  { value: 2, label: 'Medium', color: '#ff9500' },
  { value: 3, label: 'High', color: '#ff3b30' },
]

interface PriorityPickerProps {
  value: Priority
  onChange: (priority: Priority) => void
  compact?: boolean
}

export function PriorityPicker({ value, onChange, compact = false }: PriorityPickerProps) {
  if (compact) {
    return (
      <div className="flex items-center gap-2">
        {PRIORITIES.map((p) => (
          <button
            key={p.value}
            type="button"
            onClick={() => onChange(p.value)}
            aria-label={`Priority: ${p.label}`}
            className={`min-h-[36px] flex items-center justify-center rounded-lg transition-all text-[13px] font-medium gap-1.5 px-3 ${
              value === p.value
                ? 'bg-gray-100 ring-1 ring-gray-200'
                : 'hover:bg-gray-50 opacity-40 hover:opacity-70'
            }`}
          >
            {p.value === 0 ? (
              <span className="text-[#86868b]">—</span>
            ) : (
              <svg width="14" height="14" viewBox="0 0 24 24" fill={p.color} stroke={p.color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
                <line x1="4" y1="22" x2="4" y2="15" />
              </svg>
            )}
            <span style={{ color: p.color }}>{p.label}</span>
          </button>
        ))}
      </div>
    )
  }

  // Full property row style
  return (
    <div className="flex items-center gap-1">
      {PRIORITIES.map((p) => (
        <button
          key={p.value}
          type="button"
          onClick={() => onChange(p.value)}
          aria-label={`Priority: ${p.label}`}
          className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-all min-h-[40px] ${
            value === p.value
              ? 'bg-gray-100 ring-1 ring-gray-200'
              : 'opacity-40 hover:opacity-80 hover:bg-gray-50'
          }`}
        >
          {p.value === 0 ? (
            <span className="text-[#86868b]">—</span>
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill={p.color} stroke={p.color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
              <line x1="4" y1="22" x2="4" y2="15" />
            </svg>
          )}
          <span style={{ color: p.value === 0 ? '#86868b' : p.color }}>{p.label}</span>
        </button>
      ))}
    </div>
  )
}

export function priorityLabel(value: Priority): string {
  return PRIORITIES[value]?.label || 'None'
}
