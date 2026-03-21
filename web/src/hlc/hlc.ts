/**
 * Hybrid Logical Clock (HLC) for client-side causal ordering.
 *
 * An HLC timestamp combines a physical wall-clock time (ms since epoch)
 * with a logical counter. This provides a total ordering that respects
 * causality while staying close to wall-clock time.
 *
 * Used by the sync engine (Phase 2) to order offline changes
 * and resolve LWW conflicts deterministically.
 */

export interface HLCTimestamp {
  /** Physical time in milliseconds since Unix epoch. */
  time: number
  /** Logical counter — breaks ties when physical time is equal. */
  counter: number
}

/** Compare two HLC timestamps. Returns -1 if a < b, 0 if equal, 1 if a > b. */
export function compare(a: HLCTimestamp, b: HLCTimestamp): number {
  if (a.time < b.time) return -1
  if (a.time > b.time) return 1
  if (a.counter < b.counter) return -1
  if (a.counter > b.counter) return 1
  return 0
}

export class HLCClock {
  private latest: HLCTimestamp = { time: 0, counter: 0 }
  private nowFn: () => number

  constructor(nowFn?: () => number) {
    this.nowFn = nowFn ?? (() => Date.now())
  }

  /** Generate a new HLC timestamp for a local event. */
  now(): HLCTimestamp {
    const pt = this.nowFn()

    if (pt > this.latest.time) {
      this.latest = { time: pt, counter: 0 }
    } else {
      this.latest = { time: this.latest.time, counter: this.latest.counter + 1 }
    }

    return { ...this.latest }
  }

  /** Merge a remote HLC timestamp with the local clock state. */
  update(remote: HLCTimestamp): HLCTimestamp {
    const pt = this.nowFn()
    const prev = this.latest

    if (pt > prev.time && pt > remote.time) {
      this.latest = { time: pt, counter: 0 }
    } else if (prev.time > remote.time) {
      this.latest = { time: prev.time, counter: prev.counter + 1 }
    } else if (remote.time > prev.time) {
      this.latest = { time: remote.time, counter: remote.counter + 1 }
    } else {
      const maxC = Math.max(prev.counter, remote.counter)
      this.latest = { time: prev.time, counter: maxC + 1 }
    }

    return { ...this.latest }
  }
}
