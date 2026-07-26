import { db } from './db/database'

const FLUSH_TIMEOUT_MS = 3000

/**
 * Sign out and wipe this device's data. Flush is best-effort with a bounded
 * wait, and the wipe must happen even if flush or the server call fails —
 * leaving another user's data (and their queued sync ops) behind is worse
 * than losing a few offline edits. Deleting the whole database also removes
 * the sync cursor, so the next login re-seeds via initial sync.
 */
export async function signOut(): Promise<void> {
  try {
    const flush = window.__syncEngine?.sync()
    if (flush) {
      await Promise.race([flush, new Promise((resolve) => setTimeout(resolve, FLUSH_TIMEOUT_MS))])
    }
  } catch {
    // best-effort only
  }
  window.__syncEngine?.stop()
  try {
    // Server failure still expires the session locally: data is wiped and
    // the stale cookie can only produce a fresh 401 → login redirect.
    await fetch('/auth/logout', { method: 'POST', credentials: 'include' })
  } catch {
    // wipe locally even if the server is unreachable
  }
  await db.delete()
  window.location.href = '/login'
}
