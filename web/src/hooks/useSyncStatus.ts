import { useLiveQuery } from 'dexie-react-hooks'
import { useEffect, useState } from 'react'
import { db } from '../db/database'

export type SyncState = 'synced' | 'pending' | 'syncing' | 'offline'

export function useSyncStatus() {
  const pendingCount = useLiveQuery(() => db.syncQueue.count()) ?? 0
  const [online, setOnline] = useState(navigator.onLine)
  const [syncing, setSyncing] = useState(false)

  useEffect(() => {
    const goOnline = () => setOnline(true)
    const goOffline = () => setOnline(false)
    window.addEventListener('online', goOnline)
    window.addEventListener('offline', goOffline)
    return () => {
      window.removeEventListener('online', goOnline)
      window.removeEventListener('offline', goOffline)
    }
  }, [])

  // Listen for custom sync events dispatched by the sync engine
  useEffect(() => {
    const onSyncStart = () => setSyncing(true)
    const onSyncEnd = () => setSyncing(false)
    window.addEventListener('sync:start', onSyncStart)
    window.addEventListener('sync:end', onSyncEnd)
    return () => {
      window.removeEventListener('sync:start', onSyncStart)
      window.removeEventListener('sync:end', onSyncEnd)
    }
  }, [])

  let state: SyncState = 'synced'
  if (!online) state = 'offline'
  else if (syncing) state = 'syncing'
  else if (pendingCount > 0) state = 'pending'

  return { state, pendingCount }
}
