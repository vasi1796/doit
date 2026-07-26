# ADR-002: CRDT Type Choices for Offline Sync

**Status:** Accepted

## Context

DoIt is an offline-first PWA used by 1-3 users across multiple devices (iPhone,
iPad, Mac). When devices are offline, users can make changes independently. When
connectivity returns, those changes must be merged without data loss or manual
conflict resolution wherever possible.

We need to choose CRDT (Conflict-free Replicated Data Type) strategies for each
type of data in the system:

- Scalar fields: task title, due date, status, priority, description
- Set-like fields: labels on a task
- Ordered collections: task ordering within a list, subtask ordering
- Timestamps: causal ordering of operations across devices

## Decision

We will use the following CRDT strategies:

| Data Type | CRDT | Rationale |
|-----------|------|-----------|
| Scalar fields (title, due date, status, priority) | **LWW-Register** | Simple, deterministic, per-field HLC tracking, acceptable for 1-3 users |
| Markdown description | **LWW-Register** | Whole-string replacement (see ADR-006) |
| Labels on a task | **OR-Set** (Observed-Remove Set) | Correctly handles concurrent add/remove of the same label |
| Task ordering within a list | **Fractional Indexing** | Allows insertion between any two items without reindexing |
| Subtask ordering | **Fractional Indexing** | Same rationale as task ordering |
| Timestamps | **HLC** (Hybrid Logical Clock) | Provides causal ordering while staying close to wall-clock time |

### Application-Level Conflict Policies

Beyond CRDT mechanics, we define these application-level policies:

- **Edit resurrects delete**: If one device deletes a task and another edits it
  concurrently, the edit wins. The task is restored with the edit applied.
- **Complete resurrects delete**: If one device deletes a task and another marks
  it complete, the completion wins. The task is restored as completed.
- **Concurrent list moves**: If two devices move the same task to different lists
  concurrently, Last-Writer-Wins based on HLC timestamp.

## Consequences

**Benefits:**
- LWW-Register is simple to implement and reason about.
- OR-Set correctly handles the concurrent add/remove edge case for labels.
- Fractional indexing avoids expensive reindexing operations on reorder.
- HLC provides a good balance of causality tracking and simplicity.
- Application-level policies (edit resurrects delete) match user expectations.

**Costs:**
- **LWW can silently lose concurrent edits to the same field** — if two users
  edit the same scalar field at the same time, one edit is discarded. HLC
  timestamps are tracked per field (not per task), so concurrent edits to
  *different* fields on the same task are both preserved. This is an acceptable
  tradeoff for 1-3 users where simultaneous editing of the exact same field is rare.
- Fractional indexing keys can grow long after many insertions between the same
  two items (mitigated by periodic rebalancing).
- HLC adds clock management complexity compared to simple wall-clock timestamps.
- OR-Set metadata (tombstones, observed set) adds storage overhead for labels.

## Addendum (2026-07-24): per-field LWW extended to lists and labels

Per-field LWW-Register was originally applied only to task scalar fields.
With the list/label edit feature (PRs #23-#25) the same mechanism now covers
list and label `name` and `colour`:

- Client rows for lists/labels carry a `field_hlcs` map (as tasks do), so a
  rename on one device and a recolour on another both survive sync.
- Granular events (`ListNameUpdated`, `ListColourUpdated`, `LabelNameUpdated`,
  `LabelColourUpdated`) mirror the task field-event pattern.
- The `UpdateList`/`UpdateLabel` sync ops are dispatched as **one atomic
  command**: every changed field is validated before any event is appended,
  and all resulting events are appended in a single transaction. This
  prevents a partially-applied edit from being retried into duplicate
  events or silently dropping one field.
- Redelivered `ListCreated`/`LabelCreated` events merge through the same
  per-field LWW path instead of overwriting the whole record, so at-least-once
  event delivery cannot revert local unsynced edits.

Rejected alternative: a single combined `ListUpdated` event carrying
name+colour — simpler, but a concurrent rename and recolour would resolve to
one winner, violating the per-field preservation principle above.

## Addendum (2026-07-26): ordering CRDT extended to sidebar lists and labels

The fractional-indexing decision above originally covered only task and
subtask ordering. With the sidebar reorder feature the same strategy now
covers the user-defined order of lists and labels in the sidebar:

- `lists.position` (which existed since Phase 1 but was write-once at
  creation) and a new `labels.position` column (migration 009) hold
  fractional-index strings; concurrent inserts at the same slot resolve by
  per-field LWW on `position`, consistent with the "concurrent list moves"
  policy above.
- Granular `ListReordered`/`LabelReordered` events mirror the task
  field-event pattern and ride the existing atomic `UpdateList`/`UpdateLabel`
  commands, so a rename on one device and a reorder on another both survive
  (`field_hlcs['position']` on client rows).
- Existing labels are backfilled in alphabetical order twice with one
  deterministic formula — SQL in migration 009 for the server, a Dexie v5
  upgrade for already-installed clients — because no events exist for the
  backfill; the two sides converge without sync. Both sides compare by raw
  code units (`COLLATE "C"` in SQL via migration 010, code-unit `<` in JS)
  so the order does not depend on the database locale. Rows that predate the
  backfill (e.g. a projection rebuild replaying historical `LabelCreated`
  events) have NULL positions, so label read paths order by position with a
  name fallback rather than requiring NOT NULL.

Rejected alternative: dedicated `ReorderList`/`ReorderLabel` sync operations
mirroring `ReorderTask` — behaviourally identical but duplicates dispatch,
interface, and mock plumbing for no per-field-LWW gain; position as a third
LWW field on the existing atomic update ops was chosen instead.
