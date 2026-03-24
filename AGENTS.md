# AGENTS.md — DoIt Project Context for LLM Coding Agents

> This file provides context for LLM coding agents (Claude Code, Codex, etc.)
> working with the DoIt codebase. Keep under 500 lines.

---

## Project Purpose

DoIt is a personal, self-hosted task management Progressive Web App (PWA)
targeting the Apple/Safari ecosystem exclusively. It is built for 1-3 users
with Google SSO authentication.

Core design goals:
- **Event-sourced architecture** as a learning vehicle and for full audit trail
- **CRDT-based sync** for offline-first multi-device usage (Phase 2)
- **Offline-first PWA** — the app must work without a network connection
- **Self-hosted** — runs on a single server via Docker Compose

---

## Architecture Overview

### Write Path (online — individual REST endpoints)
```
HTTP Request → Handler → CommandHandler(HLC) → Aggregate (validates + produces events)
    → TX { EventStore.Append + Outbox.Insert } (single Postgres transaction)
    → HTTP Response (immediate — projections are async)
    ↓ (async, 200ms poll)
Outbox Poller → RabbitMQ (topic exchange: doit.events, broker auto-reconnects with backoff)
    → Projection Worker (doit.projections queue) → updates read model tables
    → Recurring Worker (doit.recurring queue) → creates next occurrence on task completion
```

### Write Path (offline-first — sync engine)
```
User action → db/operations.ts → IndexedDB write (instant)
    → SyncOp queued in syncQueue table
    → SyncEngine flushes on foreground/30s poll (failed ops retry up to 5 times)
    → POST /api/v1/sync (batched operations)
    → Server: CommandHandler processes batch → events appended
    → Response: confirmation + remote events
    → Client: merge-events.ts applies remote events to IndexedDB via per-field LWW
    → useLiveQuery auto-re-renders
```

### Read Path
```
useLiveQuery (Dexie.js) → IndexedDB → React component
```
All reads come from local IndexedDB. The API is never queried directly for reads
(except during initial sync on app launch).

**Key patterns:**
- **Event Sourcing**: All state mutations produce events appended to the event store.
  Read models are projections derived from events. Never update read models directly.
- **CQRS**: Commands go through domain aggregates; queries hit read model tables directly.
- **Transactional Outbox**: Events and outbox rows written in a single Postgres transaction.
  Outbox poller publishes to RabbitMQ. Projections are async via workers.
- **Offline-first**: Writes go to IndexedDB immediately, sync to server when online.
- **HLC timestamps**: Hybrid Logical Clocks provide causal ordering for CRDT merge. Tracked per field so concurrent edits to different fields are both preserved.
- **CRDTs**: LWW-Register (scalars, per-field HLC), OR-Set (labels), Fractional Indexing (ordering).
- **Consumer-side interfaces**: Each package defines the interfaces it needs from its dependencies.

---

## Tech Stack

### Backend
- **Go 1.26+** — `api/go.mod`
- **Chi v5** — HTTP router with middleware
- **pgx v5 + pgxpool** — Postgres driver and connection pooling
- **goose v3** — SQL migrations
- **zerolog** — structured JSON logging
- **golang-jwt/jwt v5** — JWT token creation/validation
- **golang.org/x/oauth2** — Google OAuth 2.0
- **go-chi/cors** — CORS middleware
- **PostgreSQL 16** — event store + read models

### Frontend
- **React 18** with **TypeScript** — `web/package.json`
- **Vite** — build + dev server with API proxy
- **Tailwind CSS v4** — utility-first styling
- **React Router v7** — client-side routing

### Infrastructure
- **Docker Compose** — Postgres + Caddy + API
- **Caddy** — reverse proxy, automatic TLS, static file serving
- **GitHub Actions** — CI with unit + integration tests

---

## Project Structure

