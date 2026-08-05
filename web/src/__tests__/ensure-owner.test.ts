import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ensureDbOwner } from '../db/ensure-owner'

const { prefs, deleteFn, openFn, fetchFn } = vi.hoisted(() => ({
  prefs: new Map<string, { key: string; value: string }>(),
  deleteFn: vi.fn(),
  openFn: vi.fn(),
  fetchFn: vi.fn(),
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

function okUser(id: string) {
  return { ok: true, json: async () => ({ user_id: id }) }
}

describe('ensureDbOwner', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    prefs.clear()
    deleteFn.mockResolvedValue(undefined)
    openFn.mockResolvedValue(undefined)
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

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(openFn).toHaveBeenCalledOnce()
    expect(prefs.get('db_owner')?.value).toBe('user-b')
  })

  it('an unreachable server proceeds without wiping — offline boots are always the same user', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockRejectedValue(new TypeError('fetch failed'))

    await expect(ensureDbOwner()).resolves.toBeUndefined()

    expect(deleteFn).not.toHaveBeenCalled()
    expect(prefs.get('db_owner')?.value).toBe('user-a')
  })

  it('a malformed response neither wipes nor rebinds', async () => {
    prefs.set('db_owner', { key: 'db_owner', value: 'user-a' })
    fetchFn.mockResolvedValue({ ok: true, json: async () => ({ nope: 1 }) })

    await ensureDbOwner()

    expect(deleteFn).not.toHaveBeenCalled()
    expect(prefs.get('db_owner')?.value).toBe('user-a')
  })
})
