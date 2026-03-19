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
- **CRDT-based sync** for offline-first multi-device usage
- **Offline-first PWA** — the app must work without a network connection
- **Self-hosted** — runs on a single server via Docker Compose

---

## Architecture Overview

```
User Action
    |
    v
[Frontend: React + Dexie.js (IndexedDB)]
    |
    v  (online: POST /events)
[Go API Server (Chi router)]
    |
    v
[Event Store (PostgreSQL — append-only)]
    |
    v  (Phase 3: Transactional Outbox)
[RabbitMQ Topic Exchanges]
    |
    v
[Projection Workers] --> [Read Models (PostgreSQL)]
```

**Key architectural patterns:**
- **Event Sourcing**: All state mutations produce events appended to the event store.
  Read models are projections derived from events. Never update read models directly.
- **CQRS**: Commands write events; queries read from projected read models.
- **Offline-first**: The frontend uses IndexedDB (via Dexie.js) as its local source
  of truth. Sync happens when connectivity is available.
- **Transactional Outbox** (Phase 3): Events and outbox rows are written in a single
  Postgres transaction. A poller publishes outbox rows to RabbitMQ.

---

## Tech Stack

### Backend
- **Go 1.22+**
- **Chi** — HTTP router
- **PostgreSQL 16** — event store + read models
- **RabbitMQ 3.13+** — async event distribution (Phase 3)
- **Google SSO** — authentication for 1-3 users

### Frontend
- **React 18+** with **TypeScript**
- **Vite** — build tooling
- **Dexie.js** — IndexedDB wrapper + `useLiveQuery` for reactive state
- **TipTap** — rich text / markdown editing for task descriptions
- **Tailwind CSS** — styling
- **Workbox** — service worker for offline caching

### Infrastructure
- **Docker Compose** — local development and self-hosted deployment

---

## Project Structure

```
doit/
  api/                    # Go backend
    cmd/
      api/                # API server entry point
      migrate/            # Migration runner CLI
    internal/             # Go packages (not importable externally)
      config/             # Environment configuration
      domain/             # Domain types, events, aggregates
      eventstore/         # Event store implementation
      projection/         # Read model projections
      handler/            # HTTP handlers
      middleware/          # HTTP middleware (auth, logging)
      rabbitmq/           # RabbitMQ publisher/consumer (Phase 3)
      outbox/             # Transactional outbox poller (Phase 3)
    migrations/           # SQL migration files (goose)
    go.mod
    Dockerfile
  web/                    # Frontend React application
    src/
      components/         # React components
      hooks/              # Custom React hooks
      db/                 # Dexie.js database schema and queries
      sync/               # Sync engine (online/offline)
      types/              # TypeScript type definitions
    Dockerfile
  docs/
    adr/                  # Architecture Decision Records
  scripts/                # Build, migration, and utility scripts
  docker-compose.yml      # Local dev / self-hosted deployment
  Caddyfile               # Reverse proxy configuration
  Makefile                # Build commands
  AGENTS.md               # This file
  CLAUDE.md               # Claude Code-specific configuration
```

---

## Code Conventions

### Go

- **Table-driven tests** for all unit tests.
- **Explicit error handling** — never use `panic()` for recoverable errors.
  Always return and check errors.
- **Interfaces defined at package boundaries** — consumers define the interfaces
  they need; producers provide concrete implementations.
- **Handler naming**: `HandleTaskCreated`, `HandleTaskCompleted`,
  `HandleLabelAdded`, etc. — prefixed with `Handle` followed by the event name.

### Frontend (React/TypeScript)

- **Dexie.js + `useLiveQuery`** is the sole state management approach.
  Do NOT introduce Redux, Zustand, Jotai, or any other state library.
- **Unidirectional data flow**:
  ```
  User Action -> IndexedDB write (via Dexie) -> useLiveQuery -> re-render
  ```
- All CSS and JS must work in **Safari/WebKit**. Do not use Chromium-only APIs.
- Minimum **44px tap targets** per Apple Human Interface Guidelines.

### Event Naming

Events use past-tense PascalCase names describing what happened:
- `TaskCreated`
- `TaskCompleted`
- `TaskUncompleted`
- `TaskDeleted`
- `TaskMoved`
- `TaskDescriptionUpdated`
- `LabelAdded`
- `LabelRemoved`
- `ListCreated`
- `SubtaskCreated`
- `SubtaskCompleted`