```
doit/
  api/                         # Go backend
    cmd/
      api/main.go              # Server: router, auth, domain stack wiring, auto-migrations
      migrate/main.go          # Goose migration runner CLI (manual operations)
      rebuild/main.go          # Projection rebuilder CLI (replays event log)
    internal/
      auth/                    # JWT (TokenService), Google OAuth, context helpers
      config/                  # Env var loading → Config struct
      domain/                  # Aggregates, commands, payloads, errors, CommandHandler
      eventstore/              # Event struct, Store (Append/Load), sentinel errors
      handler/                 # HTTP handlers: auth, task, list, label, iCal feed, response utils
      middleware/              # JWT auth middleware (cookie-based)
      projection/              # Projector: events → read model table updates
    migrations/                # SQL files: 001–007, embedded via migrations/embed.go
    Dockerfile
    go.mod
  web/                         # React frontend
    src/
      api/                     # Typed fetch client + TS interfaces
      components/
        common/                # DatePicker, TimePicker, PriorityPicker, RecurrencePicker,
                               # ListSelect, LabelPicker, Toast, EmptyState,
                               # MarkdownEditor, InlineMarkdown, InstallBanner,
                               # SearchOverlay (Cmd+K global search),
                               # CalendarFeedLink (iCal subscription)
        layout/                # AppLayout (extracted hooks: useTaskCounts, useKeyboardShortcuts,
                               # useMobileDrawer), Sidebar, BottomNav
        tasks/                 # QuickAdd, TaskItem, TaskDetail, TaskList, TaskProperties,
                               # SubtaskSection, LabelsSection
      hooks/                   # useTasks, useLists, useLabels, useTaskDetail
      pages/                   # Inbox, Today (with Overdue section), Upcoming, Matrix, Calendar, List, Label, Completed, Trash, Login
      constants.ts             # Shared color palette, PRIORITY_COLORS
    public/                    # PWA manifest, app icons
    Dockerfile
  deploy/                      # Deploy webhook sidecar
    main.go                    # Standalone Go binary — HMAC-verified GitHub webhook
    Dockerfile                 # docker:27-cli + git + webhook binary
  docs/
    adr/                       # Architecture Decision Records
    diagrams/                  # Mermaid architecture diagrams
    deployment.md              # Production deployment guide
    design-document.md         # Full design spec
  scripts/
    backup.sh                  # Database backup with daily/weekly retention
    deploy.sh                  # First-time deploy + health check convenience script
  .github/workflows/ci.yml    # GitHub Actions CI (auto-updates Linux visual baselines)
  docker-compose.yml           # Full stack: Postgres, RabbitMQ, API, workers, Caddy, web-build, deployer
  Caddyfile                    # Reverse proxy, TLS, static assets, security headers
  Makefile                     # Build commands
```

---

## Code Conventions

### Go
- **Table-driven tests** for all unit tests.
- **Explicit error handling** — never use `panic()` for recoverable errors.
  Always return and check errors.
- **Interfaces defined by consumer** — consumers define the interfaces
  they need; producers provide concrete implementations.
- **Event naming**: past-tense PascalCase — `TaskCreated`, `TaskCompleted`, `LabelAdded`
- **Aggregate handler naming**: `HandleCreate`, `HandleComplete`, `HandleDelete`, etc.

### Frontend (React/TypeScript)
- **Hooks + fetch** for Phase 1 data fetching (replaced by Dexie.js `useLiveQuery` in Phase 2).
- **No state management libraries** — React Context only for layout-level shared data (lists, labels, counts).
- **Custom pickers** — DatePicker (calendar), TimePicker (grid), RecurrencePicker, ListSelect use
  fixed-position popovers, not native `<select>` or `<input type="date">`.
- **Toast notifications** for all user-facing feedback (success, error).
- **Shared constants** — color palette in `constants.ts`, not duplicated across files.
- All CSS and JS must work in **Safari/WebKit**. No Chromium-only APIs.
- Minimum **44px tap targets** per Apple HIG.
- All inputs ≥16px font to prevent iOS Safari zoom.

---

## CRDT Conventions (Phase 2+)

