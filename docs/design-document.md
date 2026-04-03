# DoIt

**Personal Task Management Application**

**Software Design Document**
Version 2.0 | March 2026
Author: Andrei
FINAL

## 1. Introduction

DoIt is a personal, self-hosted task management application designed as a feature-complete alternative to TickTick/Things 3. The primary goal is twofold: to eliminate dependency on paywalled third-party todo applications, and to serve as a deep learning vehicle for distributed systems patterns including CRDTs, event sourcing, CQRS, and message-driven architectures.

The application targets a small user base of 1--3 people, authenticated via Google SSO, with full offline-first capability and real-time cross-device synchronisation. The target platform is exclusively the Apple ecosystem: macOS (via Safari "Add to Dock" web app), iOS (via Safari "Add to Home Screen"), and iPadOS. All platform-specific design decisions are made with Apple's WebKit engine and Safari PWA capabilities in mind.

### 1.1 Goals

- Replace TickTick/Things for daily personal task management across Mac, iPhone, and iPad
- Full offline-first operation -- the app must work identically with zero connectivity
- Real-time sync across iPhone, iPad, and Mac via PWA installed through Safari
- Event-sourced architecture with CRDT-based conflict resolution
- Message queue integration for async processing and observability
- Self-hosted on personal infrastructure with minimal running cost

### 1.2 Non-Goals

- App Store distribution (PWA via Safari eliminates this requirement)
- Multi-tenant SaaS -- this is a personal tool, not a product
- Native Swift/SwiftUI apps -- PWA provides sufficient experience for a todo app
- Android or Windows support -- Apple ecosystem only
- Complex permission models -- all authenticated users have full access

## 2. Feature Specification

Features are prioritised as P0 (MVP -- must ship in Phase 1), P1 (core -- Phase 2), and P2 (enhancement -- Phase 3+). The feature set is modelled after TickTick and Things 3, focusing on the daily workflow of capturing, organising, scheduling, and completing tasks.

### 2.1 Task Management (Core)

| Feature                    | Description                                                                                                                                                                                                                                                                                                                                                                                                              | Priority |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Create Task                | Quick-add task with title. Optionally set description, due date, priority, list, and labels at creation time.                                                                                                                                                                                                                                                                                                            | P0       |
| Edit Task                  | Modify any task field inline. Changes are captured as discrete events for the event store.                                                                                                                                                                                                                                                                                                                               | P0       |
| Complete Task              | Mark task as complete. Task moves to Completed view. Reversible (uncomplete).                                                                                                                                                                                                                                                                                                                                            | P0       |
| Delete Task                | Soft-delete to Trash. Tasks remain in Trash for 30 days before permanent deletion by a background worker.                                                                                                                                                                                                                                                                                                                | P0       |
| Task Priority              | Four levels: None, Low, Medium, High. Visual indicator (colour/icon) in all views.                                                                                                                                                                                                                                                                                                                                      | P0       |
| Task Description           | Obsidian-style live preview markdown editor (CodeMirror 6). Type markdown syntax directly — rendered inline as you type. Supports headings, bold, italic, strikethrough, inline code, code blocks, bullet/numbered lists, links, and tables. Raw markdown string stored in CRDT for sync (LWW-Register, whole-string replacement per ADR-006).                                                                            | P1       |
| Subtasks / Checklist       | Nested checklist items within a task. Each subtask has its own completion state. Progress shown as fraction (e.g. 2/5).                                                                                                                                                                                                                                                                                                  | P1       |
| Inline Markdown (Titles)   | Lightweight inline markdown rendering in task titles within list views. Supports bold, italic, strikethrough, and inline code. Rendered live -- no raw asterisks visible in the task list. Title remains single-line and compact.                                                                                                                                                                                         | P1       |
| Global Quick-Add           | Floating action button visible on all screens (not just Inbox). Pre-fills context from current view: List page → that list, Today → today's due date, Label page → that label. Modelled after TickTick's global add button.                                                                                                                                                                                              | P1       |
| ~~Task Attachments~~       | ~~Attach images or small files to tasks. Stored in object storage (S3-compatible). Size limit per attachment: 10MB.~~                                                                                                                                                                                                                                                                                                    | Dropped  |

### 2.2 Organisation & Structure

| Feature        | Description                                                                                                                          | Priority |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Lists          | Group tasks into named lists (e.g. Work, Personal, Shopping). Each list has a colour and icon. Tasks belong to exactly one list.      | P0       |
| Labels / Tags  | Cross-cutting tags applied to tasks across any list. Many-to-many relationship. Filterable in all views.                              | P0       |
| Smart Lists    | System-generated views: Inbox (no list assigned), Today, Upcoming (next 7 days), Someday (no date).                                  | P0       |
| ~~List Folders~~   | ~~Group related lists into collapsible folders for sidebar organisation.~~                                                            | Dropped  |
| Task Ordering  | Manual drag-and-drop reordering within lists. Position tracked via fractional indexing CRDT.                                          | P1       |

### 2.3 Scheduling & Time