---

## Event Sourcing Model

### Event Store Schema

```sql
CREATE TABLE events (
    id              UUID PRIMARY KEY,
    aggregate_id    UUID NOT NULL,
    aggregate_type  TEXT NOT NULL,       -- 'task', 'list', 'label'
    event_type      TEXT NOT NULL,       -- 'TaskCreated', 'TaskCompleted', etc.
    user_id         UUID NOT NULL,
    data            JSONB NOT NULL,      -- event-specific payload
    timestamp       TIMESTAMPTZ NOT NULL,-- HLC timestamp
    version         INTEGER NOT NULL     -- per-aggregate monotonic version
);
```

### Mutation Flow

1. Client sends a command (e.g., "complete task X").
2. The aggregate validates the command and produces one or more events.
3. Events are appended to the event store within a transaction.
4. (Phase 3) An outbox row is written in the same transaction.
5. (Phase 3) A poller reads the outbox and publishes events to RabbitMQ.
6. Projection workers consume events and update read models.

### Read Model Entities

- `users` — user profile data from Google SSO
- `lists` — task lists
- `tasks` — projected task state
- `labels` — user-defined labels
- `task_labels` — many-to-many relationship
- `subtasks` — task sub-items
- `user_config` — per-user settings (theme, default list, etc.)
- `aggregate_snapshots` — periodic aggregate state snapshots for performance

---

## CRDT Conventions

Used for offline-first sync between devices (Phase 2+):

| Data Type | CRDT Strategy | Notes |
|-----------|--------------|-------|
| Scalar fields (title, due date, status) | **LWW-Register** | Last-Writer-Wins using HLC timestamps |
| Labels on a task | **OR-Set** | Observed-Remove Set — concurrent add/remove handled correctly |
| Task/subtask ordering | **Fractional Indexing** | Position keys between adjacent items |
| Timestamps | **HLC** | Hybrid Logical Clock for causal ordering |
| Markdown descriptions | **LWW-Register** | Whole-string replacement, no character-level merge (see ADR-006) |

### Conflict Resolution Policies

- **Edit resurrects delete**: If one device deletes a task and another edits it
  concurrently, the edit wins and the task is restored.
- **Complete resurrects delete**: If one device deletes a task and another completes
  it concurrently, the completion wins and the task is restored.
- **Concurrent list moves**: Last-Writer-Wins based on HLC timestamp.

---

## API Endpoints Summary

All API routes are prefixed with `/api/v1`.

### Authentication
- `GET  /auth/google/login` — initiate Google SSO
- `GET  /auth/google/callback` — OAuth callback
- `POST /auth/refresh` — refresh JWT token

### Events
- `POST /events` — append new events (primary write path)
- `GET  /events?since=<timestamp>` — fetch events since timestamp (sync)

### Read Models (Queries)
- `GET    /lists` — all lists for current user
- `GET    /lists/:id` — single list with tasks
- `GET    /tasks/:id` — single task detail
- `GET    /labels` — all labels for current user
- `GET    /user/config` — user configuration

### iCal
- `GET /ical/:token` — iCal feed (token-based auth, no SSO)

---

## Running Locally

```bash
# Start all services (Postgres, RabbitMQ, API server, frontend dev server)
docker compose up

# Run Go tests
make test

# Run Go linter
make lint

# Run frontend tests
cd web && npm test

# Run everything
make run
```

---

## Phase Overview

| Phase | Scope | Key Deliverables |
|-------|-------|-----------------|
| **Phase 1** | Online-only MVP | Event store, basic projections, CRUD UI, Google SSO, Docker Compose |
| **Phase 2** | Offline + CRDT | Dexie.js local store, service worker, CRDT merge, sync engine |
| **Phase 3** | RabbitMQ + Workers | Transactional outbox, RabbitMQ topic routing, projection workers, DLQ |
| **Phase 4** | Polish | iCal feed, keyboard shortcuts, drag-and-drop, recurring tasks, performance |

---

## Key Constraints

- **Safari-only PWA**: No Background Sync API. Push notifications are best-effort.
  Storage eviction is a risk — the app must handle graceful re-sync.
- **Token auth for iCal**: The iCal feed uses a long-lived token in the URL,
  not session-based auth.
- **All mutations through event store**: Never write directly to read model tables.
- **1-3 users**: Designed for personal/family use. Do not over-engineer for scale.
