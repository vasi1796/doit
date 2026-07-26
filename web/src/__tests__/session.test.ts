import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { signOut } from '../session'

const { deleteFn } = vi.hoisted(() => ({ deleteFn: vi.fn() }))

vi.mock('../db/database', () => ({ db: { delete: deleteFn } }))

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
    location.href = ''
    vi.stubGlobal('window', { __syncEngine: { drain: drainFn, halt: haltFn }, location })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }))
  })

  afterEach(() => {
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

  it('works when no sync engine is mounted', async () => {
    vi.stubGlobal('window', { location })

    await signOut()

    expect(deleteFn).toHaveBeenCalledOnce()
    expect(location.href).toBe('/login')
  })
})
