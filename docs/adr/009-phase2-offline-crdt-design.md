# ADR-009: Phase 2 Offline-First & CRDT Sync Design

**Status:** Planned

## Context

Phase 1 delivers an online-only MVP. Phase 2 adds offline-first operation with
CRDT-based sync across multiple devices (iPhone, iPad, Mac). The user must be
able to create, edit, complete, and delete tasks while offline, with changes
merging automatically when connectivity returns.

This is fundamentally a multi-leader replication problem (DDIA Chapter 5). Each
device is a leader that accepts writes independently. The system must converge
to a consistent state without manual conflict resolution.

## Pre-requisite Reading

- DDIA Chapter 5: Replication (multi-leader, conflict handling, concurrent writes)
- DDIA Chapter 9: Consistency and Consensus (ordering, linearizability, HLC)
- Kleppmann: "CRDTs: The Hard Parts" (talk)
- James Long: "CRDTs for Mortals" (practical implementation)

## Decision

### Architecture

```
User Action → Dexie.js write (IndexedDB) → useLiveQuery re-render (instant)
                                          → Sync engine queues operation
                                          → On foreground/poll: POST /api/v1/sync
                                          → Server merges, responds with remote events
                                          → Client merges remote events into IndexedDB
                                          → useLiveQuery re-renders with merged state
```

### Implementation Slices

**Slice 1: Hybrid Logical Clock (HLC)**
Replace `time.Now().UTC()` in CommandHandler with HLC timestamps. The HLC
combines wall-clock time with a logical counter to provide causal ordering
without clock synchronization. Implement in both Go (server) and TypeScript
(client). The aggregate `Handle*` methods already accept `now time.Time` as
a parameter — only the caller changes.

**Slice 2: Dexie.js Local Storage**
Replace the Phase 1 `useState + useEffect + fetch` hooks with Dexie.js
`useLiveQuery`. All reads come from IndexedDB. All writes go to IndexedDB
first. The UI updates instantly from local state without waiting for the
server. The `LayoutContext` React Context is removed — Dexie's reactive
queries replace it.

IndexedDB schema mirrors the server read model tables:
- `tasks` — same columns as Postgres tasks table
- `lists` — same as lists table
- `labels` — same as labels table
- `task_labels` — join table
- `subtasks` — same as subtasks table
- `sync_queue` — pending operations to send to server
- `sync_state` — last sync timestamp per aggregate

**Slice 3: Service Worker**
Workbox for app shell caching. The PWA must launch instantly from the home
screen with zero connectivity. Cache-first strategy for static assets (JS,
CSS, icons). Network-first for API calls (handled by sync engine, not
service worker). No Background Sync API (Safari doesn't support it).

**Slice 4: CRDT Types**
Shared implementations in Go and TypeScript:
- **LWW-Register** — for scalar fields (title, description, priority, due_date,
  due_time, recurrence_rule). Each field tracks its own HLC timestamp independently
  (per-field LWW), so concurrent edits to different fields on the same task are
  both preserved. Compare HLC timestamps per field, keep later value.
- **OR-Set** (Observed-Remove Set) — for labels on a task. Tracks add/remove
  operations with unique tags. Concurrent add+remove of same label resolves
  correctly.
- **Fractional Indexing** — for task ordering within lists. Position is a string
  that sorts lexicographically between any two adjacent items.

**Slice 5: Sync Engine — Push**
Client-side sync engine triggered by:
- `visibilitychange` event (app comes to foreground)
- Polling interval: 30s base, exponential backoff on failure (30→60→120,
  capped at 5 minutes), random jitter ±5s
Operations are batched from the `sync_queue` table and sent via
`POST /api/v1/sync`. Failed operations are retained in the queue with a retry
count (max 5 retries) instead of being permanently deleted on first failure.
The server validates, appends events, and returns a confirmation with the
server-assigned HLC timestamp.

**Slice 6: Sync Engine — Pull**
The sync endpoint returns remote events since the client's last sync
timestamp. The client merges these into local IndexedDB using the CRDT
merge functions. `useLiveQuery` automatically re-renders affected
components. The `LoadByUserSince` event store method already exists for
this.

**Slice 7: WebSocket Real-Time Push**
`WS /api/v1/ws` pushes events from other devices in real-time when online.
On disconnect: immediately fall back to polling, attempt reconnection with
exponential backoff (1s→2s→4s, capped at 30s) plus jitter. On successful
reconnection: perform a full sync pull to catch events missed during the
disconnect window.

**Slice 8: Aggregate Snapshots**
Per-aggregate server-side snapshots updated incrementally on each sync.
Keyed by `aggregate_id + user_id` in the existing `aggregate_snapshots`
table. Client tracks `last_synced_version` per aggregate for incremental
pull. If local IndexedDB is evicted by Safari, all snapshots for the user
are pulled and IndexedDB is rehydrated on next launch.

**Slice 9: Conflict Testing**
Test harness simulating two devices making conflicting offline edits:
- Concurrent title edit → LWW, later HLC wins
- Edit on device A + delete on device B → edit resurrects (policy from ADR-002)
- Concurrent label add + remove → OR-Set resolves
- Concurrent list moves → LWW on list_id
- Complete on device A + delete on device B → complete resurrects
All scenarios must converge to the same state regardless of merge order.

### Application-Level Conflict Policies (from ADR-002)

- **Edit resurrects delete**: concurrent edit + delete → edit wins, task restored
- **Complete resurrects delete**: concurrent complete + delete → complete wins
- **Concurrent list moves**: LWW based on HLC timestamp (one move silently lost)

### Key Constraints

- **Safari only**: No Background Sync API. Sync only while app is in foreground.
- **Storage eviction**: Safari may evict IndexedDB if PWA is unused for weeks.
  Server snapshots provide recovery.
- **Cache API limit**: ~50MB per origin. Use IndexedDB for task data, Cache API
  only for app shell.

## Consequences

**Benefits:**
- App works identically online and offline
- UI updates are instant (no network round trip)
- Multiple devices converge automatically
- Full audit trail preserved in event store

**Costs:**
- Significant implementation complexity (CRDTs, HLC, sync engine)
- Eventual consistency — devices may temporarily show different states
- LWW can silently lose concurrent edits (acceptable for 1-3 users)
- IndexedDB storage management needed for Safari eviction scenarios
- Two codebases for CRDT logic (Go server + TypeScript client)

## Migration from Phase 1

See ADR-008 for known refactor points. Key changes:
- `CommandHandler`: replace `time.Now()` with HLC clock (low effort)
- Frontend hooks: replace `useState+fetch` with Dexie.js `useLiveQuery` (medium effort)
- New sync endpoint: `POST /api/v1/sync` (new handler, reuses existing store methods)
- New WebSocket endpoint: `WS /api/v1/ws` (new handler)
- No aggregate or event store structural changes needed
