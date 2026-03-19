# ADR-004: Transactional Outbox Pattern

**Status:** Accepted

## Context

When a command is processed, DoIt must:

1. Append event(s) to the PostgreSQL event store.
2. Publish those events to RabbitMQ for async consumers (projection workers).

This is a **dual-write problem**. Writing to Postgres and publishing to RabbitMQ
are two separate operations that cannot be made atomic with a standard database
transaction. If the application crashes between step 1 and step 2, events are
stored but never published, and consumers never learn about them.

Possible solutions:
- **Distributed transactions (2PC)**: Complex, slow, poorly supported by RabbitMQ.
- **Change Data Capture (CDC)**: Tools like Debezium could tail the Postgres WAL,
  but this adds significant infrastructure complexity.
- **Transactional Outbox**: Write events and outbox rows in a single Postgres
  transaction, then poll the outbox and publish to RabbitMQ.

## Decision

We will use the **Transactional Outbox** pattern:

1. When handling a command, a single Postgres transaction writes:
   - The event(s) to the `events` table.
   - Corresponding row(s) to an `outbox` table with a `published` flag set to
     `false`.
2. A **poller** (background goroutine) periodically queries the `outbox` table
   for unpublished rows, publishes them to RabbitMQ, and marks them as
   `published = true`.
3. The poller uses `SELECT ... FOR UPDATE SKIP LOCKED` to allow concurrent
   pollers without contention.
4. Published outbox rows are periodically cleaned up (deleted) by a maintenance
   job.

## Consequences

**Benefits:**
- **Atomicity without distributed transactions**: the event and outbox row are
  written in a single Postgres transaction. Either both are written or neither.
- **Guaranteed delivery**: if the app crashes after writing, the outbox row
  persists and the poller will eventually publish it.
- **Simple implementation**: no external CDC tools, no WAL tailing, just SQL
  queries and a polling loop.
- **Exactly-once semantics** (with idempotent consumers): even if the poller
  publishes a message twice (crash between publish and marking as published),
  consumers handle it idempotently.

**Costs:**
- **Polling latency**: there is a small delay (configurable, e.g., 100-500ms)
  between writing and publishing. Acceptable for this application.
- **Outbox table maintenance**: published rows accumulate and must be cleaned up
  periodically.
- **Additional database load**: the poller runs periodic queries against the
  outbox table. Negligible at this scale (1-3 users).
