import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { authAwareFetch } from '../api/http'

describe('authAwareFetch', () => {
  const location = { href: '' }

  beforeEach(() => {
    location.href = ''
    vi.stubGlobal('window', { location })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends session credentials', async () => {
    const fetchFn = vi.fn().mockResolvedValue({ status: 200 })
    vi.stubGlobal('fetch', fetchFn)

    await authAwareFetch('/api/v1/thing', { method: 'POST' })

    expect(fetchFn).toHaveBeenCalledWith('/api/v1/thing', expect.objectContaining({ credentials: 'include', method: 'POST' }))
  })

  it('redirects to login and throws on 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ status: 401 }))

    await expect(authAwareFetch('/api/v1/thing')).rejects.toThrow('Unauthorized')
    expect(location.href).toBe('/login')
  })

  it('returns non-401 responses to the caller untouched', async () => {
    const res = { status: 500, ok: false }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(res))

    await expect(authAwareFetch('/api/v1/thing')).resolves.toBe(res)
    expect(location.href).toBe('')
  })
})
