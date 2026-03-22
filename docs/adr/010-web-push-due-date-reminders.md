# ADR-010: Web Push Notifications for Due Date Reminders

**Status:** Planned

## Context

Users want background push notifications when tasks are due today, even when the
app tab is closed or the device is locked. Safari supports standard Web Push with
VAPID keys since macOS 16.1 (Oct 2022) and iOS 16.4 (Mar 2023) for installed PWAs.

The scope is **due date reminders only** — not sync events, task completions, or
other activity notifications. This keeps the notification volume low and high-value.

## Decision

Implement standard Web Push (VAPID) with a dedicated reminder worker that checks
for tasks due today on a schedule.

### Architecture

```
┌────────────────┐     ┌───────────────┐     ┌──────────────────┐
│ Reminder Worker│────▶│ push_subscr.  │────▶│ Push Service     │
│ (cron, 1h)     │     │ (Postgres)    │     │ (Apple/Mozilla)  │
│                │     └───────────────┘     └──────────────────┘
│ Query: tasks   │                                    │
│ due today      │                                    ▼
│ not completed  │                           ┌──────────────────┐
└────────────────┘                           │ Service Worker    │
                                             │ push event        │
                                             │ → showNotification│
                                             └──────────────────┘
```

### Key Design Choices

1. **Standalone timer worker, not RabbitMQ consumer.** There is no event that
   naturally triggers "send morning reminders" — it is time-based. A simple
   goroutine with a ticker is the simplest correct approach.

2. **Direct table writes for subscriptions.** Push subscriptions are infrastructure
   state (like `user_config`), not domain events. They do not need event sourcing,
   CRDT merge, or the sync engine.

3. **Idempotency via `reminder_log` table.** A `(user_id, sent_date)` primary key
   prevents duplicate notifications if the worker restarts or the interval fires
   twice in the same hour.

4. **Single aggregated notification per user.** One notification ("3 tasks due
   today: Buy groceries, Call dentist, Review PR") is better UX than spamming
   individual notifications per task.

5. **VAPID keys as env vars.** Generated once, stored in `.env`. Not
   auto-generated at runtime to avoid key rotation issues with existing
   subscriptions. Generate with:
   ```bash
   go run -e 'import webpush "github.com/SherClockHolmes/webpush-go"; priv, pub, _ := webpush.GenerateVAPIDKeys(); println("VAPID_PUBLIC_KEY=" + pub); println("VAPID_PRIVATE_KEY=" + priv)'
   ```
   Or use any VAPID key generator (e.g., `npx web-push generate-vapid-keys`).

### Components

| Component | Location | Purpose |
|-----------|----------|---------|
| Config | `api/internal/config/config.go` | VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, VAPID_SUBJECT |
| Migration | `api/migrations/006_push_subscriptions.sql` | push_subscriptions + reminder_log tables |
| Push handler | `api/internal/handler/push.go` | Subscribe/unsubscribe/VAPID key endpoints |
| VAPID keygen | `npx web-push generate-vapid-keys` | One-time setup (not in repo) |
| Service worker | `web/public/sw.js` | Push + notificationclick event handlers |
| Push utility | `web/src/push.ts` | PushManager subscribe/unsubscribe |
| Sidebar toggle | `web/src/components/layout/Sidebar.tsx` | Enable/disable notifications |
| Reminder worker | `api/cmd/worker-reminder/main.go` | Hourly check, send push for due tasks |

### API Endpoints

```
GET    /api/v1/push/vapid-key   → returns VAPID public key
POST   /api/v1/push/subscribe   → stores push subscription
DELETE /api/v1/push/subscribe   → removes push subscription
```

### Database Tables

```sql
push_subscriptions (
    id BIGSERIAL PK,
    user_id UUID REFERENCES users(id),
    endpoint TEXT UNIQUE,
    key_p256dh TEXT,
    key_auth TEXT,
    created_at TIMESTAMPTZ
)

reminder_log (
    user_id UUID,
    sent_date DATE,
    task_count INTEGER,
    sent_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, sent_date)
)
```

### Push Payload

```json
{
  "title": "DoIt: 3 tasks due today",
  "body": "Buy groceries, Call dentist, Review PR",
  "url": "/today"
}
```

Up to 3 task titles in body; if more, append "+ N more".

## Consequences

- Requires Safari 16.4+ on iOS (must be installed as PWA) or Safari 16.1+ on macOS
- Adds a new worker process to the deployment (`worker-reminder`)
- VAPID keys must be generated and configured before push works
- Stale subscriptions (410 Gone from push service) are automatically cleaned up
- No notification batching/rate limiting needed at 1-3 user scale

## Implementation

4 slices: Backend API → Service Worker handlers → Frontend UI → Reminder Worker.
See plan file for detailed breakdown.