| Data Type | CRDT Strategy | Notes |
|-----------|--------------|-------|
| Scalar fields (title, due date, status) | **LWW-Register** | Last-Writer-Wins using per-field HLC timestamps |
| Labels on a task | **OR-Set** | Observed-Remove Set — concurrent add/remove resolved |
| Task/subtask ordering | **Fractional Indexing** | String position keys between adjacent items |
| Timestamps | **HLC** | Hybrid Logical Clock for causal ordering |
| Markdown descriptions | **LWW-Register** | Whole-string replacement (see ADR-006) |

### Conflict Resolution Policies
- **Edit resurrects delete** — concurrent edit + delete → edit wins, task restored
- **Complete resurrects delete** — concurrent complete + delete → complete wins
- **Concurrent list moves** — Last-Writer-Wins based on HLC timestamp

---

## Backend Package Guide

### `eventstore` — Append-only event persistence
- `Event` struct: ID, AggregateID, AggregateType, EventType, UserID, Data (json.RawMessage), Timestamp, Version
- `Store.Append(ctx, []Event)` — transactional insert, returns `ErrVersionConflict` on duplicate version
- `Store.LoadByAggregate(ctx, id)` — all events for an aggregate, ordered by version
- `Store.LoadByUserSince(ctx, userID, since)` — for sync

### `domain` — Business rules (no DB dependency)
- **Aggregates** (`TaskAggregate`, `ListAggregate`, `LabelAggregate`): pure objects that replay events via `Apply()` and validate commands via `Handle*()` methods
- **CommandHandler**: orchestrates load → replay → handle → append → project. Verifies user ownership on load.
- **Commands**: `CreateTask`, `CompleteTask`, `DeleteTask`, `RestoreTask`, `MoveTask`, `AddLabel`, `RemoveLabel`, `CreateSubtask`, `CompleteSubtask`, `UncompleteSubtask`, `UpdateTaskTitle`, `UpdateTaskPriority`, `UpdateTaskDueDate`, `UpdateTaskDueTime`, `UpdateTaskRecurrence`, `UpdateSubtaskTitle`, etc.
- **Payloads**: typed structs for `Event.Data` JSON
- **EventLoader interface** (consumer-defined): `LoadByAggregate` + `Append`
- **EventProjector interface** (consumer-defined): `Project`
- **Recurring tasks**: completing a task with recurrence_rule + due_date auto-creates the next occurrence as a separate aggregate

### `projection` — Events → read model tables
- `Projector.Project(ctx, []Event)` — dispatches to per-event handlers
- INSERT handlers use `ON CONFLICT DO UPDATE` (idempotent upsert)
- UPDATE handlers log warnings on zero rows affected

### `handler` — HTTP layer
- `TaskHandler`, `ListHandler`, `LabelHandler` — each with consumer-defined commander interfaces
- `AuthHandler` — Google OAuth login/callback, dev login, logout
- `scanTaskRow` helper — shared task row scanning for List and Get
- `loadLabelsForTasks` / `loadSubtasksForTasks` — batch loaders to avoid N+1
- Shared utils: `writeJSON`, `writeError`, `readJSON`, `parseUUID`, `requireUserID`, `mapDomainError`

### `auth` — Authentication primitives
- `TokenService` — issue/validate HS256 JWTs
- `GoogleOAuth` — OAuth2 config, code exchange, userinfo fetch
- `WithUserID` / `UserIDFromContext` — context helpers

### `middleware` — HTTP middleware
- `JWTAuth` — reads `doit_token` cookie, validates JWT, injects user ID into context

---

## API Endpoints

### Auth (unauthenticated)
```
GET  /auth/google/login       → redirect to Google consent
GET  /auth/google/callback    → exchange code, set JWT cookie, redirect
POST /auth/dev                → dev login (DEV_MODE only)
POST /auth/logout             → clear cookie
```

