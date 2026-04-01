---
paths:
  - "api/**"
---

# Go Backend Constraints

## Event Sourcing (non-negotiable)
- ALL state mutations go through the event store — never write directly to
  read model tables (tasks, lists, labels, subtasks)
- Events + outbox rows written in a single transaction via `appendWithOutbox()`
- Every event handler/projection must be idempotent: `ON CONFLICT` for inserts,
  plain `UPDATE` for modifications

## Testing
- Table-driven tests for all unit tests — group cases in a slice of structs
  with `name`, `input`, `expected`, run with `t.Run(tc.name, ...)`
- Integration tests use build tag `//go:build integration`
- Run `go vet ./...` after changes to catch issues early

## HLC Timestamps
- Never use `time.Now()` for event timestamps — use `hlc.Clock.Now()`
- `time.Now()` is acceptable only for display timestamps (CompletedAt, DeletedAt)
  that are not used for causal ordering
- HLC timestamps are tracked per-field, not per-entity

## User Scoping
- Every read query must include `WHERE user_id = $1`
- Every write command must verify aggregate ownership via `loadTaskAggregate`

## Error Handling
- Explicit error handling — never `panic()` for recoverable errors
- Always check error return values — never `_ = someFunc()` unless documented

## API Types
- `api/openapi.yaml` is the source of truth for the API contract
- Never edit `*.gen.*` files directly
- When changing the API surface: edit openapi.yaml → `make generate` → update handlers
