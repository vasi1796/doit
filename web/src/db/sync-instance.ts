import type { SyncEngine } from './sync-engine'

let instance: SyncEngine | null = null

/** AppLayout registers the app-wide engine on mount; operations and session
 * reach it through this module instead of a window global. */
export function setSyncEngine(engine: SyncEngine | null): void {
  instance = engine
}

export function getSyncEngine(): SyncEngine | null {
  return instance
}
