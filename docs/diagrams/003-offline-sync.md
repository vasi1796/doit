# Offline-First Sync — Client ↔ Server

How writes happen locally first and sync to the server in the background.

```mermaid
sequenceDiagram
    participant UI as React Component
    participant Ops as operations.ts
    participant IDB as IndexedDB<br/>(Dexie.js)
    participant SQ as syncQueue
    participant SE as SyncEngine
    participant API as POST /api/v1/sync
    participant Merge as merge-events.ts

    UI->>Ops: createTask(data)
    Ops->>Ops: Generate UUID + HLC timestamp

    par Optimistic Write
        Ops->>IDB: db.tasks.put(task)
        Note over IDB: useLiveQuery fires →<br/>UI re-renders instantly
    and Queue Sync
        Ops->>SQ: db.syncQueue.add(syncOp)
    end

    Ops->>SE: nudge() (debounced 500ms)

    Note over SE: 30s polling / nudge / foreground

    SE->>SQ: Read pending ops
    SE->>API: POST {operations, cursor}
    API->>API: For each op:<br/>Update HLC → dispatch command<br/>→ append events + outbox

    API-->>SE: {cursor, events, failed_ops}

    SE->>SQ: Delete synced ops
    SE->>Merge: mergeRemoteEvents(events)

    loop For each remote event
        Merge->>Merge: Update client HLC
        Merge->>IDB: LWW merge (compare HLC)<br/>→ update if remote wins
    end

    Note over IDB: useLiveQuery fires →<br/>UI shows merged state
```

**Key points:**
- Writes are **instant** — IndexedDB updated before network call
- Sync engine retries with exponential backoff (5s → 10s → ... → 5min)
- Server returns new events from other devices — merged via LWW-Register
- No rollback on sync failure — operations stay in queue and retry
- Client HLC is updated from remote events to maintain causal ordering
