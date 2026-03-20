# CLAUDE.md — Claude Code Configuration for DoIt

> Configuration and rules for Claude Code when working in this repository.

---

## Project Overview

DoIt is a self-hosted, offline-first task management PWA for Safari/Apple
ecosystem. Event-sourced backend in Go, React/TypeScript frontend with
Dexie.js for local state. See AGENTS.md for full architectural context.

---

## Directory Structure

```
api/              Go backend
  cmd/api/        Server entry point (router, auth, domain stack wiring)
  cmd/migrate/    Migration runner CLI (goose)
  internal/
    auth/          JWT tokens, Google OAuth, context helpers
    config/        Env var loading
    domain/        Aggregates, commands, payloads, CommandHandler
    eventstore/    Event store (append/load/query)
    handler/       HTTP handlers (task, list, label, auth, response utils)
    middleware/     JWT auth middleware
    projection/    Event → read model table updates
  migrations/     SQL migration files
web/              React frontend (not yet implemented)
docs/adr/         Architecture Decision Records
scripts/          Backup and utility scripts
```

---

## Coding Style

### Go
- **Table-driven tests** for all unit tests. Group test cases in a slice of
  structs with `name`, `input`, `expected`, and run with `t.Run(tc.name, ...)`.
- **Explicit error handling** — never use `panic()` for recoverable errors.
  Always `return err` and check every error return value.
- Handle errors, do not ignore them. Never use `_ = someFunc()` for functions
  that return errors unless there is a documented reason.
- Interfaces are defined by the consumer package, not the provider.
- Keep functions short and focused. Prefer composition over inheritance.

### Frontend (TypeScript/React)
- Use functional components with hooks.
- Tailwind CSS for styling. No CSS modules or styled-components.
- All browser APIs must work in Safari/WebKit. Test assumptions against
  WebKit compatibility.
- Minimum 44px touch targets (Apple HIG).

### Commit Messages
- Use conventional commit format: `type(scope): description`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`
- Scope examples: `eventstore`, `projection`, `sync`, `ui`, `auth`, `domain`
- Example: `feat(eventstore): add aggregate snapshot support`
- Keep the subject line under 72 characters.

---

## Critical Rules

These rules are non-negotiable. Violating them will break the architecture.

### 1. ALL state mutations MUST go through the event store
Never write directly to read model tables (`tasks`, `lists`, `labels`, etc.).
All changes produce events that are appended to the `events` table. Projection
handlers then update read models from those events.

### 2. Every event handler must be idempotent
Event handlers (projections, workers) may receive the same event more than once.
They must produce the same result regardless of how many times they process an
event. Use `ON CONFLICT` for inserts, plain `UPDATE` for modifications.

### 3. Frontend state comes only from Dexie.js useLiveQuery
Do NOT introduce Redux, Zustand, Jotai, React Context for state management,
or any other state library. The data flow is:
```
User action -> Dexie.js write to IndexedDB -> useLiveQuery auto-updates -> React re-renders
```

### 4. Safari/WebKit only — no Chromium-only APIs
This is a Safari PWA. Do not use:
- Background Sync API (not available in Safari)
- Web Bluetooth, Web USB, or other Chromium-only APIs
- CSS features without WebKit support

### 5. Minimum 44px tap targets
All interactive elements must have at least 44x44px touch targets per Apple
Human Interface Guidelines.

### 6. User scoping on all operations
Every read query must include `WHERE user_id = $1`. Every write command must
verify the aggregate belongs to the requesting user (done in `loadTaskAggregate`).

---

## Test Expectations

- **Unit tests** for all business logic (domain, event handling, config, auth).
- **Table-driven tests** in Go — no exceptions.
- **Integration tests** (build tag `//go:build integration`) for event store
  and projection flows against real Postgres.
- Frontend: test Dexie.js database operations and sync logic.

---

## Common Commands

```bash
# Start Postgres
docker compose up postgres -d

# Run migrations
DATABASE_URL=postgres://doit:changeme@localhost:5432/doit?sslmode=disable make migrate

# Run API (dev mode)
DATABASE_URL=... DEV_MODE=true JWT_SECRET=... SECURE_COOKIES=false make run

# Run Go tests
make test

# Run Go tests with verbose output
make test-verbose

# Run integration tests (needs Postgres)
make test-integration

# Run Go vet
make vet

# Run frontend dev server
cd web && npm run dev

# Build everything
make build
```

---

## Environment Setup

- Go 1.26+ required
- Node.js 20+ and npm for the frontend
- Docker and Docker Compose for local services (Postgres)
- PostgreSQL 16 (via Docker for local dev)
- Google OAuth 2.0 credentials for production auth
- `DEV_MODE=true` enables `/auth/dev` endpoint for local testing without Google

---

## Event Naming Reference

Events: `TaskCreated`, `TaskCompleted`, `TaskUncompleted`, `TaskDeleted`,
`TaskMoved`, `TaskDescriptionUpdated`, `LabelAdded`, `LabelRemoved`,
`LabelCreated`, `ListCreated`, `SubtaskCreated`, `SubtaskCompleted`

---

## Useful Context Files

- `AGENTS.md` — full architecture overview, package guide, API endpoints
- `docs/adr/` — Architecture Decision Records explaining key design choices
- `docs/design-document.md` — complete design specification
