# CLAUDE.md — Claude Code Configuration for DoIt

> Configuration and rules for Claude Code when working in this repository.

---

## Project Overview

DoIt is a self-hosted, offline-first task management PWA for Safari/Apple
ecosystem. Event-sourced backend in Go, React/TypeScript frontend.
See AGENTS.md for full architectural context.

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
web/              React frontend
  src/
    api/           Typed fetch client + TypeScript interfaces
    components/    Common pickers, layout (sidebar/bottom nav), task components
    hooks/         Data fetching hooks (useTasks, useLists, useLabels, useTaskDetail)
    pages/         Route pages (Inbox, Today, Upcoming, List, Label, Completed, Trash, Login)
    constants.ts   Shared color palette
docs/adr/         Architecture Decision Records (8 total)
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
- Extract helpers when functions exceed ~80 lines (`scanTaskRow`, `loadLabelsForTasks`, etc.).

### Frontend (TypeScript/React)
- Use functional components with hooks.
- Tailwind CSS for styling. No CSS modules or styled-components.
- All browser APIs must work in Safari/WebKit. No Chromium-only APIs.
- Minimum 44px touch targets (Apple HIG).
- All text inputs ≥16px font size to prevent iOS Safari auto-zoom.
- Custom pickers (DatePicker, TimePicker, RecurrencePicker, ListSelect) — use
  fixed-position popovers, not native `<select>` or `<input type="date/time">`.
- Toast notifications for all user-facing feedback.
- Shared color constants in `constants.ts`.
- `aria-label` on all icon-only buttons and placeholder-only inputs.

### Commit Messages
- Use conventional commit format: `type(scope): description`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`
- Scope examples: `eventstore`, `projection`, `sync`, `ui`, `auth`, `domain`, `web`
- Keep the subject line under 72 characters.

---

## Critical Rules

These rules are non-negotiable. Violating them will break the architecture.

### 1. ALL state mutations MUST go through the event store
Never write directly to read model tables (`tasks`, `lists`, `labels`, etc.).
All changes produce events that are appended to the `events` table. Projection
handlers then update read models from those events.
Exception: list/label deletion uses direct SQL in Phase 1 (not event-sourced).

### 2. Every event handler must be idempotent
Event handlers (projections, workers) may receive the same event more than once.
They must produce the same result regardless of how many times they process an
event. Use `ON CONFLICT` for inserts, plain `UPDATE` for modifications.

### 3. Frontend state comes only from hooks (Phase 1) or Dexie.js useLiveQuery (Phase 2)
Do NOT introduce Redux, Zustand, Jotai, or any other state library.
React Context is used only for layout-level shared data (lists, labels, counts).

### 4. Safari/WebKit only — no Chromium-only APIs
This is a Safari PWA. Do not use:
- Background Sync API (not available in Safari)
- Web Bluetooth, Web USB, or other Chromium-only APIs
- CSS features without WebKit support
- Native `<input type="date/time">` — use custom pickers instead (Safari PWA compatibility)

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
- Frontend: no tests yet (Phase 1). Add when Dexie.js is introduced (Phase 2).

---

## Common Commands

```bash
# Start Postgres
docker compose up postgres -d

# Run API locally (dev mode)
set -a && source .env && set +a
DATABASE_URL=postgres://doit:changeme@localhost:5432/doit?sslmode=disable \
DEV_MODE=true SECURE_COOKIES=false make run

# Run frontend dev server (hot reload)
cd web && npm run dev -- --host

# Run Go tests
make test

# Run integration tests (needs Postgres)
make test-integration

# Run Go vet
make vet

# Build frontend
cd web && npm run build

# Docker full stack
docker compose up -d --build
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
`TaskRestored`, `TaskMoved`, `TaskTitleUpdated`, `TaskDescriptionUpdated`,
`TaskPriorityUpdated`, `TaskDueDateUpdated`, `TaskDueTimeUpdated`,
`TaskRecurrenceUpdated`, `LabelAdded`, `LabelRemoved`, `LabelCreated`,
`LabelDeleted`, `ListCreated`, `ListDeleted`, `SubtaskCreated`,
`SubtaskCompleted`, `SubtaskUncompleted`, `SubtaskTitleUpdated`

---

## Useful Context Files

- `AGENTS.md` — full architecture overview, package guide, API endpoints
- `docs/adr/` — Architecture Decision Records (8 total)
- `docs/adr/008-phase1-migration-risks.md` — known refactor points for Phase 2-4
- `docs/design-document.md` — complete design specification
