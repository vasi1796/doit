import type { SyncEngine } from './db/sync-engine'

declare global {
  interface Window {
    __syncEngine?: SyncEngine
  }
}
