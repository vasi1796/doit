import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { SyncEngine } from '../db/sync-engine'
import { mergeRemoteEvents } from '../db/merge-events'

const { bulkDeleteSpy, cursorPutSpy, queuedOps } = vi.hoisted(() => ({
  bulkDeleteSpy: vi.fn(),
  cursorPutSpy: vi.fn(),
  queuedOps: [] as Record<string, unknown>[],
}))

vi.mock('../db/database', () => ({
  db: {
    syncQueue: {
      orderBy: () => ({ toArray: async () => queuedOps }),
      bulkDelete: bulkDeleteSpy,
      get: async (id: number) => queuedOps.find((op) => op.id === id),
      update: async () => {},
    },
    syncState: {
      get: async () => undefined,
      put: cursorPutSpy,
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

let locationStub: { protocol: string; host: string; href: string }

beforeEach(() => {
  vi.clearAllMocks()
  queuedOps.length = 0
  vi.mocked(mergeRemoteEvents).mockResolvedValue(true)
  fetchResolvers = []
  fetchMock = vi.fn(
    () => new Promise((resolve) => fetchResolvers.push(resolve)),
  )
  locationStub = { protocol: 'http:', host: 'test', href: '' }
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('window', { dispatchEvent: () => {}, location: locationStub })
  vi.stubGlobal('document', {
    addEventListener: () => {},
    removeEventListener: () => {},
  })
  vi.stubGlobal('location', locationStub)
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

describe('halt', () => {
  it('a response landing after halt() cannot write to the database', async () => {
    const engine = new SyncEngine()
    const run = engine.sync()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    engine.halt()
    fetchResolvers[0]({
      ok: true,
      status: 200,
      json: async () => ({ cursor: { hlc_time: 1, hlc_counter: 0 }, events: [{ id: 'e1' }] }),
    })
    await run

    expect(bulkDeleteSpy).not.toHaveBeenCalled()
    expect(cursorPutSpy).not.toHaveBeenCalled()
    expect(mergeRemoteEvents).not.toHaveBeenCalled()
  })

  it('sync() after halt() is a no-op', async () => {
    const engine = new SyncEngine()
    engine.halt()
    await engine.sync()
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('drain', () => {
  it('awaits the in-flight run and flushes once more', async () => {
    const engine = new SyncEngine()
    const first = engine.sync()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    let drained = false
    const drain = engine.drain().then(() => {
      drained = true
    })
    await new Promise((r) => setTimeout(r, 10))
    // Must not resolve while the first run is still in flight — a plain
    // sync() here would have resolved without flushing anything
    expect(drained).toBe(false)

    fetchResolvers[0](okResponse)
    await first
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
    fetchResolvers[1](okResponse)
    await drain
    expect(drained).toBe(true)
  })
})

describe('cursor persistence', () => {
  const eventsResponse = {
    ok: true,
    status: 200,
    json: async () => ({ cursor: { hlc_time: 5, hlc_counter: 1 }, events: [{ id: 'e1' }] }),
  }

  it('merges events before persisting the cursor', async () => {
    const engine = new SyncEngine()
    const run = engine.sync()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    fetchResolvers[0](eventsResponse)
    await run

    expect(cursorPutSpy).toHaveBeenCalledWith({ key: 'cursor', hlcTime: 5, hlcCounter: 1 })
    // Ordering is the invariant: a cursor written first would mark events as
    // consumed before they are applied, so a crash in between loses them.
    expect(vi.mocked(mergeRemoteEvents).mock.invocationCallOrder[0])
      .toBeLessThan(cursorPutSpy.mock.invocationCallOrder[0])
  })

  it('a failed merge leaves the cursor untouched so events are redelivered', async () => {
    vi.mocked(mergeRemoteEvents).mockResolvedValue(false)
    const notify = vi.fn()
    const engine = new SyncEngine()
    engine.setNotifier(notify)
    const run = engine.sync()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    fetchResolvers[0](eventsResponse)
    await run

    expect(cursorPutSpy).not.toHaveBeenCalled()
    expect(notify).toHaveBeenCalledOnce()
  })
})

describe('retry expiry', () => {
  it('discarding an op past max retries surfaces a notification', async () => {
    queuedOps.push({
      id: 1, operationType: 'UpdateTask', aggregateId: 'a', data: '{}',
      hlcTime: 1, hlcCounter: 0, createdAt: 1, retryCount: 5,
    })
    const notify = vi.fn()
    const engine = new SyncEngine()
    engine.setNotifier(notify)
    const run = engine.sync()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    fetchResolvers[0]({ ok: true, status: 200, json: async () => ({ failed_ops: [0] }) })
    await run

    expect(bulkDeleteSpy).toHaveBeenCalledWith([1])
    expect(notify).toHaveBeenCalledWith('An offline change could not be synced and was discarded')
  })
})

describe('session expiry', () => {
  it('a 401 halts the engine and redirects to login without wiping', async () => {
    const engine = new SyncEngine()
    const run = engine.sync()
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))

    fetchResolvers[0]({ ok: false, status: 401 })
    await run

    expect(locationStub.href).toBe('/login')
    expect(bulkDeleteSpy).not.toHaveBeenCalled()
    expect(cursorPutSpy).not.toHaveBeenCalled()

    await engine.sync() // halted — must not flush under a dead session
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
