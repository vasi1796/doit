import { authAwareFetch } from '../api/http'
import { db } from './database'

const OWNER_KEY = 'db_owner'

/**
 * Bind the local database to the signed-in account. A different account
 * signing in wipes the previous user's rows and queued sync ops before the
 * sync engine starts, so nothing leaks across users on a shared device.
 * An unreachable server resolves without checking: a user switch requires an
 * online login, so an offline boot is always the same user.
 */
export async function ensureDbOwner(): Promise<void> {
  let res: Response
  try {
    res = await authAwareFetch('/auth/me')
  } catch (err) {
    if (err instanceof Error && err.message === 'Unauthorized') throw err
    return
  }
  if (!res.ok) return

  const body = (await res.json().catch(() => null)) as { user_id?: unknown } | null
  const userId = body && typeof body.user_id === 'string' ? body.user_id : ''
  if (!userId) return

  const owner = await db.userPreferences.get(OWNER_KEY)
  if (owner && owner.value !== userId) {
    await db.delete()
    await db.open()
  }
  await db.userPreferences.put({ key: OWNER_KEY, value: userId })
}
