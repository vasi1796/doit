import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { signOut } from '../session'
import { setSyncEngine } from '../db/sync-instance'
import type { SyncEngine } from '../db/sync-engine'

const { deleteFn, unsubscribeFn } = vi.hoisted(() => ({
  deleteFn: vi.fn(),
  unsubscribeFn: vi.fn(),
}))

vi.mock('../db/database', () => ({ db: { delete: deleteFn } }))
vi.mock('../push', () => ({ unsubscribeFromPush: unsubscribeFn }))

describe('signOut', () => {
  const drainFn = vi.fn()
  const haltFn = vi.fn()
  const location = { href: '' }

  beforeEach(() => {
    vi.clearAllMocks()
    // clearAllMocks leaves implementations in place — a hung promise set by one
    // test would otherwise leak into the next
    deleteFn.mockResolvedValue(undefined)
    drainFn.mockResolvedValue(undefined)
    unsubscribeFn.mockResolvedValue(undefined)
    location.href = ''
    setSyncEngine({ drain: drainFn, halt: haltFn } as unknown as SyncEngine)
    vi.stubGlobal('window', { location })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }))
  })

  afterEach(() => {
    setSyncEngine(null)
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('wipes local data and redirects even when flush and logout both fail', async () => {
    drainFn.mockRejectedValue(new Error('offline'))
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))

    await signOut()

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })

  it('drains the queue and halts engine writes before wiping', async () => {
    drainFn.mockResolvedValue(undefined)

    await signOut()

    expect(drainFn.mock.invocationCallOrder[0]).toBeLessThan(deleteFn.mock.invocationCallOrder[0])
    expect(haltFn.mock.invocationCallOrder[0]).toBeLessThan(deleteFn.mock.invocationCallOrder[0])
  })

  it('a hung flush cannot block sign-out beyond the bounded wait', async () => {
    vi.useFakeTimers()
    drainFn.mockReturnValue(new Promise(() => {}))

    const done = signOut()
    await vi.advanceTimersByTimeAsync(3000)
    await done

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })

  it('a wipe blocked by another tab cannot strand the user on the app', async () => {
    vi.useFakeTimers()
    deleteFn.mockReturnValue(new Promise(() => {}))

    const done = signOut()
    // The wipe timer is only scheduled once the logout await settles
    await vi.advanceTimersByTimeAsync(0)
    await vi.advanceTimersByTimeAsync(3000)
    await done

    expect(location.href).toBe('/login')
  })

  it('unsubscribes from push before the session cookie is expired', async () => {
    const logout = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', logout)

    await signOut()

    // The unsubscribe DELETE is user-scoped, so it fails silently once the
    // logout has cleared the cookie — order is the whole point
    expect(unsubscribeFn.mock.invocationCallOrder[0]).toBeLessThan(logout.mock.invocationCallOrder[0])
  })

  it('a failing unsubscribe still wipes and redirects', async () => {
    unsubscribeFn.mockRejectedValue(new Error('no service worker'))

    await signOut()

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })

  it('works when no sync engine is mounted', async () => {
    setSyncEngine(null)

    await signOut()

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })
})
