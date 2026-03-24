# ADR-006: LWW-Register for Markdown Fields

**Status:** Accepted

## Context

Tasks in DoIt have a description field that supports markdown content, edited
via TipTap in the frontend. When multiple devices are offline and edit the same
task's description, we need a strategy to merge those changes.

Options for merging text content:

1. **Operational Transform (OT)**: Track individual character insertions and
   deletions. Complex to implement, designed for real-time collaborative editing.
2. **CRDT text types (Yjs, Automerge)**: Character-level CRDTs that merge
   concurrent text edits. Powerful but add significant dependency and complexity.
3. **LWW-Register**: Treat the entire markdown string as an atomic unit.
   Last write (by HLC timestamp) wins. No character-level merging.

Considerations:
- DoIt serves 1-3 users. Simultaneous editing of the exact same task's
  description field by different users is extremely unlikely.
- Even for a single user across devices, editing the same description field on
  two devices while both are offline is an edge case.
- OT and CRDT text types add substantial implementation complexity and library
  dependencies for a scenario that almost never occurs at this user scale.

## Decision

We will use **LWW-Register** for markdown description fields. The entire
markdown string is stored as a single CRDT unit. When concurrent edits occur,
the edit with the later HLC timestamp wins and the other is discarded.

- The `TaskDescriptionUpdated` event carries the full markdown string.
- On merge, the per-field HLC timestamp for the description field is compared;
  the edit with the later timestamp overwrites the earlier one. Because HLC
  timestamps are tracked per field, a concurrent edit to a different field
  (e.g., title) is not affected.
- No character-level diffing, merging, or operational transforms.

## Consequences

**Benefits:**
- **Simple implementation** — no OT/CRDT text library needed. Just compare HLC
  timestamps and keep the later value.
- **No additional dependencies** — avoids pulling in Yjs, Automerge, or similar
  libraries.
- **Predictable behavior** — the user can reason about what happened (their
  edit either persisted or was overwritten by a newer edit).
- **Consistent with other scalar fields** — same LWW-Register approach used for
  title, due date, status, etc.

**Costs:**
- **Concurrent edits to the same description lose one version** — if User A
  edits a description offline and User B edits the same description offline, one
  edit is silently discarded when they sync. This is the primary tradeoff.
- **No partial merge** — even if the two edits touched different paragraphs, the
  entire string is replaced. Character-level CRDTs would preserve both changes.
- **Scales poorly** — if DoIt ever grew to many concurrent users, this decision
  would need to be revisited. Acceptable for 1-3 users.
