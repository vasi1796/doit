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
  cmd/            Binary entry points (e.g., api/cmd/api/main.go)
  internal/       Go packages — domain, eventstore, projection, handler, middleware
  migrations/     SQL migration files (goose)
  go.mod
web/              React frontend — components, hooks, db, sync, types
docs/adr/         Architecture Decision Records
scripts/          Build, migration, and utility scripts
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
- Scope examples: `eventstore`, `projection`, `sync`, `ui`, `auth`
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
event. Use the event's `id` and `version` for deduplication.

### 3. Frontend state comes only from Dexie.js useLiveQuery
Do NOT introduce Redux, Zustand, Jotai, React Context for state management,
or any other state library. The data flow is:
```
User action -> Dexie.js write to IndexedDB -> useLiveQuery auto-updates -> React re-renders
```
This is the single source of truth for the frontend.

### 4. Safari/WebKit only — no Chromium-only APIs
This is a Safari PWA. Do not use:
- Background Sync API (not available in Safari)
- Web Bluetooth, Web USB, or other Chromium-only APIs
- CSS features without WebKit support
Always verify WebKit compatibility for any browser API.

### 5. Minimum 44px tap targets
All interactive elements must have at least 44x44px touch targets per Apple
Human Interface Guidelines.

---

## Test Expectations

- **Unit tests** for all business logic (domain, event handling, CRDT merge).
- **Table-driven tests** in Go — no exceptions.
- **Integration tests** for the event store -> projection flow: write events,
  verify read models are correctly projected.
- **CRDT merge tests**: test all conflict scenarios (concurrent edit, edit vs
  delete, concurrent label add/remove, concurrent list moves).
- Frontend: test Dexie.js database operations and sync logic.

---

## Common Commands

```bash
# Start all services locally
docker compose up

# Run Go tests
make test

# Run Go tests with verbose output
make test-verbose

# Run Go linter
make lint

# Run frontend dev server
cd web && npm run dev

# Run frontend tests
cd web && npm test

# Run frontend linter
cd web && npm run lint

# Build everything
make build

# Run the full stack locally
make run
```

---

## Environment Setup

- Go 1.22+ required
- Node.js 20+ and npm for the frontend
- Docker and Docker Compose for local services (Postgres, RabbitMQ)
- PostgreSQL 16 (via Docker for local dev)
- RabbitMQ 3.13+ (via Docker for local dev, Phase 3)

---

## Event Naming Reference

Events: `TaskCreated`, `TaskCompleted`, `TaskUncompleted`, `TaskDeleted`,
`TaskMoved`, `TaskDescriptionUpdated`, `LabelAdded`, `LabelRemoved`,
`ListCreated`, `SubtaskCreated`, `SubtaskCompleted`

Handlers: `HandleTaskCreated`, `HandleTaskCompleted`, etc.

---

## Useful Context Files

- `AGENTS.md` — full architecture overview, data model, API endpoints
- `docs/adr/` — Architecture Decision Records explaining key design choices
