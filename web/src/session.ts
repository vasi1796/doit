import { db } from './db/database'

const FLUSH_TIMEOUT_MS = 3000
const WIPE_TIMEOUT_MS = 3000

function bounded(work: Promise<unknown>, ms: number): Promise<unknown> {
  return Promise.race([work, new Promise((resolve) => setTimeout(resolve, ms))])
}

/**
 * Sign out and wipe this device's data. Flush is best-effort with a bounded
 * wait, and the wipe must happen even if flush or the server call fails —
 * leaving another user's data (and their queued sync ops) behind is worse
 * than losing a few offline edits. Deleting the whole database also removes
 * the sync cursor, so the next login re-seeds via initial sync.
 */
export async function signOut(): Promise<void> {
  try {
    // drain (not sync): a plain sync() coalesces into an in-flight run and
    // resolves without flushing anything — exactly when unflushed ops exist
    const flush = window.__syncEngine?.drain()
    if (flush) {
      await bounded(flush, FLUSH_TIMEOUT_MS)
    }
  } catch {
    // best-effort only
  }
  // halt (not stop): forbids engine DB writes so an in-flight response
  // cannot recreate the database after the wipe below
  window.__syncEngine?.halt()
  try {
    // Server failure still expires the session locally: data is wiped and
    // the stale cookie can only produce a fresh 401 → login redirect.
    await fetch('/auth/logout', { method: 'POST', credentials: 'include' })
  } catch {
    // wipe locally even if the server is unreachable
  }
  try {
    // A second tab holding the database open makes deleteDatabase fire
    // "blocked" and wait; the redirect must not hang behind it. The delete
    // still lands once the other connection closes.
    await bounded(db.delete(), WIPE_TIMEOUT_MS)
  } catch {
    // an unreachable wipe must not strand the user on an authenticated page
  }
  window.location.href = '/login'
}
