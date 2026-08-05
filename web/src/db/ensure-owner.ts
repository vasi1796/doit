import { authAwareFetch } from '../api/http'
import { db } from './database'

const OWNER_KEY = 'db_owner'

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
    await db.delete()
    await db.open()
  }
  await db.userPreferences.put({ key: OWNER_KEY, value: userId })
}
