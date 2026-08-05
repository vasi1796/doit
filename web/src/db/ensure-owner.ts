import { authAwareFetch } from '../api/http'
import { db } from './database'
import { unsubscribeFromPushLocally } from '../push'

const OWNER_KEY = 'db_owner'
// Matches session.ts signOut's wipe bound
const WIPE_TIMEOUT_MS = 3000
const UNSUBSCRIBE_TIMEOUT_MS = 3000

function bounded(work: Promise<unknown>, ms: number): Promise<unknown> {
  return Promise.race([work, new Promise((resolve) => setTimeout(resolve, ms))])
}

/** Ownership unconfirmed — callers must not start the sync engine. */
export class OwnerCheckError extends Error {}

/**
 * Bind the local database to the signed-in account, wiping it on an account
 * switch. Offline (fetch rejection) resolves: a switch requires an online
 * login. A served non-ok/malformed response fails closed — the switch may
 * have happened and only the check failed.
 */
export async function ensureDbOwner(): Promise<void> {
  let res: Response
  try {
    res = await authAwareFetch('/auth/me')
  } catch (err) {
    if (err instanceof Error && err.message === 'Unauthorized') throw err
    return
  }
  if (!res.ok) throw new OwnerCheckError(`auth/me returned ${res.status}`)

  const body = (await res.json().catch(() => null)) as { user_id?: unknown } | null
  const userId = body && typeof body.user_id === 'string' ? body.user_id : ''
  if (!userId) throw new OwnerCheckError('auth/me returned no user_id')

  const owner = await db.userPreferences.get(OWNER_KEY)
  if (owner && owner.value !== userId) {
    try {
      // Browser-side only: the previous user's reminders must stop reaching
      // this device, but the wipe proceeds even if this fails.
      await bounded(unsubscribeFromPushLocally(), UNSUBSCRIBE_TIMEOUT_MS)
    } catch {
      // best-effort only
    }
    // A suspended tab holding a connection blocks deleteDatabase forever;
    // fail closed rather than boot the sync engine over the old data.
    const outcome = await Promise.race([
      db.delete().then(() => 'wiped' as const),
      new Promise<'blocked'>((resolve) => setTimeout(() => resolve('blocked'), WIPE_TIMEOUT_MS)),
    ])
    if (outcome === 'blocked') {
      throw new OwnerCheckError('database wipe blocked by another connection')
    }
    await db.open()
  }
  await db.userPreferences.put({ key: OWNER_KEY, value: userId })
}
