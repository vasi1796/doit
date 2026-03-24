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

---

## RabbitMQ Reconnection Flow

What happens when the broker connection drops (e.g., RabbitMQ restart, network hiccup).

```mermaid
sequenceDiagram
    participant B as Broker
    participant WG as watchConnection<br/>goroutine
    participant RMQ as RabbitMQ
    participant W as Worker

    B->>WG: Start (on New())
    WG->>RMQ: conn.NotifyClose(ch)
    Note over WG: Blocks waiting for<br/>close notification

    RMQ--xWG: Connection closed!
    WG->>WG: Log warning

    loop Reconnect with backoff (1s → 30s)
        WG->>RMQ: amqp.Dial(url)
        alt Connection failed
            WG->>WG: Sleep (backoff × jitter)
        else Connected
            WG->>RMQ: conn.Channel()
            WG->>B: Setup() — redeclare<br/>exchanges + queues
            WG->>WG: Close reconnected channel<br/>(signal consumers)
            WG->>WG: Create new reconnected<br/>channel for next cycle
            WG->>RMQ: conn.NotifyClose(ch)
            Note over WG: Resume watching
        end
    end

    Note over W: Delivery channel closes
    W->>B: Reconnected() — wait for signal
    B-->>W: Signal received
    W->>B: Consume(queue)
    B-->>W: New delivery channel
    Note over W: Resume processing
```

**Key points:**
- `watchConnection()` goroutine runs for the broker's lifetime
- Uses `NotifyClose` callback — no polling, instant detection
- Exponential backoff: 1s base, 30s max, with 75-125% jitter
- After reconnect, `Setup()` redeclares exchanges and queues (idempotent)
- Workers detect reconnection via `Reconnected()` channel and re-subscribe
- All access to conn/channel protected by `sync.RWMutex`
