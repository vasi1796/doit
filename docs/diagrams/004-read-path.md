# Read Path — IndexedDB → useLiveQuery → React

How data flows from local storage to the UI. All reads come from IndexedDB — never from the API.

```mermaid
flowchart LR
    subgraph Write["What writes to IndexedDB"]
        IS[Initial Sync]
        OW[Optimistic Writes]
        ME[Remote Merge]
    end

    subgraph IDB["IndexedDB (Dexie.js)"]
        DB[(tasks · lists · labels<br/>taskLabels · subtasks)]
    end

    subgraph Hooks["useLiveQuery Hooks"]
        UT[useTasks]
        UL[useLists]
        ULB[useLabels]
        UTD[useTaskDetail]
    end

    subgraph Pages["React Pages"]
        P[Inbox · Today · Upcoming<br/>List · Label · Completed · Trash]
    end

    IS --> DB
    OW --> DB
    ME --> DB

    DB -->|auto re-render<br/>on change| UT & UL & ULB & UTD

    UT --> P
    UL --> P
    ULB --> P
    UTD --> P
```

## What each hook queries

| Hook | Tables | What it does |
|------|--------|-------------|
| `useTasks` | tasks + taskLabels + subtasks + labels | Filters by list/label/status, sorts by position, joins subtasks and labels |
| `useLists` | lists | All user lists, sorted by position |
| `useLabels` | labels | All user labels |
| `useTaskDetail` | tasks + subtasks + taskLabels + labels | Single task with full subtask and label data |

**Key points:**
- `useLiveQuery` auto-re-renders components when IndexedDB data changes
- No state management libraries (Redux, Zustand, etc.) — Dexie is the sole state layer
- Three sources write to IndexedDB: initial sync, local optimistic writes, and remote event merges