| Feature         | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                | Priority |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| Due Date        | Assign a due date to any task. Overdue tasks visually highlighted in red across all views.                                                                                                                                                                                                                                                                                                                                                                                                 | P0       |
| Due Time        | Optional time component on due date for time-specific tasks.                                                                                                                                                                                                                                                                                                                                                                                                                               | P1       |
| Recurring Tasks | Repeating schedules: daily, weekly, monthly, yearly, custom intervals. On completion, next occurrence auto-generated by background worker.                                                                                                                                                                                                                                                                                                                                                 | P1       |
| Reminders       | Push notifications via Web Push API. Supported on iOS 16.4+ for home-screen-installed PWAs and on macOS Sonoma+ Safari web apps. Requires explicit user permission grant via a user-initiated gesture. Reliability on iOS is lower than native -- treat as best-effort, not guaranteed delivery.                                                                                                                                                                                            | P2       |
| Calendar View   | Monthly calendar view showing tasks by due date. Click a day to see/add tasks. Drag tasks between days to reschedule.                                                                                                                                                                                                                                                                                                                                                                     | P1       |
| iCal Feed       | Read-only iCalendar (.ics) feed served at `/cal/feed.ics`. Generates VEVENT entries for tasks with due dates, subscribable in Apple Calendar and Google Calendar so tasks appear alongside regular events. Authenticated via per-user token in the URL (calendar apps do not support OAuth for feed subscriptions). Rebuilt on task changes by a background worker. Uses go-ical (`github.com/emersion/go-ical`).                                                                            | P1       |
| ~~Start Date~~  | ~~Optional start date -- task hidden from Today/Upcoming views until start date arrives.~~                                                                                                                                                                                                                                                                                                                                                                                                 | Dropped  |

### 2.4 Views & Navigation

