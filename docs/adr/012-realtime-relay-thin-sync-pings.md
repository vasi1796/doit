# ADR-012: Realtime Event Delivery via API-Side Broker Relay with Thin Sync Pings

**Status:** Accepted

## Context

Events created by background workers (recurring-task occurrences, and any future
worker output) had no realtime path to clients. The only WebSocket broadcast
lived in the sync HTTP handler, so it covered client-pushed operations only;
worker-created events reached clients solely via the 30-second HTTP poll (up to
5 minutes under backoff). Completing a recurring task meant waiting up to 30s
for the next occurrence to appear.

Forces at play:

- ADR-003 (RabbitMQ) and ADR-004 (transactional outbox) already route every
  event through the `doit.events` topic exchange — new consumers subscribe
  without touching producers.
- ADR-009 established the client's idempotent, HLC-ordered merge and the poll
  as the convergence mechanism.
- The app should scale honestly to thousands of users without redesign.

## Decision

We will make the API a consumer of its own event stream and keep the WebSocket
channel pure signalling:

1. The API binds a **server-named, exclusive, auto-delete queue** to
   `doit.events` with routing key `#`. A relay goroutine consumes it and sends
   a constant **`{"type":"sync"}` ping** to the event's user via the in-process
   WS hub. No event payloads cross the WebSocket.
2. Clients respond to a ping by pulling through `POST /api/v1/sync` — the
   single state-transfer path. A sync requested while one is in flight sets a
   pending flag and runs exactly one trailing sync (client-side coalescing; no
   server-side state).
3. The sync handler's direct broadcast is removed — one delivery path for all
   events regardless of origin (HTTP push, recurring worker, reminder worker,
   future workers).

## Alternatives Considered

### Fat events over WebSocket (relay broadcasts full event payloads)
- **Description**: Relay maps `broker.EventMessage` → client event shape and
  pushes event arrays over WS, merged client-side.
- **Pros**: Lower latency (no pull round trip); matched the previous client code.
- **Cons**: Two state-transfer paths (WS payloads + sync pull) that must stay
  shape-compatible forever; WS-merged events bypass the sync cursor; a classic
  source of drift bugs.
- **Why rejected**: Thin pings concentrate all state transfer and cursor logic
  in one code path; the extra round trip is tens of milliseconds locally.

### Durable named queue (`doit.ws`)
- **Description**: Same relay, but consuming a durable queue declared in the
  shared topology.
- **Pros**: Events published during a brief API outage are still delivered on
  restart.
- **Cons**: Backlog accumulates while the API is down and replays to clients
  pointlessly (poll already covers gaps); queue grows unbounded if unconsumed.
- **Why rejected**: Durability duplicates the poll's guarantee at real
  operational cost. Best-effort semantics match what a notification channel is.

### Client-side follow-up sync after completing a recurring task
- **Description**: Schedule one extra sync ~1.5s after flushing a
  `CompleteTask` with a recurrence rule.
- **Pros**: ~10 lines, frontend only.
- **Cons**: A timing guess racing the worker pipeline; special-cases one event
  type; does nothing for other workers.
- **Why rejected**: Heuristic, not architecture.

## Consequences

### Positive
- Worker-created events reach all connected devices in well under 1s
  (measured: 204ms for a server-origin event to render; 816ms
  tick-to-next-occurrence including the client's 500ms push debounce).
- Any future worker gets realtime delivery for free — no per-feature client or
  server changes.
- WS remains best-effort by design; the 30s poll is the sole correctness
  mechanism, unchanged.
- Old cached PWA clients degrade gracefully: they ignore the ping object (they
  expect arrays) and converge via poll.
- Fixed a latent bug: `sync()` no longer silently drops requests arriving
  mid-flight.

### Negative
- Client-pushed events now take ~200ms longer (outbox poll) to echo to the
  user's other devices than the old direct broadcast — imperceptible in
  practice.
- Each API instance receives **all users' events** and filters in-process.
  Fine to thousands of users at task-app event rates. **Scaling ceiling**: past
  that, shard with per-user routing keys (`user.<id>.<type>`) and dynamic
  bindings per connected user, or move fan-out to a dedicated pub/sub tier.
  That is a new ADR.
- One more goroutine/AMQP consumer per API instance to reason about
  (mitigated: it shares the broker's existing reconnect machinery).

### Cross-Repo Impact
- None — single repo. Within the repo: `broker` (new `ConsumeBroadcast`),
  `handler` (hub `Notify`, new relay, sync handler slimmed), `web` sync engine
  (ping handling, coalescing). Workers unchanged.

## Implementation Notes

- Relay: `api/internal/handler/ws_relay.go`; re-subscribes on broker reconnect
  (via `Reconnected()`) or delivery-channel close, with capped backoff.
  Malformed messages are logged and skipped — never crash the loop.
- Auto-ack on the broadcast queue is deliberate: manual acks on a best-effort
  channel would be reliability theatre.
- Rollback: revert the relay wiring in `cmd/api/main.go` and restore the sync
  handler broadcast; no schema or contract changes.
