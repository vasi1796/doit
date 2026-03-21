import { useSyncStatus } from '../../hooks/useSyncStatus'

/**
 * Subtle sync status indicator. Shows nothing when fully synced (clean state).
 * Appears only when there's something the user should know about.
 */
export function SyncStatus() {
  const { state, pendingCount } = useSyncStatus()

  if (state === 'synced') return null

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
