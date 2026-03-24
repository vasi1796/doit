# Task Lifecycle — State Machine

Valid states and transitions for a task aggregate.

```mermaid
stateDiagram-v2
    [*] --> Active: TaskCreated

    Active --> Completed: TaskCompleted
    Active --> Deleted: TaskDeleted

    Completed --> Active: TaskUncompleted
    Completed --> Deleted: TaskDeleted

    Deleted --> Active: TaskRestored

    state Active {
        [*] --> Ready
        Ready --> Ready: TaskTitleUpdated
        Ready --> Ready: TaskDescriptionUpdated
        Ready --> Ready: TaskPriorityUpdated
        Ready --> Ready: TaskDueDateUpdated
        Ready --> Ready: TaskDueTimeUpdated
        Ready --> Ready: TaskRecurrenceUpdated
        Ready --> Ready: TaskMoved (list + position)
        Ready --> Ready: LabelAdded / LabelRemoved
        Ready --> Ready: SubtaskCreated / Updated / Completed
    }

    note right of Completed
        If task has recurrence_rule + due_date,
        the recurring worker creates a new
        task with the next due date.
    end note

    note right of Deleted
        Soft delete — task stays in event store.
        Can be restored at any time.
    end note
```

## Conflict Resolution Policies

```mermaid
flowchart LR
    subgraph Concurrent["Concurrent Operations"]
        A["Device A: Edit task"]
        B["Device B: Delete task"]
    end

    A --> R{Resolve}
    B --> R

    R -->|"Edit resurrects delete"| W1["Task restored with edit applied"]

    subgraph LWW["Scalar Field Conflicts"]
        C["Device A: title='Buy milk'<br/>HLC: t=100, c=1"]
        D["Device B: title='Buy eggs'<br/>HLC: t=100, c=2"]
    end

    C --> R2{Compare HLC}
    D --> R2
    R2 -->|"Higher counter wins"| W2["title = 'Buy eggs'"]

    subgraph ORSet["Label Conflicts"]
        E["Device A: Add 'Urgent'"]
        F["Device B: Remove 'Urgent'<br/>(different tag)"]
    end

    E --> R3{OR-Set merge}
    F --> R3
    R3 -->|"Add survives<br/>(different tags)"| W3["'Urgent' label present"]
```

**Policies:**
- **Edit vs Delete** → edit wins (task restored)
- **Complete vs Delete** → complete wins (task restored as completed)
- **Concurrent scalar edits** → Last-Writer-Wins by per-field HLC timestamp
- **Concurrent label add/remove** → OR-Set semantics (add survives if different operation tags)
- **Concurrent list moves** → LWW by HLC timestamp

---

## Per-Field HLC Merge

How concurrent edits to **different fields** on different devices are both preserved.

```mermaid
sequenceDiagram
    participant A as Device A
    participant IDB_A as IndexedDB A
    participant Server
    participant IDB_B as IndexedDB B
    participant B as Device B

    Note over A,B: Task "Buy groceries" — priority=0, title="Buy groceries"

    A->>IDB_A: Update title → "Buy milk"<br/>field_hlcs.title = {t:100, c:1}
    B->>IDB_B: Update priority → 3 (High)<br/>field_hlcs.priority = {t:101, c:1}

    A->>Server: Sync: TaskTitleUpdated (HLC 100:1)
    B->>Server: Sync: TaskPriorityUpdated (HLC 101:1)

    Server-->>A: Pull: TaskPriorityUpdated (HLC 101:1)
    Server-->>B: Pull: TaskTitleUpdated (HLC 100:1)

    Note over IDB_A: mergeTaskField("priority", HLC 101:1)<br/>field_hlcs.priority is empty → remote wins<br/>✅ priority = 3

    Note over IDB_B: mergeTaskField("title", HLC 100:1)<br/>field_hlcs.title is empty → remote wins<br/>✅ title = "Buy milk"

    Note over A,B: Both devices: title="Buy milk", priority=3<br/>No data lost!
```

```mermaid
flowchart TD
    Event["Remote event:<br/>TaskTitleUpdated<br/>HLC = {t:100, c:2}"]
    Event --> Load["Load task from IndexedDB"]
    Load --> CheckField{"field_hlcs.title<br/>exists?"}

    CheckField -->|Yes| Compare{"Compare HLCs:<br/>remote vs local field"}
    CheckField -->|No| FallbackCompare{"Compare HLCs:<br/>remote vs task-level<br/>(backward compat)"}

    Compare -->|Remote wins| Apply["Update title +<br/>field_hlcs.title"]
    Compare -->|Local wins| Skip["Skip — local is newer"]

    FallbackCompare -->|Remote wins| Apply
    FallbackCompare -->|Local wins| Skip
```

**Key points:**
- Each field has its own HLC timestamp in `field_hlcs` map
- Concurrent edits to **different** fields are both preserved (no conflict)
- Concurrent edits to the **same** field → LWW (higher HLC wins, other is lost)
- Backward compatible: tasks without `field_hlcs` fall back to task-level HLC
- `TaskCreated` initializes all field HLCs to the creation event's HLC
