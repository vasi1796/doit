# Data Loop — How Server Changes Reach the Client

How events produced on the server (by workers or other devices) flow back to the client's IndexedDB.

```mermaid
sequenceDiagram
    participant PW as Projection Worker
    participant RM as Read Models (Postgres)
    participant ES as Events Table (Postgres)
    participant Sync as POST /api/v1/sync
    participant SE as Sync Engine (client)
    participant IDB as IndexedDB
    participant UI as React (useLiveQuery)

    Note over PW,ES: Server-side: worker processes event
    PW->>RM: Upsert read model row
    Note over ES: Event already stored<br/>(from command handler)

    Note over SE,IDB: Client-side: periodic sync pull
    SE->>Sync: {operations: [], cursor: lastHLC}
    Sync->>ES: LoadByUserSince(userID, cursor)
    ES-->>Sync: new events since cursor
    Sync-->>SE: {events: [...], cursor: newHLC}

    SE->>SE: mergeRemoteEvents(events)
    loop For each event
        SE->>IDB: LWW merge (compare HLC timestamps)
    end

    IDB-->>UI: useLiveQuery fires → re-render
```

## Two paths to the client

| Path | When | Source | Used for |
|------|------|--------|----------|
| **Initial sync** | First app load (no cursor) | `GET /api/v1/tasks` → read models | Bootstrapping IndexedDB with full state |
| **Incremental sync** | Every 30s + on foreground | `POST /api/v1/sync` → events table | Pulling new events from other devices / workers |

**Key insight:** After the initial load, the client never reads from the Postgres read models again. It stays in sync through the **events table** — the same immutable log that is the source of truth for everything. The read models exist for the REST API and initial sync only.
