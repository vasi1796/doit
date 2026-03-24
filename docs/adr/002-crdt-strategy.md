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
