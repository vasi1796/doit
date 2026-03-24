import { db } from './database'
import { mergeRemoteEvents } from './merge-events'

const BASE_INTERVAL = 30_000     // 30 seconds
const MAX_INTERVAL = 300_000     // 5 minutes
const JITTER_RANGE = 5_000       // ±5 seconds

// WebSocket reconnection
const WS_BASE_DELAY = 1_000     // 1 second
const WS_MAX_DELAY = 30_000     // 30 seconds

/**
 * SyncEngine handles push (queue flush) and pull (remote event merge).
 * Uses HTTP polling as the reliable baseline and WebSocket for near-instant updates.
 */
export class SyncEngine {
  private timerId: ReturnType<typeof setTimeout> | null = null
  private interval = BASE_INTERVAL
  private syncing = false

  // WebSocket
  private ws: WebSocket | null = null
  private wsDelay = WS_BASE_DELAY
  private wsTimerId: ReturnType<typeof setTimeout> | null = null
  private stopped = false

  start(): void {
    this.stopped = false
    document.addEventListener('visibilitychange', this.handleVisibility)
    // Short delay before first sync, then periodic polling
    this.timerId = setTimeout(() => {
      this.sync().finally(() => this.scheduleNext())
    }, 2_000)
    // Connect WebSocket for real-time push
    this.connectWS()
  }

  stop(): void {
    this.stopped = true
    document.removeEventListener('visibilitychange', this.handleVisibility)
    if (this.timerId !== null) {
      clearTimeout(this.timerId)
      this.timerId = null
    }
    if (this.wsTimerId !== null) {
      clearTimeout(this.wsTimerId)
      this.wsTimerId = null
    }
    if (this.nudgeTimer !== null) {
      clearTimeout(this.nudgeTimer)
      this.nudgeTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  async sync(): Promise<void> {
    if (this.syncing) return
    this.syncing = true
    window.dispatchEvent(new Event('sync:start'))

    try {
      const ops = await db.syncQueue.orderBy('createdAt').toArray()
      const cursor = await db.syncState.get('cursor')

      const body = {
        operations: ops.map((op) => ({
          type: op.operationType,
          aggregate_id: op.aggregateId,
          data: JSON.parse(op.data),
          hlc_time: op.hlcTime,
          hlc_counter: op.hlcCounter,
        })),
        cursor: cursor
          ? { hlc_time: cursor.hlcTime, hlc_counter: cursor.hlcCounter }
          : null,
      }

      const res = await fetch('/api/v1/sync', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      if (res.status === 401) {
        window.location.href = '/login'
        return
      }

      if (!res.ok) {
        const body = await res.text().catch(() => '(unreadable)')
        console.warn(`sync: HTTP ${res.status} — ${body}`)
        this.increaseBackoff()
        return
      }

      const result = await res.json()

      const failedIndices = new Set<number>(result.failed_ops ?? [])
      const MAX_RETRIES = 5

      const successOpIds: number[] = []
      const failedOps: { id: number; index: number }[] = []

      for (let i = 0; i < ops.length; i++) {
        const opId = ops[i].id
        if (!opId) continue
        if (failedIndices.has(i)) {
          failedOps.push({ id: opId, index: i })
        } else {
          successOpIds.push(opId)
        }
      }

      await db.syncQueue.bulkDelete(successOpIds)

      if (failedOps.length > 0) {
        console.warn(`sync: ${failedOps.length} operation(s) failed, will retry`)
        const expiredIds: number[] = []
        for (const { id } of failedOps) {
          const op = await db.syncQueue.get(id)
          if (!op) continue
          const newRetryCount = (op.retryCount ?? 0) + 1
          if (newRetryCount > MAX_RETRIES) {
            console.error(`sync: operation ${id} (${op.operationType} on ${op.aggregateId}) exceeded ${MAX_RETRIES} retries, discarding`)
            expiredIds.push(id)
          } else {
            await db.syncQueue.update(id, { retryCount: newRetryCount })
          }
        }
        if (expiredIds.length > 0) {
          await db.syncQueue.bulkDelete(expiredIds)
        }
      }

      if (result.cursor) {
        await db.syncState.put({
          key: 'cursor',
          hlcTime: result.cursor.hlc_time,
          hlcCounter: result.cursor.hlc_counter,
        })
      }

      if (result.events && result.events.length > 0) {
        await mergeRemoteEvents(result.events)
      }

      this.resetBackoff()
    } catch (err) {
      console.warn('sync: poll failed', err)
      this.increaseBackoff()
    } finally {
      this.syncing = false
      window.dispatchEvent(new Event('sync:end'))
    }
  }

  /** Trigger a sync soon after a local write. Debounces rapid writes. */
  nudge(): void {
    if (this.nudgeTimer) return
    this.nudgeTimer = setTimeout(() => {
      this.nudgeTimer = null
      this.sync()
    }, 500) // 500ms debounce — batches rapid writes
  }

  private nudgeTimer: ReturnType<typeof setTimeout> | null = null

  // --- WebSocket ---

  private connectWS(): void {
    if (this.stopped) return

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/api/v1/ws`

    try {
      this.ws = new WebSocket(url)
    } catch (err) {
      console.warn('sync: WebSocket connect failed', err)
      this.scheduleWSReconnect()
      return
    }

    this.ws.onmessage = async (event) => {
      try {
        const events = JSON.parse(event.data)
        if (Array.isArray(events) && events.length > 0) {
          await mergeRemoteEvents(events)
        }
      } catch (err) {
        console.warn('sync: WebSocket message parse failed', err)
      }
    }

    this.ws.onopen = () => {
      this.wsDelay = WS_BASE_DELAY
    }

    this.ws.onclose = () => {
      this.ws = null
      if (!this.stopped) {
        this.scheduleWSReconnect()
      }
    }

    this.ws.onerror = () => {}  // onclose handles reconnect
  }

  private scheduleWSReconnect(): void {
    if (this.stopped) return
    const jitter = Math.round(Math.random() * 1000)
    this.wsTimerId = setTimeout(() => {
      this.connectWS()
      // On reconnect, trigger full sync to catch missed events
      this.sync()
    }, this.wsDelay + jitter)
    this.wsDelay = Math.min(this.wsDelay * 2, WS_MAX_DELAY)
  }

  // --- Polling ---

  private handleVisibility = (): void => {
    if (document.visibilityState === 'visible') {
      this.sync()
    }
  }

  private scheduleNext(): void {
    const jitter = Math.round((Math.random() - 0.5) * 2 * JITTER_RANGE)
    this.timerId = setTimeout(() => {
      this.sync().finally(() => this.scheduleNext())
    }, this.interval + jitter)
  }

  private resetBackoff(): void {
    this.interval = BASE_INTERVAL
  }

  private increaseBackoff(): void {
    this.interval = Math.min(this.interval * 2, MAX_INTERVAL)
  }
}
