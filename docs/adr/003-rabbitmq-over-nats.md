# ADR-003: RabbitMQ Over NATS/Redis Streams

**Status:** Accepted

## Context

In Phase 3, DoIt introduces asynchronous event distribution from the event store
to projection workers and other consumers. We need a message broker that
supports:

- **Topic-based routing**: different consumers subscribe to different event types.
- **Dead-letter queues (DLQ)**: failed messages must be captured for inspection
  and retry rather than silently dropped.
- **Durable queues**: messages must survive broker restarts.
- **Mature tooling**: management UI, monitoring, well-documented operations.
- **Self-hosted simplicity**: easy to run via Docker Compose.

The main candidates considered were:

1. **RabbitMQ** — full-featured message broker with AMQP protocol.
2. **NATS** — lightweight, high-performance messaging system.
3. **Redis Streams** — stream data structure in Redis.

## Decision

We will use **RabbitMQ 3.13+** with topic exchanges and dead-letter queues.

- Events are published to a **topic exchange** with routing keys matching event
  types (e.g., `task.created`, `task.completed`).
- Each projection worker binds a durable queue with routing key patterns
  (e.g., `task.*` or `#` for all events).
- Failed messages are routed to a **dead-letter exchange/queue** for inspection
  and manual retry.
- RabbitMQ's management UI provides visibility into queue depths, consumer
  status, and message rates.

## Consequences

**Benefits:**
- Topic exchanges provide flexible routing without consumer-side filtering.
- Built-in DLQ support via dead-letter exchanges — no custom implementation.
- Management UI simplifies debugging and monitoring.
- Mature, well-documented, battle-tested in production systems.
- Easy to self-host via Docker with the `rabbitmq:3.13-management` image.
- Supports quorum queues for durability (useful even at small scale).

**Costs:**
- **Heavier than NATS** — more memory and CPU usage, but acceptable for a
  self-hosted single-server deployment.
- AMQP protocol is more complex than NATS's simpler text protocol.
- Requires Erlang runtime (handled by Docker image).
- Operational overhead of managing exchanges, queues, and bindings (mitigated
  by the management UI and declarative configuration at startup).
