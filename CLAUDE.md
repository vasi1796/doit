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
    crdt/          CRDT merge functions (LWW-Register, OR-Set, Fractional Indexing)
    domain/        Aggregates, commands, payloads, CommandHandler
    eventstore/    Event store (append/load/query, HLC counter)
    handler/       HTTP handlers (task, list, label, auth, sync, response utils)
    hlc/           Hybrid Logical Clock (causal ordering for sync)
    middleware/     JWT auth middleware
    projection/    Event → read model table updates
  migrations/     SQL migration files
  openapi.yaml    API contract (source of truth for Go + TS type generation)
web/              React frontend
  src/
    api/           Typed fetch client + generated types (from OpenAPI spec)
    crdt/          CRDT merge functions (TypeScript, mirrors Go)
    components/    Common pickers, layout (sidebar/bottom nav), task components
    db/            Dexie.js database, operations, sync engine, event merger
    hlc/           Hybrid Logical Clock (TypeScript, mirrors Go)
    hooks/         Dexie.js useLiveQuery hooks (useTasks, useLists, useLabels, useTaskDetail)
    pages/         Route pages (Inbox, Today, Upcoming, List, Label, Completed, Trash, Login)
    constants.ts   Shared color palette
  e2e/            Playwright visual regression + accessibility tests
  public/sw.js    Service worker (app shell caching for offline launch)
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

### 3. Frontend state comes from Dexie.js useLiveQuery
All data reads use `useLiveQuery` from Dexie.js (IndexedDB). Do NOT introduce
Redux, Zustand, Jotai, or any other state library. React Context is used only
for layout-level computed data (task counts, quick-add ref).

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

### 7. API types are generated from OpenAPI spec — never hand-edit
`api/openapi.yaml` is the single source of truth for the API contract.
Go response types (`openapi_types.gen.go`) and TypeScript types (`types.gen.ts`)
are both generated from it. Never edit `*.gen.*` files directly.
When changing the API surface:
1. Edit `api/openapi.yaml`
2. Run `make generate`
3. Update handler code to match the new generated types

### 8. All timestamps use HLC — never use time.Now() directly
The `hlc.Clock` provides causal ordering for CRDT conflict resolution.
Server: `CommandHandler` owns the clock, all Handle* methods accept `hlc.Timestamp`.
Client: `web/src/db/clock.ts` exports the singleton clock.
Never call `time.Now()` for event timestamps — always use the HLC clock.

### 9. Frontend reads from Dexie.js — never from the API
All UI reads come from IndexedDB via `useLiveQuery`. The `api/client.ts` is used
only by `initial-sync.ts` (initial load) and the sync engine. Components must
never call `api.*` directly — use `db/operations.ts` for writes.

### 10. Writes go to IndexedDB + sync queue — not to the API
`db/operations.ts` writes optimistically to IndexedDB and queues a `SyncOp`.
The `SyncEngine` flushes the queue to `POST /api/v1/sync` periodically.
No rollback on failure — the sync engine retries with exponential backoff.

---

## Test Expectations

- **Unit tests** for all business logic (domain, event handling, config, auth).
- **Table-driven tests** in Go — no exceptions.
- **Integration tests** (build tag `//go:build integration`) for event store
  and projection flows against real Postgres.
- **Visual regression tests** (Playwright + WebKit) compare page screenshots
  against committed baselines. Run with `cd web && npm run test:visual`.
- **Accessibility tests** (Playwright + axe-core) scan all pages for WCAG 2.0
  AA violations and 16px input font sizes.
- **ESLint jsx-a11y** plugin enforces accessibility rules at lint time.
- **No flaky tests.** E2E tests must pass deterministically without retries.
  Do not use `waitForTimeout` as a fix for race conditions — wait for specific
  elements or URLs. Do not add retries to mask flakiness. If a test cannot be
  made deterministic, remove it and document why.

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

# Regenerate Go + TypeScript types from OpenAPI spec
make generate

# Build frontend
cd web && npm run build

# Run frontend lint (includes jsx-a11y)
cd web && npm run lint

# Run all frontend E2E tests (visual + a11y)
cd web && npm run test:e2e

# Run visual regression tests only
cd web && npm run test:visual

# Update visual regression baselines after intentional UI changes
cd web && npm run test:visual:update

# Run accessibility tests only
cd web && npm run test:a11y

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