### Tasks (authenticated via JWT cookie)
```
POST   /api/v1/tasks                                → create task
GET    /api/v1/tasks                                → list tasks (?list_id, ?label_id, ?is_completed, ?is_deleted, ?inbox)
GET    /api/v1/tasks/{id}                           → task detail with subtasks + labels
PATCH  /api/v1/tasks/{id}                           → update (title, description, priority, due_date, due_time, recurrence_rule, list_id+position)
DELETE /api/v1/tasks/{id}                           → soft-delete
POST   /api/v1/tasks/{id}/complete                  → mark complete (auto-creates next for recurring)
POST   /api/v1/tasks/{id}/uncomplete                → mark incomplete
POST   /api/v1/tasks/{id}/restore                   → restore from trash
POST   /api/v1/tasks/{id}/subtasks                  → create subtask
PATCH  /api/v1/tasks/{id}/subtasks/{sid}            → update subtask title
POST   /api/v1/tasks/{id}/subtasks/{sid}/complete   → complete subtask
POST   /api/v1/tasks/{id}/subtasks/{sid}/uncomplete → uncomplete subtask
POST   /api/v1/tasks/{id}/labels                    → add label
DELETE /api/v1/tasks/{id}/labels/{lid}              → remove label
```

### Lists & Labels (authenticated)
```
POST   /api/v1/lists      → create list
GET    /api/v1/lists       → all user lists
DELETE /api/v1/lists/{id}  → delete list (moves tasks to inbox)
POST   /api/v1/labels      → create label
GET    /api/v1/labels       → all user labels
DELETE /api/v1/labels/{id}  → delete label (cascade removes from tasks)
```

### Infrastructure (unauthenticated)
```
GET /healthz    → DB connectivity check (supports HEAD for Docker health check)
```

---

## Event Types

```
TaskCreated, TaskCompleted, TaskUncompleted, TaskDeleted, TaskRestored,
TaskMoved, TaskTitleUpdated, TaskDescriptionUpdated, TaskPriorityUpdated,
TaskDueDateUpdated, TaskDueTimeUpdated, TaskRecurrenceUpdated, TaskReordered,
LabelAdded, LabelRemoved, LabelCreated, LabelDeleted,
ListCreated, ListDeleted,
SubtaskCreated, SubtaskCompleted, SubtaskUncompleted, SubtaskTitleUpdated
```

Aggregate types: `task`, `list`, `label`

---

## Running Locally

```bash
docker compose up postgres -d                    # start Postgres

# Dev mode (3 terminals):
# T1: DATABASE_URL=... DEV_MODE=true SECURE_COOKIES=false make run
# T2: cd web && npm run dev -- --host
# T3: open http://localhost:5173

# Docker (full stack):
docker compose up -d --build                     # builds API + serves frontend via Caddy
open https://localhost

# Tests:
make test                                        # unit tests
make test-integration                            # needs running Postgres
make vet                                         # go vet
cd web && npm run build                          # frontend build
```

---

## Phase Overview

| Phase | Scope | Status |
|-------|-------|--------|
| **Phase 1** | Online-only MVP — event store, projections, CRUD API+UI, Google SSO, Docker Compose | Backend + frontend done |
| **Phase 2** | Offline-first + CRDT sync — Dexie.js, service worker, LWW/OR-Set merge, sync engine, WebSocket push, HLC timestamps, aggregate snapshots | Done |
| **Phase 3** | RabbitMQ + async projections — transactional outbox, topic exchanges, DLQ, projection worker, recurring tasks worker | Done |
| **Phase 4** | Polish — projection rebuilder CLI, calendar view, dark mode, search, drag-and-drop, keyboard shortcuts | Not started |

See `docs/design-document.md` for full phase details.
See `docs/adr/008-phase1-migration-risks.md` for known refactor points between phases.

---

## Key Constraints

- **All mutations through event store** — never write directly to read model tables
- **Consumer-side interfaces** — interfaces defined where they're used, not where they're implemented
- **User scoping** — every read query filters by `user_id`; write commands verify aggregate ownership
- **Safari-only PWA** — no Background Sync, no Chromium-only APIs
- **1-3 users** — do not over-engineer for scale
