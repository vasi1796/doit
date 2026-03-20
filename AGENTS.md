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

### Write Path
```
HTTP Request → Handler → CommandHandler → Aggregate (validates + produces events)
    → EventStore.Append (Postgres transaction)
    → Projector.Project (updates read model tables)
    → HTTP Response
```

### Read Path
```
HTTP Request → Handler → SELECT from read model tables (pgxpool) → HTTP Response
```

**Key patterns:**
- **Event Sourcing**: All state mutations produce events appended to the event store.
  Read models are projections derived from events. Never update read models directly.
- **CQRS**: Commands go through domain aggregates; queries hit read model tables directly.
- **Consumer-side interfaces**: Each package defines the interfaces it needs from its dependencies.

---

## Tech Stack

### Backend (implemented)
- **Go 1.22+** — `api/go.mod`
- **Chi v5** — HTTP router with middleware
- **pgx v5 + pgxpool** — Postgres driver and connection pooling
- **goose v3** — SQL migrations
- **zerolog** — structured JSON logging
- **golang-jwt/jwt v5** — JWT token creation/validation
- **golang.org/x/oauth2** — Google OAuth 2.0
- **go-chi/cors** — CORS middleware
- **PostgreSQL 16** — event store + read models

### Frontend (not yet implemented)
- **React 18+** with **TypeScript**, **Vite**, **Dexie.js**, **Tailwind CSS**

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
      api/main.go              # Server entry point: router, auth, domain stack wiring
      migrate/main.go          # Goose migration runner CLI
    internal/
      auth/                    # JWT (TokenService), Google OAuth, context helpers
      config/                  # Env var loading → Config struct
      domain/                  # Aggregates, commands, payloads, errors, CommandHandler
      eventstore/              # Event struct, Store (Append/Load), sentinel errors
      handler/                 # HTTP handlers: auth, task, list, label, response utils
      middleware/              # JWT auth middleware (cookie-based)
      projection/              # Projector: events → read model table updates
    migrations/                # SQL files: 001_events, 002_read_models
    Dockerfile
    go.mod
  web/                         # Frontend (not yet implemented)
    Dockerfile
  docs/
    adr/                       # 7 Architecture Decision Records
    design-document.md         # Full design spec
  scripts/backup.sh            # Database backup with retention
  .github/workflows/ci.yml    # GitHub Actions CI
  docker-compose.yml           # Postgres + Caddy + API
  Caddyfile                    # Reverse proxy config
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
- **Dexie.js + `useLiveQuery`** is the sole state management approach.
  Do NOT introduce Redux, Zustand, Jotai, or any other state library.
- **Unidirectional data flow**:
  `User Action → IndexedDB write (Dexie) → useLiveQuery → re-render`
- All CSS and JS must work in **Safari/WebKit**. No Chromium-only APIs.
- Minimum **44px tap targets** per Apple HIG.

---

## CRDT Conventions (Phase 2+)

| Data Type | CRDT Strategy | Notes |
|-----------|--------------|-------|
| Scalar fields (title, due date, status) | **LWW-Register** | Last-Writer-Wins using HLC timestamps |
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
- **Commands**: `CreateTask`, `CompleteTask`, `DeleteTask`, `MoveTask`, `AddLabel`, `RemoveLabel`, `CreateSubtask`, etc.
- **Payloads**: typed structs for `Event.Data` JSON (e.g., `TaskCreatedPayload`, `TaskCompletedPayload`)
- **EventLoader interface** (consumer-defined): `LoadByAggregate` + `Append`
- **EventProjector interface** (consumer-defined): `Project`

### `projection` — Events → read model tables
- `Projector.Project(ctx, []Event)` — dispatches to per-event handlers
- INSERT handlers use `ON CONFLICT DO UPDATE` (idempotent upsert)
- UPDATE handlers log warnings on zero rows affected

### `handler` — HTTP layer
- `TaskHandler`, `ListHandler`, `LabelHandler` — each with consumer-defined commander interfaces
- `AuthHandler` — Google OAuth login/callback, dev login, logout
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
POST   /api/v1/tasks                           → create task
GET    /api/v1/tasks                           → list tasks (?list_id, ?is_completed, ?inbox=true)
GET    /api/v1/tasks/{id}                      → task detail with subtasks + labels
PATCH  /api/v1/tasks/{id}                      → update description, move to list
DELETE /api/v1/tasks/{id}                      → soft-delete
POST   /api/v1/tasks/{id}/complete             → mark complete
POST   /api/v1/tasks/{id}/uncomplete           → mark incomplete
POST   /api/v1/tasks/{id}/subtasks             → create subtask
POST   /api/v1/tasks/{id}/subtasks/{sid}/complete → complete subtask
POST   /api/v1/tasks/{id}/labels               → add label
DELETE /api/v1/tasks/{id}/labels/{lid}         → remove label
```

### Lists & Labels (authenticated)
```
POST /api/v1/lists    → create list
GET  /api/v1/lists    → all user lists
POST /api/v1/labels   → create label
GET  /api/v1/labels   → all user labels
```

### Infrastructure (unauthenticated)
```
GET /healthz          → DB connectivity check
```

---

## Event Types

```
TaskCreated, TaskCompleted, TaskUncompleted, TaskDeleted, TaskMoved,
TaskDescriptionUpdated, LabelAdded, LabelRemoved, LabelCreated,
ListCreated, SubtaskCreated, SubtaskCompleted
```

Aggregate types: `task`, `list`, `label`

---

## Running Locally

```bash
docker compose up postgres -d                    # start Postgres
DATABASE_URL=postgres://doit:changeme@localhost:5432/doit?sslmode=disable make migrate  # apply migrations

# Dev mode (no Google creds needed):
DATABASE_URL=... DEV_MODE=true JWT_SECRET=$(openssl rand -base64 32) SECURE_COOKIES=false make run

# Tests:
make test                                        # unit tests
make test-integration                            # needs running Postgres
make vet                                         # go vet
```

---

## Phase Overview

| Phase | Scope | Status |
|-------|-------|--------|
| **Phase 1** | Online-only MVP — event store, projections, CRUD API, Google SSO, Docker Compose | Backend done, frontend pending |
| **Phase 2** | Offline-first + CRDT sync — Dexie.js, service worker, LWW/OR-Set merge, WebSocket push, HLC timestamps | Not started |
| **Phase 3** | RabbitMQ + workers — transactional outbox, topic exchanges, DLQ, recurring tasks, trash purge, Prometheus/Grafana | Not started |
| **Phase 4** | Polish — projection rebuilder CLI, calendar view, dark mode, search, subtasks, drag-and-drop, keyboard shortcuts | Not started |

See `docs/design-document.md` for full phase details including DDIA reading plan.

---

## Key Constraints

- **All mutations through event store** — never write directly to read model tables
- **Consumer-side interfaces** — interfaces defined where they're used, not where they're implemented
- **User scoping** — every read query filters by `user_id`; write commands verify aggregate ownership
- **Safari-only PWA** — no Background Sync, no Chromium-only APIs
- **1-3 users** — do not over-engineer for scale
