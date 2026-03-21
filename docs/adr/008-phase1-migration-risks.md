# ADR-008: Phase 1 Implementation — Known Migration Points

**Status:** Accepted

## Context

Phase 1 implements a synchronous, online-only architecture. Phases 2-4 introduce
offline-first CRDTs, a transactional outbox with RabbitMQ, and batch processing.
This ADR documents the known refactor points in the Phase 1 codebase that will
need attention when migrating to later phases.

## Phase 2: Offline-First + CRDT Sync

### Timestamps: wall clock → HLC

**Current:** `CommandHandler` passes `time.Now().UTC()` to aggregate handlers.

**Migration:** Replace with a Hybrid Logical Clock. The aggregates already accept
`now time.Time` as a parameter — no aggregate changes needed. Only the
`CommandHandler` and the new sync endpoint need to use the HLC clock.

**Risk:** Low. The parameterized timestamp design was intentional.

### Sync endpoint

**Current:** Clients call individual REST endpoints (POST, PATCH, DELETE).

**Migration:** Add `POST /api/v1/sync` that accepts batched CRDT operations from
the client and returns remote events since the client's last sync. The event
store's `LoadByUserSince` method already supports the pull side. The push side
needs to validate and merge incoming events with conflict resolution.

**Risk:** Low. The event store and aggregate replay infrastructure support this.

### Client-generated IDs

**Current:** The frontend generates UUIDs for task/list/label IDs.

**Migration:** No change needed — this already works for offline-first.

## Phase 3: Transactional Outbox + RabbitMQ

### appendAndProject → outbox pattern

**Current:** `CommandHandler.appendAndProject` calls `Store.Append` then
`Projector.Project` synchronously. If projection fails, events are stored but
read models are stale.

**Migration:** Replace with:
1. `Store.Append` writes events + outbox rows in a single Postgres transaction
2. An outbox poller publishes to RabbitMQ
3. Projection workers consume from RabbitMQ and call `Projector.Project`

**Refactor needed:** `Store.Append` currently manages its own transaction. To
write outbox rows atomically, either:
- Pass an external `pgx.Tx` to `Append` (changes the method signature)
- Create a new `AppendWithOutbox` method that wraps both writes
- Move outbox logic into `Store.Append` (couples store to outbox — less clean)

Recommendation: add `AppendTx(ctx, tx, events)` that accepts an external
transaction, keep `Append` as a convenience wrapper. The outbox publisher uses
`AppendTx` within a transaction that also writes the outbox row.

**Risk:** Medium. One method signature change, but it propagates through
`CommandHandler` and `EventLoader` interface. Plan for this to touch ~5 files.

### Projector becomes a worker

**Current:** `Projector.Project` is called synchronously by the command handler.

**Migration:** The same `Projector.Project` method is called by RabbitMQ consumer
workers instead. The projector itself doesn't change — only who calls it changes.

**Risk:** Low. The projector is already stateless and idempotent.

## Phase 4: Projection Rebuilder + Batch Processing

### Projection rebuild

**Current:** Projector handles all event types with idempotent SQL.

**Migration:** A CLI tool loads all events via `Store.LoadByUserSince` (or a new
`LoadAll` method) and feeds them through `Projector.Project`. The projector's
`ON CONFLICT` upserts make replay safe.

**Risk:** None. The architecture was designed for this.

### Aggregate snapshots

**Current:** Every command loads all events for an aggregate via
`loadTaskAggregate`, which replays from event #1.

**Migration:** Periodically snapshot aggregate state. On load, start from the
latest snapshot + replay only events after the snapshot version. The
`LoadByAggregateFromVersion` method already exists for this.

**Risk:** Low. Needs a snapshot store and a change to `loadTaskAggregate`, but
the query infrastructure is already in place.

## Consequences

The Phase 1 architecture deliberately chose simplicity (synchronous projections,
wall-clock timestamps, direct SQL) while keeping the interfaces compatible with
the Phase 2-4 requirements. The known migration points are:

1. **HLC clock** — swap `time.Now()` calls in CommandHandler (low effort)
2. **Outbox pattern** — add `AppendTx` to event store, refactor CommandHandler (medium effort)
3. **Sync endpoint** — new handler using existing event store methods (medium effort)
4. **Snapshot loading** — extend `loadTaskAggregate` (low effort)

None of these require structural rewrites. The aggregate, event store, and
projector interfaces remain stable across all phases.
