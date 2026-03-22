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
- **Concurrent scalar edits** → Last-Writer-Wins by HLC timestamp
- **Concurrent label add/remove** → OR-Set semantics (add survives if different operation tags)
- **Concurrent list moves** → LWW by HLC timestamp
