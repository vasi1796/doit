import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { signOut } from '../session'

const { deleteFn } = vi.hoisted(() => ({ deleteFn: vi.fn() }))

vi.mock('../db/database', () => ({ db: { delete: deleteFn } }))

describe('signOut', () => {
  const syncFn = vi.fn()
  const stopFn = vi.fn()
  const location = { href: '' }

  beforeEach(() => {
    vi.clearAllMocks()
    location.href = ''
    vi.stubGlobal('window', { __syncEngine: { sync: syncFn, stop: stopFn }, location })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('wipes local data and redirects even when flush and logout both fail', async () => {
    syncFn.mockRejectedValue(new Error('offline'))
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))

    await signOut()

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })

  it('attempts the flush and stops the engine before wiping', async () => {
    syncFn.mockResolvedValue(undefined)

    await signOut()

    expect(syncFn.mock.invocationCallOrder[0]).toBeLessThan(deleteFn.mock.invocationCallOrder[0])
    expect(stopFn.mock.invocationCallOrder[0]).toBeLessThan(deleteFn.mock.invocationCallOrder[0])
  })

  it('a hung flush cannot block sign-out beyond the bounded wait', async () => {
    vi.useFakeTimers()
    syncFn.mockReturnValue(new Promise(() => {}))

    const done = signOut()
    await vi.advanceTimersByTimeAsync(3000)
    await done

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })

  it('works when no sync engine is mounted', async () => {
    vi.stubGlobal('window', { location })

    await signOut()

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })
})