| Feature           | Description                                                                                    | Priority |
| ----------------- | ---------------------------------------------------------------------------------------------- | -------- |
| Inbox             | Default capture location. Tasks with no list assigned land here for later triage.              | P0       |
| Today View        | All tasks due today plus overdue tasks (shown in a separate "Overdue" section above today's tasks). Primary daily working view. | P0       |
| Upcoming View     | Tasks due in the next 7 days, grouped by day.                                                  | P0       |
| List View         | All tasks within a specific list, with filters and sort options.                               | P0       |
| Completed View    | Archive of completed tasks, grouped by completion date. Searchable.                            | P0       |
| Trash View        | Soft-deleted tasks. Restore or permanently delete. Auto-purge after 30 days.                   | P0       |
| Label Filter View | View all tasks matching one or more selected labels, across all lists.                         | P1       |
| Search            | Full-text search across task titles, descriptions, and labels.                                 | P1       |
| Eisenhower Matrix | Four-quadrant view using priority (importance) and due date (urgency). Tasks with due date today/overdue are urgent. Accessible from bottom nav. Frontend-only — no new fields needed. | P2       |
| Filtering & Sorting | Filter tasks by priority, due date, labels, and completion status. Sort by due date, priority, title, or creation date. Sort preference is persistent per view (stored in IndexedDB and synced via user config). Available on List, Inbox, Label, and Today views. | P1       |

### 2.5 Sync & Offline

Running exclusively within the Apple ecosystem, all PWA capabilities are governed by Apple's WebKit engine and Safari's feature set. This directly shapes the sync architecture. What works well: Service Workers are fully functional for caching and network interception on all Apple platforms; IndexedDB is available and reliable (home-screen-installed PWAs get up to ~60% of device disk); push notifications are supported on iOS 16.4+ (home-screen PWAs only) and macOS Sonoma+ via Web Push API, with Safari 18.4 adding Declarative Web Push as a simpler alternative; standalone mode runs full-screen without browser chrome on all platforms (macOS: Dock and Cmd+Tab; iOS: App Library and Spotlight); the `prefers-color-scheme` media query is fully supported for automatic dark mode; Badge API is supported since iOS 16.4 for notification count badges on the home screen icon. Key constraints: Safari does not implement the Background Sync API -- all sync must occur while the app is in the foreground; IndexedDB and Cache Storage may be evicted if the PWA is unused for several weeks (7-day script-writable storage cap for Safari browsing, more lenient for home-screen PWAs); Cache API is capped at ~50MB per origin (use IndexedDB for task data, Cache API only for app shell); Safari never shows an automatic install prompt -- users must manually use Share -> Add to Home Screen (iOS) or File -> Add to Dock (macOS); iOS push notifications are less reliable than native APNs and may be delayed or missed when the device has been idle; PWAs cannot create home screen widgets, Siri Shortcuts, or Apple Watch apps; data stored while browsing in Safari is not shared with the installed PWA instance and vice versa -- the OAuth flow must redirect back to the standalone PWA context. The CRDT + server-sync design below works within these constraints.

| Feature            | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | Priority |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| Offline-First      | All reads/writes go to local IndexedDB. App is fully functional with zero connectivity. Service worker caches app shell. On iOS/iPadOS, PWA must be installed to home screen for reliable storage persistence.                                                                                                                                                                                                                                                                                                        | P0       |
| Background Sync    | Safari does not support the Background Sync API. Sync is triggered on app foreground (`visibilitychange` event) and on a polling interval when the app is active (base 30s). On consecutive server failures, polling backs off exponentially (30s -> 60s -> 120s, capped at 5 minutes) with random jitter (+/-5s) to prevent synchronised retries across devices. Queued CRDT operations sync automatically when connectivity is restored.                                                                            | P0       |
| Real-Time Push     | WebSocket connection pushes changes from other devices in real-time when online. On disconnect (server restart, network change, iOS backgrounding): immediately fall back to polling, attempt WebSocket reconnection with exponential backoff (1s -> 2s -> 4s, capped at 30s) plus jitter, and on successful reconnection perform a full sync pull to catch events missed during the disconnect window.                                                                                                                | P1       |
| Conflict Resolution| CRDT-based merge: LWW-Register for scalar fields, OR-Set for labels/list membership, fractional index for ordering.                                                                                                                                                                                                                                                                                                                                                                                                  | P1       |
| Sync Status        | Visual indicator showing sync state: synced, syncing, offline, conflict.                                                                                                                                                                                                                                                                                                                                                                                                                                             | P1       |
| Server Backup Sync | Per-aggregate server-side snapshots as insurance against Safari storage eviction. On each sync, only changed aggregates (tasks, lists, labels, user config) are snapshot to a server-side table keyed by `aggregate_id` + `user_id`. Client tracks `last_synced_version` per aggregate for incremental pull. If local IndexedDB is evicted, all snapshots for the user are pulled and IndexedDB is rehydrated on next launch.                                                                                          | P0       |

### 2.6 Authentication & Users

| Feature         | Description                                                                                                               | Priority |
| --------------- | ------------------------------------------------------------------------------------------------------------------------- | -------- |
| Google SSO      | Login via Google OAuth 2.0. No password management. Uses `golang.org/x/oauth2`.                                           | P0       |
| User Allowlist  | Hardcoded list of permitted Google email addresses. Unauthenticated/unlisted users rejected.                               | P0       |
| JWT Sessions    | Server issues JWT on successful OAuth. PWA stores token in memory (not localStorage). Refresh token rotation.              | P0       |
| Per-User Data   | All tasks, lists, and labels are scoped to the authenticated user. No shared lists in MVP.                                 | P0       |

### 2.7 UX & PWA

The PWA targets Safari exclusively across all Apple platforms. Installation paths differ per platform:

- **macOS (Sonoma 14+):** Safari -> File -> Add to Dock. App gets its own Dock icon, runs in standalone window with no browser chrome, appears in Cmd+Tab and Spotlight.
- **iOS (16.4+) / iPadOS:** Safari -> Share -> Add to Home Screen. From iOS 26 onwards, the "Open as Web App" toggle defaults to on for all sites. App runs full-screen with its own icon, appears in App Library and Spotlight.
- **Important:** All iOS browsers use WebKit under the hood, so Safari is the only engine that matters. Chrome/Firefox/Edge on iOS offer no additional PWA capabilities.

| Feature             | Description                                                                                                                                                                                                                    | Priority |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- |
| PWA Install         | Installable via Safari on macOS (Add to Dock), iOS and iPadOS (Add to Home Screen). In-app guidance banner to walk users through the installation process since Safari has no automatic install prompt.                         | P0       |
| Responsive Layout   | Sidebar navigation on Mac/iPad landscape. Bottom tab bar on iPhone and iPad portrait. Fluid transitions. Designed for Safari WebKit rendering.                                                                                 | P0       |
| Dark Mode           | System-preference detection via `prefers-color-scheme` media query. Matches macOS/iOS system-wide dark mode setting. Manual toggle persisted per device.                                                                       | P1       |
| ~~Keyboard Shortcuts~~  | ~~Quick-add (Cmd+N), search (Cmd+K), navigate views. Primarily for Mac desktop productivity.~~ Cmd+N and Cmd+K are implemented; extended shortcuts dropped.                                                                   | Dropped  |
| Drag & Drop         | Reorder tasks within lists, move tasks between lists, reschedule on calendar by dragging. Touch-friendly long-press drag on iOS/iPadOS.                                                                                        | P1       |
| iPad Multitasking   | Support for Split View and Slide Over on iPadOS. PWA adapts layout to reduced window sizes.                                                                                                                                    | P2       |

## 3. Architecture Overview

The system follows an event-sourced, CQRS architecture with offline-first clients. Every state mutation is captured as an immutable event. Read models are projected from the event stream. CRDTs handle conflict-free merging of concurrent offline edits. The architecture is specifically designed around Safari/WebKit's capabilities and limitations on Apple platforms.

### 3.1 Backend (Go)

**Language:** Go 1.22+
**Router:** Chi (lightweight, idiomatic, middleware-friendly)
**Database:** PostgreSQL 16 (event store + read models). Connection pooling via Go `database/sql`: API gets `SetMaxOpenConns(10)`, each worker gets 3, outbox poller gets 3. `SetConnMaxLifetime(5m)` to recycle connections.
**Message Queue:** RabbitMQ 3.13+ (event fan-out, async workers, dead-letter queues, auto-reconnect with exponential backoff)
**Auth:** Google OAuth 2.0 -> JWT (`golang.org/x/oauth2`, `golang-jwt/jwt`)
**WebSockets:** `gorilla/websocket` or `nhooyr.io/websocket` for real-time push
**Deployment:** Single binary + Docker Compose. Phase 1: Go app + Postgres + Caddy (synchronous in-process projections, no message queue). Phase 3: adds RabbitMQ + Transactional Outbox + workers.

The backend exposes a REST API for CRUD operations and a WebSocket endpoint for real-time sync. All mutations are written to the event store first, then projected to read models. In Phase 1, projections are updated synchronously in-process. From Phase 3 onwards, RabbitMQ distributes events via topic exchanges to workers for recurring task generation, trash auto-purge, and notification delivery. RabbitMQ's mature routing model (topic exchanges, dead-letter queues, TTLs) provides more expressive event routing than lighter alternatives, and its operational characteristics are well understood. Graceful shutdown: on SIGTERM (deployment/restart), the Go API stops accepting new connections, drains in-flight HTTP requests (15s timeout), closes WebSocket connections with a close frame so clients reconnect, flushes pending outbox publications (Phase 3+), then exits. Implemented via `signal.NotifyContext` and `http.Server.Shutdown`.

### 3.2 Frontend (PWA -- Safari/WebKit)

**Framework:** React 18+ with TypeScript
**State:** Dexie.js with `useLiveQuery` hook. IndexedDB is the single source of truth on the client -- no separate state management layer (no Redux, no Zustand). Dexie wraps IndexedDB with a clean API; `useLiveQuery` provides reactive queries that re-render React components automatically when underlying data changes, whether from user actions or sync engine merges. Unidirectional flow: user action -> write to IndexedDB -> Dexie live query fires -> React re-renders.
**Markdown Editor:** TipTap 2.x (built on ProseMirror). Live WYSIWYG markdown editing for task descriptions. Serialises to/from markdown strings for CRDT storage. Extensions: StarterKit, CodeBlockLowlight (syntax highlighting), TaskList, Table, Link.
**Sync Engine:** Custom CRDT sync layer -- sync-on-foreground via `visibilitychange` event + WebSocket for real-time push when active
**Service Worker:** Workbox for app shell caching. No Background Sync (unsupported by Safari) -- sync is triggered on app foreground and on polling interval.
**Styling:** Tailwind CSS with dark mode via `prefers-color-scheme`. Touch-friendly sizing for iOS (44px minimum tap targets per Apple HIG).
**Build:** Vite
**Target Engine:** Safari/WebKit exclusively. CSS and JS must be tested against WebKit. No reliance on Chromium-only APIs.

The PWA operates as an offline-first client targeting Safari on macOS, iOS, and iPadOS. All user interactions write to IndexedDB immediately via Dexie.js, which triggers `useLiveQuery` re-renders automatically. The sync engine batches CRDT operations and sends them to the server when online. Failed sync operations are retained in the queue with a retry count (max 5 retries) before being discarded. Incoming changes from other devices arrive via WebSocket and are merged into the local CRDT state using per-field LWW timestamps. Because Safari lacks Background Sync, the app aggressively syncs on every foreground event and maintains a polling interval (base 30s, exponential backoff with jitter on failure) when active. Per-aggregate snapshots are updated server-side on every sync as insurance against Safari storage eviction.

### 3.3 Data Flow

The data flow evolves across phases. In Phase 1, event processing is synchronous and in-process (no message queue). From Phase 3 onwards, RabbitMQ and the Transactional Outbox decouple event ingestion from processing. The full Phase 3+ flow is:

- User action (e.g. create task) writes a CRDT operation to local IndexedDB
- UI updates immediately from local state (optimistic)
- Sync engine queues the operation for server delivery
- On foreground event (`visibilitychange`) or polling tick, operation is sent via REST POST to `/api/v1/sync`
- Server validates, appends event to PostgreSQL event store
- Server publishes event to RabbitMQ via the Transactional Outbox (event + outbox row written in same PostgreSQL transaction; poller publishes to RabbitMQ)
- Server broadcasts event via WebSocket to all other connected devices for that user
- Receiving devices merge incoming CRDT state into their local IndexedDB
- Server updates per-aggregate snapshots for changed aggregates as insurance against Safari storage eviction

## 4. Data Model

The data model comprises two layers: the event store (append-only source of truth) and the projected read models (materialised views for fast querying).

### 4.1 Event Store Schema

All mutations are stored as immutable events. The event store is the single source of truth.

| Column         | Type          | Description                                                  |
| -------------- | ------------- | ------------------------------------------------------------ |
| id             | UUID          | Unique event identifier                                      |
| aggregate_id   | UUID          | The task/list/label this event belongs to                    |
| aggregate_type | VARCHAR       | `task` \| `list` \| `label` \| `user`                       |
| event_type     | VARCHAR       | e.g. `TaskCreated`, `TaskCompleted`, `LabelAdded`            |
| user_id        | UUID          | User who triggered the event                                 |
| data           | JSONB         | Event payload (CRDT operation, field values, metadata)       |
| timestamp      | TIMESTAMPTZ   | Hybrid logical clock timestamp for causal ordering           |
| version        | INTEGER       | Monotonic version per aggregate for optimistic concurrency   |

### 4.2 Read Model Entities

Projected from the event stream. These are disposable and can be rebuilt from events at any time.

- **users** -- `id`, `google_id`, `email`, `name`, `avatar_url`, `allowed` (boolean), `created_at`
- **lists** -- `id`, `user_id`, `name`, `colour`, `icon`, `position` (fractional index), `created_at`, `updated_at`
- **tasks** -- `id`, `user_id`, `list_id`, `title` (inline markdown string), `description` (full markdown string via TipTap), `priority` (0--3), `due_date`, `due_time`, `recurrence_rule`, `position` (fractional index), `is_completed`, `completed_at`, `is_deleted`, `deleted_at`, `created_at`, `updated_at`
- **labels** -- `id`, `user_id`, `name`, `colour`, `created_at`
- **task_labels** -- `task_id`, `label_id` (join table, OR-Set CRDT)
- **subtasks** -- `id`, `task_id`, `title`, `is_completed`, `position`, `created_at`
- **user_config** -- `id`, `user_id`, `theme` (light/dark/system), `sidebar_collapsed`, `default_list_id`, `updated_at` (separate aggregate for per-device and per-user preferences)
- **aggregate_snapshots** -- `aggregate_id`, `aggregate_type`, `user_id`, `data` (JSONB, materialised state of the aggregate), `version` (matches event store version), `updated_at`. Used for client rehydration after Safari storage eviction. Incrementally updated on sync -- only changed aggregates are written.

### 4.3 CRDT Strategy

- **Scalar fields** (title, description, priority, due_date): **LWW-Register** -- last writer wins based on per-field hybrid logical clock timestamps. Each scalar field tracks its own HLC timestamp independently, so concurrent edits to different fields on the same task are both preserved (e.g., Device A edits title while Device B edits due_date -- both edits survive). Title and description store raw markdown strings; the entire string is the CRDT unit (no character-level merging). This is appropriate for 1--3 users where simultaneous edits to the same field are rare.
- **Set fields** (labels on a task): **OR-Set** (Observed-Remove Set) -- concurrent add and remove resolve without conflict
- **Ordering** (task position within list): **Fractional Indexing** -- positions are strings that sort lexicographically, allowing insertions between any two items without reindexing
- **Completed/deleted state**: **LWW-Flag** -- boolean with timestamp, last toggle wins

### 4.4 Application-Level Conflict Policies

CRDTs guarantee convergence but not always sensible outcomes. The following application-level policies resolve edge cases where raw CRDT merging produces unintuitive results:

- **Edit resurrects concurrent delete:** If Device A soft-deletes a task while Device B concurrently edits a non-deleted field (title, description, labels, etc.), the edit wins and the task is restored from trash. Rationale: editing a task implies intent for it to exist. Implemented by comparing the HLC timestamps of the delete event and the edit event -- if they are concurrent (neither causally depends on the other), the edit takes priority.
- **Concurrent list moves:** If Device A moves a task from List X to List Y while Device B moves the same task from List X to List Z, the `list_id` LWW-Register resolves to whichever write has the later HLC timestamp. This is a known limitation of LWW -- one user's move is silently lost. Acceptable for 1--3 users where this scenario is extremely unlikely. The sync status UI should surface when a field was overwritten by a remote change so the user can verify.
- **Complete resurrects concurrent delete:** Same policy as edit: completing a task that was concurrently deleted restores it in a completed state. The user intended to mark it done, not destroy it.

## 5. API Design

RESTful API with JSON payloads. All `/api/*` endpoints require JWT authentication. Auth, health, metrics, and iCal feed endpoints use their own authentication mechanisms as noted below. The sync endpoint handles bidirectional CRDT state exchange.

| Method | Endpoint                            | Description                                                          |
| ------ | ----------------------------------- | -------------------------------------------------------------------- |
| GET    | `/auth/google`                      | Initiate Google OAuth flow                                           |
| GET    | `/auth/callback`                    | OAuth callback -> issue JWT                                          |
| POST   | `/api/v1/sync`                      | Push local CRDT ops, receive remote ops since last sync              |
| WS     | `/api/v1/ws`                        | WebSocket for real-time event push                                   |
| GET    | `/api/v1/tasks`                     | List tasks (filterable by list, label, date range, status)           |
| POST   | `/api/v1/tasks`                     | Create task (generates `TaskCreated` event)                          |
| PATCH  | `/api/v1/tasks/:id`                 | Update task fields (generates field-specific events)                 |
| DELETE | `/api/v1/tasks/:id`                 | Soft-delete to trash (generates `TaskDeleted` event)                 |
| GET    | `/api/v1/lists`                     | List all user lists                                                  |
| POST   | `/api/v1/lists`                     | Create list                                                          |
| PATCH  | `/api/v1/lists/:id`                 | Update list                                                          |
| DELETE | `/api/v1/lists/:id`                 | Delete list (moves contained tasks to Inbox)                         |
| GET    | `/api/v1/labels`                    | List all user labels                                                 |
| POST   | `/api/v1/labels`                    | Create label                                                         |
| PATCH  | `/api/v1/labels/:id`                | Update label (rename, recolour)                                      |
| DELETE | `/api/v1/labels/:id`                | Delete label (removes from all tasks via OR-Set)                     |
| GET    | `/api/v1/tasks/:id/subtasks`        | List subtasks for a task                                             |
| POST   | `/api/v1/tasks/:id/subtasks`        | Create subtask (generates `SubtaskCreated` event)                    |
| PATCH  | `/api/v1/tasks/:id/subtasks/:sid`   | Update subtask (toggle completion, edit title, reorder)              |
| DELETE | `/api/v1/tasks/:id/subtasks/:sid`   | Delete subtask (generates `SubtaskDeleted` event)                    |
| GET    | `/healthz`                          | Liveness probe: process up, PostgreSQL reachable, RabbitMQ reachable (Phase 3+). Used by Docker health checks. No auth. |
| GET    | `/metrics`                          | Prometheus metrics endpoint. Scraped by Prometheus for Grafana dashboards. No auth. |
| GET    | `/cal/:token/feed.ics`              | iCalendar feed of tasks with due dates (token-authenticated, no JWT) |

## 6. Infrastructure & Deployment

The entire stack runs on a single VPS using Docker Compose. Caddy handles TLS termination and reverse proxying.

### 6.1 Docker Compose Services

- **doit-api** -- Go binary, exposes REST + WebSocket on `:8080`
- **postgres** -- PostgreSQL 16, persistent volume, event store + read models
- **rabbitmq** -- RabbitMQ 3.13+ with management plugin, persistent queues, topic exchanges for event distribution (Phase 3+)
- **caddy** -- Reverse proxy with automatic Let's Encrypt TLS. Serves PWA static assets and proxies `/api/*` to Go backend
- **worker-recurring** -- Go binary, consumes from RabbitMQ, generates next occurrences for recurring tasks (Phase 3+)
- **worker-cleanup** -- Go binary with internal `time.Ticker` (runs daily), permanently deletes tasks in trash older than 30 days (Phase 3+)

### 6.2 Estimated Running Cost

- Hetzner CX22 (2 vCPU, 4GB RAM, 40GB SSD): EUR 4.35/month
- Domain name: ~GBP 10/year
- Backups (Hetzner automated): EUR 0.87/month

**Total: ~GBP 6/month**

### 6.3 Monitoring & Observability

Prometheus metrics exported from the Go backend and RabbitMQ (via the built-in Prometheus plugin). Grafana dashboard for the four golden signals: latency, traffic, errors, and saturation. RabbitMQ-specific metrics: queue depth, consumer utilisation, message rates, and dead-letter queue size. Structured JSON logging with zerolog. This connects directly to the observability curriculum from the home lab setup.

### 6.4 Backup & Disaster Recovery

The PostgreSQL event store is the single source of truth -- all read models, snapshots, and derived data can be rebuilt from it, but losing the event store means losing everything. Backup strategy: daily `pg_dump` of the PostgreSQL database via a cron job on the VPS, compressed and uploaded to a Hetzner Storage Box (or equivalent off-VPS storage). Retain 7 daily backups and 4 weekly backups. Hetzner's automated VPS snapshots provide an additional layer but should not be the sole backup since they are tied to the same provider. Test the restore path at least once before going live: drop the database, restore from `pg_dump`, run the projection rebuilder, and verify all data is intact.

## 7. LLM Agent Development Harness

Modern software development increasingly involves LLM-based coding agents (Claude Code, Codex) that operate directly on the codebase. These agents are significantly more effective when the repository contains structured context files that explain the project's architecture, conventions, and constraints. Setting this up from day one means every agent interaction -- whether generating a new worker, debugging sync logic, or writing tests -- starts with the right context.

### 7.1 Repository Context Files

The following files should live in the repository root and be maintained as living documents alongside the code:

**AGENTS.md:** The primary context file for coding agents. This should document: the project's purpose and architecture at a high level, the tech stack and key dependencies (Go, Chi, PostgreSQL, RabbitMQ, React/TipTap, Workbox), code conventions (naming, error handling patterns, file structure), how to run the project locally (`docker compose up`, test commands, seed data), the event sourcing model (how events flow from API -> event store -> outbox -> RabbitMQ -> workers/projections), CRDT conventions (which fields use which CRDT type), and any gotchas or constraints (Safari-only PWA, no Background Sync, token auth for iCal feed). Keep this under 500 lines -- agents work best with focused, dense context rather than exhaustive documentation.

**CLAUDE.md:** Claude Code-specific configuration. Automatically read by Claude Code on session start. Use this for: preferred coding style (e.g. "use table-driven tests in Go", "handle errors explicitly, no panic"), commit message format, which directories contain what (`cmd/` for binaries, `internal/` for packages, `web/` for frontend), and any project-specific rules (e.g. "all state mutations must go through the event store, never update read models directly").

**Architecture Decision Records (`docs/adr/`):** Short markdown files documenting key decisions and their reasoning (e.g. "ADR-001: Why RabbitMQ over NATS", "ADR-002: LWW-Register for markdown fields", "ADR-003: Transactional Outbox pattern"). Agents can reference these to understand why the codebase is structured a certain way, preventing them from suggesting architecturally incompatible changes. Also valuable for your own future reference.

### 7.2 Agent-Friendly Project Structure

Beyond context files, certain structural practices make the codebase more navigable for agents:

- Clear package boundaries with `README.md` files in key directories explaining that package's responsibility and public API. An agent navigating `internal/eventstore/` should immediately understand what it does without reading every file.
- Consistent naming conventions for event types, handlers, and projections. If every event handler follows the same pattern (e.g. `HandleTaskCreated`, `HandleTaskCompleted`), agents can generate new handlers by analogy.
- Comprehensive test examples. Agents generate better tests when they can see existing test patterns. Include at least one well-documented table-driven test, one integration test that exercises the event store -> projection flow, and one CRDT merge test.
- A Makefile or Taskfile with common commands (`make test`, `make run`, `make rebuild-projections`, `make lint`). Agents can invoke these to verify their changes work.
- Typed interfaces over concrete implementations. Go interfaces at package boundaries make it easier for agents to understand contracts without diving into implementation details, and enable them to generate correct mock implementations for testing.

### 7.3 Why This Matters

The investment is small -- a few hundred lines of markdown maintained alongside the code. The payoff is that every agent interaction starts with accurate architectural context rather than hallucinated assumptions. For a project with event sourcing, CRDTs, and a transactional outbox, this is especially important: these are patterns that agents frequently get wrong without explicit guidance. An `AGENTS.md` that says "all state mutations must flow through the event store, never write to read model tables directly" prevents an entire class of agent-generated bugs. ADRs that explain why you chose LWW-Register over operational transforms for markdown fields prevent agents from suggesting incompatible approaches. This is not overhead -- it's the same documentation you'd want for a collaborator, structured for the collaborators you actually have.

## 8. Implementation Plan

Each phase combines feature delivery with structured learning from *Designing Data-Intensive Applications* (DDIA) by Martin Kleppmann. The reading is sequenced so that you study the relevant theory immediately before implementing it. Each phase specifies what to read, what to build, and what you should understand by the end.

### Phase 1: Foundation (MVP) -- Online-Only

**Duration:** 2--3 weeks
**Goal:** A usable online-only todo app with core features, built on an event-sourced foundation.

**Read first (DDIA):** Chapter 2 (Data Models and Query Languages) -- sections on relational vs document models and how data relationships are expressed. Chapter 3 (Storage and Retrieval) -- focus on "Data Structures That Power Your Database" covering log-structured storage and append-only logs. Skim the B-tree section for contrast, but focus on the log-structured approach since your event store is an append-only log with indexes. Chapter 4 (Encoding and Evolution) -- pay attention to "Formats for Encoding Data" (JSON, Protobuf, Avro) and "Modes of Dataflow." The key concept is forward and backward compatibility for your event schemas.

**What to build:**

- Go backend: Chi router, PostgreSQL, event store (append-only events table), synchronous in-process projections (no message queue yet), JWT auth, Google SSO with email allowlist
- REST API: full CRUD for tasks, lists, labels -- all mutations written as events first, then projected to read models
- Versioned event schema system: each event type carries a schema version with forward/backward-compatible decoders
- PWA frontend: Inbox, Today, Upcoming, List, Completed, Trash views
- Basic responsive layout (sidebar on Mac/iPad, bottom tabs on iPhone)
- Plain text description field for tasks (TipTap markdown editor deferred to Phase 2, aligning with P1 priority)
- Docker Compose deployment with Caddy for automatic TLS

**Done when:** You use DoIt as your daily todo app for one full week, replacing TickTick entirely. All P0 features work reliably on Mac and iPhone. You can create, complete, and organise tasks without reaching for TickTick.

**You should understand:** Why separating the write model (event log) from the read model (projections) is powerful. The tradeoff between write amplification and read performance. Why an append-only log is simpler than in-place mutation for audit trails. Why event stores make schema evolution both easier (no data mutation) and harder (every historical version must be handled forever). The difference between schema-on-write and schema-on-read.

### Phase 2: Offline-First & CRDT Sync

**Duration:** 3--4 weeks
**Goal:** Full offline-first operation with CRDT-based sync across Mac, iPhone, and iPad.

**Read first (DDIA):** Chapter 5 (Replication) -- read the full chapter, focusing on: "Multi-Leader Replication" (your devices are effectively multiple leaders accepting writes independently), "Handling Write Conflicts" (LWW, merge functions, CRDTs), "Leaderless Replication" (quorum concepts, read-your-own-writes guarantees), and "Detecting Concurrent Writes" (version vectors). Also read Chapter 9 (Consistency and Consensus) -- focus on "Ordering Guarantees" (total order vs causal order), "Linearizability" (what it means and why your system deliberately doesn't provide it), and "Sequence Number Ordering" (Lamport timestamps, vector clocks). Supplement with Kleppmann's "CRDTs: The Hard Parts" talk and James Long's "CRDTs for Mortals."

**What to build:**

- IndexedDB local storage with Dexie.js for offline-first data
- Service worker for app shell caching (Workbox) -- no Background Sync (unsupported by Safari)
- CRDT implementation: LWW-Register for scalar fields (start here, simplest), OR-Set for labels, fractional indexing for task ordering
- Hybrid Logical Clock (HLC) implementation in Go for causal event ordering instead of wall-clock timestamps
- Sync engine: sync-on-foreground via `visibilitychange` event, batch operations, conflict resolution, sync status UI
- WebSocket real-time push for multi-device updates when online
- Server-side per-aggregate snapshots updated incrementally on sync as insurance against Safari storage eviction
- Test harness simulating two devices making conflicting offline edits, verifying correct merge
- Read-your-own-writes consistency: UI always reflects local writes before server confirmation
- TipTap markdown editor for task descriptions (P1 feature, raw markdown string stored as CRDT LWW-Register)

**Done when:** You toggle airplane mode on your iPhone, create and complete several tasks, reconnect, and see them appear on your Mac within 30 seconds. You do the same in reverse. The CRDT test harness passes all conflict scenarios. The app launches instantly from the home screen with zero connectivity.

**You should understand:** Why multi-device sync is fundamentally a multi-leader replication problem. The difference between state-based and operation-based CRDTs. Why LWW trades correctness (silent data loss on concurrent edits) for simplicity, and when that tradeoff is acceptable. The CAP theorem: your system chooses availability (offline writes always succeed) over consistency (devices may temporarily diverge). Why linearizability is too expensive for offline-first systems. Why causal consistency is the strongest model you can achieve while remaining available during partitions. The relationship between Lamport timestamps, vector clocks, and HLCs.

### Phase 3: Message Queue, Workers & Observability

**Duration:** 2--3 weeks
**Goal:** Async event processing, reliable delivery, recurring tasks, and production observability.

**Read first (DDIA):** Chapter 7 (Transactions) -- focus on "The Meaning of ACID" (atomicity, isolation), "Weak Isolation Levels" (read committed, snapshot isolation), and multi-object transactions. Then Chapter 11 (Stream Processing) -- "Messaging Systems" (direct vs broker, exactly-once semantics), "Uses of Stream Processing" (event sourcing, change data capture), "Reasoning About Time" (event time vs processing time), and the "Exactly-once message processing" section which motivates the Transactional Outbox pattern. Chapter 8 (The Trouble with Distributed Systems) -- "Unreliable Networks" (timeouts, retries), "Unreliable Clocks" (NTP drift, logical clocks), and "Knowledge, Truth, and Lies." Chapter 12 (The Future of Data Systems) -- "Unbundling Databases" (the philosophical foundation of your architecture: the event log as source of truth, everything else as derived views).

**What to build:**

- Transactional Outbox pattern: within a single PostgreSQL transaction, append events to the event store and write to an outbox table. A separate poller publishes outbox rows to RabbitMQ with publisher confirms enabled, then marks rows as published only after RabbitMQ confirms receipt. If the poller crashes, it restarts and picks up unpublished rows. Rows in-flight for longer than 60 seconds are retried. This guarantees exactly-once delivery without distributed transactions.
- RabbitMQ integration with topic exchanges for event fan-out and dead-letter queues for failed processing
- Idempotent event handlers in all workers -- safe for at-least-once redelivery
- Recurring task worker (generates next occurrence on completion)
- Trash auto-purge worker (30-day cleanup)
- iCal feed worker (rebuilds `.ics` on task changes)
- Push notification worker using Web Push API (`webpush-go`) -- best-effort on iOS 16.4+, macOS Sonoma+
- Retry logic with exponential backoff for sync operations
- Prometheus metrics + Grafana dashboard (four golden signals + RabbitMQ queue depth, consumer utilisation, dead-letter size)
- Structured logging with zerolog
- Chaos testing: kill workers mid-processing, drop WebSocket connections, delay sync responses

**Done when:** You kill a worker mid-processing, verify the message is redelivered via RabbitMQ, and the idempotent handler processes it exactly once. The Grafana dashboard shows all four golden signals. Recurring tasks auto-generate on completion. Trash auto-purges after 30 days. The outbox poller reliably delivers events to RabbitMQ.

**You should understand:** The dual-write problem: why you cannot atomically write to PostgreSQL and publish to RabbitMQ without the outbox pattern. The difference between exactly-once delivery (impossible in general) and exactly-once processing (achievable with idempotent handlers). Why the event log is the "inside out database" -- every traditional database feature can be reimagined as a consumer of a log. Why this makes adding new features (search, analytics, notifications) a matter of adding a new consumer, not modifying existing code. Why timeout-based failure detection is imprecise. Why "just retry" is dangerous without idempotency. The difference between event time and processing time.

### Phase 4: Polish, Batch Processing & Enhancement

**Duration:** Ongoing
**Goal:** Feature parity with TickTick daily workflow, plus batch processing tools for recovery and analytics.

**Read first (DDIA):** Chapter 10 (Batch Processing) -- the opening sections on Unix philosophy and MapReduce. Your event log is the input, the projection rebuilder is the batch job, and the output is new read models. Focus on "The Output of Batch Workflows" which discusses building search indexes and materialised views from batch processing -- exactly what your projection rebuilder does.

**What to build:**

- Projection rebuilder CLI tool (Go binary): replays entire event log, reconstructs all read models from scratch -- used for disaster recovery, migration, and verifying projection correctness
- Event snapshot mechanism: periodically checkpoint aggregate state so replay doesn't start from event zero
- Batch analytics: process event log for statistics (tasks completed per week, average time to completion, most-used labels, overdue rate)
- Calendar view (monthly/weekly)
- Drag-and-drop reordering with touch-friendly long-press on iOS (exercises fractional index CRDT)
- Dark mode with system preference detection
- Full-text search (optionally via Bleve as another derived view from the event stream)
- Subtasks/checklists
- Keyboard shortcuts (Mac desktop productivity)
- iPad multitasking support (Split View, Slide Over)
- In-app install guidance banner for Safari Add to Home Screen / Add to Dock

**Done when:** You run the projection rebuilder CLI, verify the rebuilt read models match the live state exactly. The calendar view shows your tasks alongside Apple Calendar events. Dark mode toggles seamlessly. You have genuine feature parity with your daily TickTick workflow -- nothing makes you want to switch back.

**You should understand:** Why the ability to rebuild derived data from the event log is one of the most powerful properties of event-sourced systems. The Unix philosophy of composable, single-purpose tools applied to data processing. Why immutable inputs make batch processing safe and repeatable. The tradeoff between full replay (slow but complete) and snapshots (fast but adds complexity). How adding a search index as another derived view demonstrates the "unbundled database" concept from Chapter 12.

## 9. Technical Risks & Mitigations

| Risk                           | Impact                                                                                           | Mitigation                                                                                                                                                                                                                                          |
| ------------------------------ | ------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CRDT complexity                | Bugs in merge logic cause data loss or duplication                                               | Start with LWW-only (simplest CRDT). Add OR-Set and fractional index incrementally. Extensive property-based testing.                                                                                                                               |
| Safari storage eviction        | iOS Safari may clear IndexedDB if PWA is unused for weeks. Data loss on device.                  | Server-side per-aggregate snapshots updated incrementally on every sync. Automatic rehydration on launch if local state is empty. Always install to home screen for higher storage quotas.                                                           |
| No Background Sync             | Safari does not support Background Sync API. Offline changes only sync when app is foregrounded. | Aggressive sync on `visibilitychange` event. Short polling interval (30s) when active. Users naturally open the app frequently enough for a todo list.                                                                                              |
| Push notification reliability  | iOS PWA push notifications are less reliable than native APNs. May be delayed or missed.         | Treat push as enhancement, not critical path. Today view and overdue indicators are primary urgency mechanism. P2 priority -- not in MVP.                                                                                                           |
| Event store growth             | Unbounded event log for 1--3 users is unlikely to be an issue, but snapshot strategy needed long-term | Implement event snapshots in Phase 4. For 1--3 users, even years of events will be manageable.                                                                                                                                                      |
| Scope creep                    | Learning goals tempt over-engineering before the app is usable                                   | Phase 1 must be usable as a daily driver before advancing. Each phase has a clear "done" criteria.                                                                                                                                                  |

## 10. Key References & Learning Resources

- Martin Kleppmann -- *Designing Data-Intensive Applications* (event sourcing, CRDTs, distributed systems)
- Martin Kleppmann -- *CRDTs: The Hard Parts* (talk + paper)
- James Long -- *CRDTs for Mortals* (practical CRDT implementation in JS)
- Ink & Switch -- *Local-First Software* (philosophy and patterns for offline-first apps)
- RabbitMQ documentation -- Topic exchanges, dead-letter queues, Go client (`amqp091-go`)
- Workbox documentation -- Service worker strategies for PWA offline support
- Fractional Indexing -- David Greenspan's algorithm for collaborative ordering
- Apple Developer -- Safari Web Extensions and Web Apps documentation
- Apple Support -- Use Safari Web Apps on Mac (support.apple.com/en-mide/104996)
- MDN Web Docs -- Making PWAs Installable (Safari/WebKit specifics)
- TipTap documentation -- WYSIWYG markdown editor, ProseMirror extensions, serialisation
