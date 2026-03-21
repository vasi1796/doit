# Async Pipeline — Outbox → RabbitMQ → Workers → Read Models

How events flow from the outbox through RabbitMQ to projection and recurring workers.

```mermaid
flowchart LR
    subgraph Postgres
        OB[(outbox)]
        ES[(events)]
        RM[(read models<br/>tasks, lists,<br/>labels, subtasks)]
    end

    subgraph API Server
        Poller[Outbox Poller<br/>200ms interval]
    end

    subgraph RabbitMQ
        EX{{doit.events<br/>topic exchange}}
        QP[doit.projections<br/>binding: #]
        QR[doit.recurring<br/>binding: TaskCompleted]
        DLQ[doit.dead-letter]
    end

    subgraph Workers
        PW[Projection Worker]
        RW[Recurring Worker]
    end

    OB -->|SELECT ... FOR UPDATE<br/>SKIP LOCKED| Poller
    Poller -->|Publish<br/>routing_key = EventType| EX
    Poller -->|UPDATE published=true| OB
    EX --> QP
    EX -->|TaskCompleted only| QR
    QP --> PW
    QR --> RW
    PW -->|ON CONFLICT<br/>DO UPDATE| RM
    RW -->|CreateTask +<br/>UpdateRecurrence +<br/>AddLabel| ES
    RW -->|new outbox rows| OB
    QP -.->|nack on failure| DLQ
    QR -.->|nack on failure| DLQ
```

**Key points:**
- Outbox poller uses `FOR UPDATE SKIP LOCKED` for safe concurrent polling
- Topic exchange routes by event type — projections get all events, recurring only gets `TaskCompleted`
- Projection worker is idempotent — all handlers use `ON CONFLICT DO UPDATE`
- Recurring worker creates new events (which cycle back through the same pipeline)
- Failed messages go to the dead-letter queue for manual inspection
