# System Overview — Full Architecture

High-level view of all components and how they connect.

```mermaid
flowchart TB
    subgraph Client["Browser (Safari PWA)"]
        SW[Service Worker<br/>app shell cache]
        React[React SPA]
        IDB[(IndexedDB<br/>Dexie.js)]
        SE[Sync Engine<br/>30s poll + nudge]
        HLC_C[HLC Clock]
    end

    subgraph Server["Go Backend"]
        Router[Chi Router<br/>+ JWT Auth]
        Handlers[HTTP Handlers<br/>task / list / label / sync]
        CMD[CommandHandler]
        Agg[Aggregates<br/>Task / List / Label]
        HLC_S[HLC Clock]
        Poller[Outbox Poller<br/>200ms]
    end

    subgraph Postgres["PostgreSQL"]
        Events[(events<br/>append-only)]
        Outbox[(outbox)]
        ReadModels[(read models<br/>tasks, lists, labels,<br/>subtasks, task_labels)]
        Snapshots[(aggregate_snapshots)]
    end

    subgraph MQ["RabbitMQ"]
        Exchange{{doit.events<br/>topic exchange}}
        ProjQ[doit.projections]
        RecQ[doit.recurring]
        DLQ[doit.dead-letter]
    end

    subgraph Workers["Worker Processes"]
        ProjW[Projection Worker<br/>updates read models]
        RecW[Recurring Worker<br/>creates next task]
    end

    %% Client flows
    React <-->|useLiveQuery| IDB
    React -->|operations.ts| IDB
    React -->|queue syncOp| SE
    SE <-->|POST /api/v1/sync| Router
    SW -.->|cache-first| React

    %% Server flows
    Router --> Handlers
    Handlers --> CMD
    CMD --> Agg
    CMD --> HLC_S
    CMD -->|atomic TX| Events & Outbox

    %% Async pipeline
    Poller -->|poll| Outbox
    Poller -->|publish| Exchange
    Exchange --> ProjQ & RecQ
    ProjQ --> ProjW
    RecQ --> RecW
    ProjW -->|upsert| ReadModels
    RecW -->|new events| Events & Outbox

    %% Snapshots
    Handlers -.->|save on sync| Snapshots

    style Client fill:#e3f2fd,stroke:#1976d2
    style Server fill:#f3e5f5,stroke:#7b1fa2
    style Postgres fill:#fff3e0,stroke:#f57c00
    style MQ fill:#e8f5e9,stroke:#388e3c
    style Workers fill:#fce4ec,stroke:#c62828
```

**Component responsibilities:**
| Component | Role |
|-----------|------|
| **React SPA** | UI rendering, user interaction, all reads from IndexedDB |
| **Dexie.js / IndexedDB** | Client-side source of truth, live queries drive UI |
| **Sync Engine** | Background push/pull of operations to/from server |
| **Service Worker** | App shell caching for offline launch |
| **HTTP Handlers** | Request parsing, routing to CommandHandler |
| **CommandHandler** | Aggregate loading, HLC timestamping, transactional append |
| **Aggregates** | Business rule validation, event production |
| **Outbox Poller** | Reliable event publishing (no message loss) |
| **RabbitMQ** | Event routing to workers via topic exchange |
| **Projection Worker** | Async read model updates (idempotent) |
| **Recurring Worker** | Creates next task occurrence on completion |
| **Rebuild CLI** | Disaster recovery — replays event log to reconstruct read models |
