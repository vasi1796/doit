# System Overview — Full Architecture

High-level view of all components and how they connect.

```mermaid
flowchart TB
    subgraph Client["Browser (Safari PWA)"]
        direction LR
        React[React SPA] <-->|useLiveQuery| IDB[(IndexedDB)]
        React -->|writes| IDB
        IDB -->|syncQueue| SE[Sync Engine]
    end

    SE <-->|"POST /api/v1/sync"| Server

    subgraph Server["Go Backend"]
        direction LR
        Router[Chi Router + JWT] --> Handlers[HTTP Handlers]
        Handlers --> CMD[CommandHandler + HLC]
        CMD --> Agg[Aggregates]
    end

    CMD -->|"atomic TX"| PG

    subgraph PG["PostgreSQL"]
        direction LR
        Events[(events)] ~~~ Outbox[(outbox)] ~~~ RM[(read models)]
    end

    Outbox -->|"poll 200ms"| Poller[Outbox Poller]
    Poller -->|publish| MQ

    subgraph MQ["RabbitMQ"]
        direction LR
        Exchange{{topic exchange}} --> ProjQ[projections queue]
        Exchange -->|TaskCompleted| RecQ[recurring queue]
    end

    ProjQ --> ProjW[Projection Worker]
    RecQ --> RecW[Recurring Worker]

    ProjW -->|upsert| RM
    RecW -->|new events| Events
```

**Component responsibilities:**

| Component | Role |
|-----------|------|
| **React SPA** | UI rendering, all reads from IndexedDB via `useLiveQuery` |
| **IndexedDB (Dexie.js)** | Client-side source of truth, drives UI reactivity |
| **Sync Engine** | Background push/pull — batches ops to `/api/v1/sync` every 30s |
| **CommandHandler** | Loads aggregate, validates via HLC clock, atomic event + outbox write |
| **Outbox Poller** | Polls every 200ms, publishes to RabbitMQ, marks as published |
| **Projection Worker** | Consumes all events, upserts into read model tables (idempotent) |
| **Recurring Worker** | Consumes `TaskCompleted`, creates next occurrence if recurring |
| **Rebuild CLI** | Disaster recovery — replays full event log to reconstruct read models |

**Data flow summary:**
1. Client writes to IndexedDB instantly, queues sync op
2. Sync engine pushes ops to server, pulls remote events back
3. Server validates commands, writes events + outbox atomically
4. Poller publishes outbox to RabbitMQ
5. Workers consume and update read models / create recurring tasks
