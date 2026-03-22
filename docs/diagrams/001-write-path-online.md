# Write Path — Online (HTTP)

How a command from the REST API becomes an event in the store and eventually updates the read model.

```mermaid
sequenceDiagram
    participant Client
    participant Handler as HTTP Handler
    participant CMD as CommandHandler
    participant Agg as Aggregate
    participant HLC as HLC Clock
    participant PG as Postgres TX
    participant ES as events table
    participant OB as outbox table

    Client->>Handler: POST /api/v1/tasks
    Handler->>Handler: Parse JSON + extract userID (JWT)
    Handler->>CMD: CreateTask(cmd)
    CMD->>Agg: NewTaskAggregate()
    CMD->>HLC: Now()
    HLC-->>CMD: hlc.Timestamp
    CMD->>Agg: HandleCreate(cmd, ts)
    Agg->>Agg: Validate (title, priority, due_time)
    Agg-->>CMD: []Event{TaskCreated}

    CMD->>PG: BEGIN
    CMD->>ES: AppendTx(tx, events)
    ES-->>CMD: ok (or ErrVersionConflict)
    CMD->>OB: InsertOutbox(tx, events)
    OB-->>CMD: ok
    CMD->>PG: COMMIT

    CMD-->>Handler: nil
    Handler-->>Client: 201 Created {id}
```

**Key invariants:**
- Events and outbox rows are written in a single Postgres transaction (no dual-write)
- The HLC clock provides causal ordering — never `time.Now()` directly
- Aggregate validates all business rules before producing events
- Response is immediate — projections happen asynchronously via workers
