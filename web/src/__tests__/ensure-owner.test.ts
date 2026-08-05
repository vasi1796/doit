import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ensureDbOwner, OwnerCheckError } from '../db/ensure-owner'

const { prefs, deleteFn, openFn, fetchFn, unsubscribeFn } = vi.hoisted(() => ({
  prefs: new Map<string, { key: string; value: string }>(),
  deleteFn: vi.fn(),
  openFn: vi.fn(),
  fetchFn: vi.fn(),
  unsubscribeFn: vi.fn(),
}))

vi.mock('../db/database', () => ({
  db: {
    userPreferences: {
      get: async (key: string) => prefs.get(key),
      put: async (row: { key: string; value: string }) => {
        prefs.set(row.key, row)
      },
    },
    delete: deleteFn,
    open: openFn,
  },
}))

vi.mock('../api/http', () => ({ authAwareFetch: fetchFn }))

vi.mock('../push', () => ({ unsubscribeFromPushLocally: unsubscribeFn }))

function okUser(id: string) {
  return { ok: true, json: async () => ({ user_id: id }) }
}

describe('ensureDbOwner', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    prefs.clear()
    deleteFn.mockResolvedValue(undefined)
    openFn.mockResolvedValue(undefined)
    unsubscribeFn.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('first boot records the owner without wiping', async () => {
    fetchFn.mockResolvedValue(okUser('user-a'))

    await ensureDbOwner()

    expect(deleteFn).not.toHaveBeenCalled()
    expect(prefs.get('db_owner')?.value).toBe('user-a')
  })

  it('the same account keeps local data and queued ops', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockResolvedValue(okUser('user-a'))

    await ensureDbOwner()

    expect(deleteFn).not.toHaveBeenCalled()
  })

  it('a different account wipes the database before recording the new owner', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockResolvedValue(okUser('user-b'))

    await ensureDbOwner()

    expect(unsubscribeFn).toHaveBeenCalledOnce()
    expect(deleteFn).toHaveBeenCalledOnce()
    expect(openFn).toHaveBeenCalledOnce()
    expect(prefs.get('db_owner')?.value).toBe('user-b')
  })

  it('the wipe proceeds even when the push unsubscribe fails', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockResolvedValue(okUser('user-b'))
    unsubscribeFn.mockRejectedValue(new Error('no service worker'))

    await ensureDbOwner()

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(prefs.get('db_owner')?.value).toBe('user-b')
  })

  it('a blocked wipe fails closed instead of booting over the previous data', async () => {
    vi.useFakeTimers()
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockResolvedValue(okUser('user-b'))
    deleteFn.mockReturnValue(new Promise(() => {})) // suspended tab holds the DB

    const run = ensureDbOwner()
    const assertion = expect(run).rejects.toBeInstanceOf(OwnerCheckError)
    await vi.advanceTimersByTimeAsync(3_000)
    await assertion

    expect(openFn).not.toHaveBeenCalled()
    expect(prefs.get('db_owner')?.value).toBe('user-a')
  })

  it('an unreachable server proceeds without wiping — offline boots are always the same user', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockRejectedValue(new TypeError('fetch failed'))

    await expect(ensureDbOwner()).resolves.toBeUndefined()

    expect(deleteFn).not.toHaveBeenCalled()
    expect(prefs.get('db_owner')?.value).toBe('user-a')
  })

  it('a malformed response fails closed without wiping or rebinding', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockResolvedValue({ ok: true, json: async () => ({ nope: 1 }) })

    await expect(ensureDbOwner()).rejects.toBeInstanceOf(OwnerCheckError)

    expect(deleteFn).not.toHaveBeenCalled()
    expect(prefs.get('db_owner')?.value).toBe('user-a')
  })

  it('a served error response fails closed — the switch may have happened and only the check failed', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockResolvedValue({ ok: false, status: 500, json: async () => ({}) })

    await expect(ensureDbOwner()).rejects.toBeInstanceOf(OwnerCheckError)

    expect(deleteFn).not.toHaveBeenCalled()
    expect(prefs.get('db_owner')?.value).toBe('user-a')
  })
})
