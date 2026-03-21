const PRIORITY_FLAGS: Record<number, { color: string; label: string }> = {
  1: { color: '#4cd964', label: 'Low' },
  2: { color: '#ff9500', label: 'Medium' },
  3: { color: '#ff3b30', label: 'High' },
}

export function PriorityFlag({ priority, size = 14 }: { priority: number; size?: number }) {
  const flag = PRIORITY_FLAGS[priority]
  if (!flag) return null

  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill={flag.color} stroke={flag.color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-label={`${flag.label} priority`}>
      <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
      <line x1="4" y1="22" x2="4" y2="15" />
    </svg>
  )
}

// Keep backward compat
export function PriorityDot({ priority }: { priority: number }) {
  return <PriorityFlag priority={priority} size={12} />
}
