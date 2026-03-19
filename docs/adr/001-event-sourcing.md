# ADR-001: Event Sourcing as Core Architecture

**Status:** Accepted

## Context

DoIt is a personal task management application that also serves as a learning
vehicle for event sourcing, CRDTs, and distributed systems patterns. We need an
architecture that provides:

- A full audit trail of every change made to every task, list, and label.
- The ability to rebuild application state from scratch at any point.
- A foundation for offline-first sync with conflict resolution.
- A meaningful learning experience with real-world distributed systems patterns.

Traditional CRUD with a mutable database would be simpler but would not provide
an audit trail, rebuild capability, or a natural integration point for async
event distribution.

## Decision

We will use **append-only event sourcing** as the core architecture:

- The **event store** (PostgreSQL `events` table) is the single source of truth.
- All state mutations produce domain events (e.g., `TaskCreated`,
  `TaskCompleted`) that are appended to the event store.
- **Read models** are projections derived from replaying events. They exist for
  query performance and can be rebuilt from the event store at any time.
- Aggregates enforce business invariants before accepting commands and producing
  events.
- Each event carries an `aggregate_id`, `aggregate_type`, `event_type`,
  `user_id`, `data` (JSONB payload), HLC `timestamp`, and a monotonic `version`
  per aggregate.

## Consequences

**Benefits:**
- Complete audit trail — every change is recorded and immutable.
- Rebuild capability — read models can be destroyed and rebuilt from events.
- Natural fit for CRDT sync — events map directly to CRDT operations.
- Decoupled projections — new read models can be added without changing the
  write path.
- Learning value — hands-on experience with event sourcing trade-offs.

**Costs:**
- **Write amplification** — every mutation writes to the event store AND triggers
  read model updates.
- **Schema evolution** — changing event shapes requires versioning and migration
  strategies (upcasting).
- **Eventual consistency** — read models may lag behind the event store briefly.
- **Complexity** — more moving parts than a simple CRUD application.
- **Snapshot management** — long-lived aggregates may need periodic snapshots to
  avoid replaying thousands of events.
