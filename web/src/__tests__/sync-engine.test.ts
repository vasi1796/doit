import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { SyncEngine } from '../db/sync-engine'

vi.mock('../db/database', () => ({
  db: {
    syncQueue: {
      orderBy: () => ({ toArray: async () => [] }),
      bulkDelete: async () => {},
      get: async () => undefined,
      update: async () => {},
    },
    syncState: {
      get: async () => undefined,
      put: async () => {},
    },
  },
}))

vi.mock('../db/merge-events', () => ({
  mergeRemoteEvents: vi.fn(),
}))

const okResponse = { ok: true, status: 200, json: async () => ({}) }

// Fake WebSocket capturing the last created instance so tests can fire
// onmessage manually.
class FakeWebSocket {
  static last: FakeWebSocket | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  constructor() {
    FakeWebSocket.last = this
  }
  close() {}
}

let fetchResolvers: ((value: unknown) => void)[]
let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  fetchResolvers = []
  fetchMock = vi.fn(
    () => new Promise((resolve) => fetchResolvers.push(resolve)),
  )
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('window', { dispatchEvent: () => {} })
  vi.stubGlobal('document', {
    addEventListener: () => {},
    removeEventListener: () => {},
  })
  vi.stubGlobal('location', { protocol: 'http:', host: 'test' })
  vi.stubGlobal('WebSocket', FakeWebSocket)
  FakeWebSocket.last = null
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('sync coalescing', () => {
  it('a sync requested mid-flight runs exactly one trailing sync', async () => {
    const engine = new SyncEngine()

    const first = engine.sync()
    const second = engine.sync() // in flight — must be remembered, not dropped
    const third = engine.sync() // coalesces with second into ONE trailing run
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    fetchResolvers[0](okResponse)
    await Promise.all([first, second, third])

    // The trailing sync fires from the finally block on a microtask.
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))

    fetchResolvers[1](okResponse)
    // Settle: no further syncs may fire after the trailing one completes.
    await new Promise((r) => setTimeout(r, 10))
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('sequential syncs do not trigger a trailing sync', async () => {
    const engine = new SyncEngine()

    const first = engine.sync()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))
    fetchResolvers[0](okResponse)
    await first

    await new Promise((r) => setTimeout(r, 10))
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})

describe('WebSocket ping handling', () => {
  const cases: { name: string; data: string; wantSync: boolean }[] = [
    { name: 'sync ping triggers a pull', data: '{"type":"sync"}', wantSync: true },
    { name: 'unknown message type is ignored', data: '{"type":"other"}', wantSync: false },
    { name: 'legacy event array is ignored', data: '[{"id":"x"}]', wantSync: false },
    { name: 'malformed JSON is ignored without crashing', data: 'not json', wantSync: false },
  ]

  for (const tc of cases) {
    it(tc.name, async () => {
      const engine = new SyncEngine() as unknown as {
        connectWS: () => void
        sync: () => Promise<void>
      }
      engine.connectWS()
      const ws = FakeWebSocket.last
      expect(ws).not.toBeNull()

      ws!.onmessage!({ data: tc.data })

      if (tc.wantSync) {
        await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))
        fetchResolvers[0](okResponse)
      } else {
        await new Promise((r) => setTimeout(r, 10))
        expect(fetchMock).not.toHaveBeenCalled()
      }
    })
  }
})
