# Read Path — IndexedDB → useLiveQuery → React

How data flows from local storage to the UI. All reads come from IndexedDB — never from the API.

```mermaid
flowchart TB
    subgraph Sources["Data Sources (write to IndexedDB)"]
        IS[Initial Sync<br/>REST → bulkPut]
        OW[Optimistic Write<br/>operations.ts → put]
        ME[Remote Merge<br/>merge-events.ts → LWW update]
    end

    subgraph Dexie["Dexie.js (IndexedDB)"]
        T[(tasks)]
        L[(lists)]
        LB[(labels)]
        TL[(taskLabels)]
        ST[(subtasks)]
    end

    subgraph Hooks["React Hooks"]
        UT[useTasks<br/>filter + sort + join]
        UL[useLists]
        ULB[useLabels]
        UTD[useTaskDetail]
    end

    subgraph UI["React Components"]
        Pages[InboxPage / TodayPage /<br/>UpcomingPage / ListPage /<br/>LabelPage / CompletedPage /<br/>TrashPage]
    end

    IS --> T & L & LB & ST & TL
    OW --> T & ST & TL
    ME --> T & L & LB & ST & TL

    T --> UT
    TL --> UT
    ST --> UT
    LB --> UT
    L --> UL
    LB --> ULB
    T & ST & TL & LB --> UTD

    UT --> Pages
    UL --> Pages
    ULB --> Pages
    UTD --> Pages

    style Sources fill:#f0f4ff,stroke:#4a90d9
    style Dexie fill:#fff3e0,stroke:#f5a623
    style Hooks fill:#e8f5e9,stroke:#4caf50
    style UI fill:#fce4ec,stroke:#e91e63
```

**Key points:**
- `useLiveQuery` auto-re-renders components when IndexedDB data changes
- No state management libraries (Redux, Zustand, etc.) — Dexie is the sole state layer
- Hooks compose data: `useTasks` joins tasks + subtasks + labels in one query
- Three sources write to IndexedDB: initial sync, local writes, and remote merges
