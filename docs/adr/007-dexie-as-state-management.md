# ADR-007: Dexie.js + useLiveQuery as Sole State Management

**Status:** Accepted

## Context

DoIt is an offline-first PWA where IndexedDB is the local source of truth on
the client. The frontend needs a reactive state management approach that:

- Reflects the current state of IndexedDB at all times.
- Automatically re-renders React components when underlying data changes.
- Works offline without a server connection.
- Does not introduce a separate state layer that could diverge from IndexedDB.

Common React state management options:
1. **Redux / Zustand / Jotai** — external state stores that would need to be
   kept in sync with IndexedDB, creating a dual-source-of-truth problem.
2. **React Context + useReducer** — lightweight but still a separate state layer.
3. **Dexie.js + useLiveQuery** — Dexie wraps IndexedDB with a Promise-based API;
   `useLiveQuery` is a React hook that subscribes to Dexie queries and
   automatically re-renders when the underlying data changes.

## Decision

We will use **Dexie.js** as the IndexedDB wrapper and **`useLiveQuery`** as the
sole mechanism for reading state in React components. No additional state
management library will be introduced.

The data flow is strictly unidirectional:

```
User Action
    |
    v
Dexie.js write (db.tasks.put(...))
    |
    v
IndexedDB updated
    |
    v
useLiveQuery detects change
    |
    v
React component re-renders with new data
```

Rules:
- All reads go through `useLiveQuery(() => db.tasks.where(...).toArray())`.
- All writes go through Dexie's API (`db.tasks.put()`, `db.tasks.add()`, etc.).
- No Redux, Zustand, Jotai, MobX, or React Context for data state.
- React Context may be used for non-data concerns (e.g., theme, auth token) but
  never for task/list/label data.

## Consequences

**Benefits:**
- **Single source of truth** — IndexedDB is the only place data lives on the
  client. No risk of state divergence between a store and IndexedDB.
- **Simpler architecture** — one fewer layer to maintain and debug. No actions,
  reducers, selectors, or store configuration.
- **Automatic reactivity** — `useLiveQuery` handles subscriptions and
  re-rendering automatically. No manual cache invalidation.
- **Offline-first by default** — since all reads come from IndexedDB, the UI
  works identically whether online or offline.
- **Sync integration** — the sync engine writes to IndexedDB, and the UI
  automatically reflects changes without explicit notification.

**Costs:**
- **All state access is async** — `useLiveQuery` returns `undefined` on the
  first render while the IndexedDB query resolves. Components must handle this
  loading state.
- **IndexedDB performance** — complex queries may be slower than in-memory state
  stores. Mitigated by Dexie's indexing and the small data volume (personal task
  manager).
- **Less ecosystem tooling** — Redux DevTools and similar debugging tools are not
  available. Dexie has its own debugging capabilities but they are less mature.
- **Tight coupling to Dexie** — switching away from Dexie would require
  significant refactoring. Acceptable because IndexedDB is a fundamental
  architectural choice, not a swappable library.
