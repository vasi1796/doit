import { useEffect, useState } from 'react'
import { useSyncStatus } from '../../hooks/useSyncStatus'

/**
 * Subtle sync status indicator. Shows nothing when fully synced (clean state).
 * Suppressed for the first 3 seconds after mount to avoid flashing during
 * initial sync cycle.
 */
export function SyncStatus() {
  const { state, pendingCount } = useSyncStatus()
  const [ready, setReady] = useState(false)

  useEffect(() => {
    const timer = setTimeout(() => setReady(true), 5000)
    return () => clearTimeout(timer)
  }, [])

  if (!ready || state === 'synced') return null

  return (
    <div className="flex items-center gap-1.5 text-[11px] px-3 py-1.5">
      <span className={`w-1.5 h-1.5 rounded-full ${dotColor(state)}`} />
      <span className="text-text-secondary">{label(state, pendingCount)}</span>
    </div>
  )
}

function dotColor(state: string): string {
  switch (state) {
    case 'syncing': return 'bg-accent animate-pulse'
    case 'pending': return 'bg-warning'
    case 'offline': return 'bg-text-tertiary'
    default: return 'bg-success'
  }
}

function label(state: string, count: number): string {
  switch (state) {
    case 'syncing': return 'Syncing...'
    case 'pending': return `${count} pending`
    case 'offline': return 'Offline'
    default: return ''
  }
}
