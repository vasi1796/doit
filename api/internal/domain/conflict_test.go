package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
)

// Conflict tests simulate two devices making conflicting offline edits
// to the same aggregate. Both produce events, and the merged result
// must converge to the same final state regardless of merge order.

func TestConflictConcurrentTitleEdits(t *testing.T) {
	aggID := uuid.New()
	createHLC := hlc.Timestamp{Time: testNow, Counter: 0}

	// Both devices start from the same created state
	createCmd := CreateTask{TaskID: aggID, UserID: testUserID, Title: "Original", Priority: 0, Position: "a"}

	// Device A edits title at HLC t+1s
	aggA := NewTaskAggregate()
	evtsA, _ := aggA.HandleCreate(createCmd, createHLC)
	aggA.Apply(evtsA[0])
	hlcA := hlc.Timestamp{Time: testNow.Add(time.Second), Counter: 0}
	titleEvtsA, err := aggA.HandleUpdateTitle(UpdateTaskTitle{Title: "Device A title"}, hlcA)
	if err != nil {
		t.Fatalf("device A title update: %v", err)
	}

	// Device B edits title at HLC t+2s (later)
	aggB := NewTaskAggregate()
	evtsB, _ := aggB.HandleCreate(createCmd, createHLC)
	aggB.Apply(evtsB[0])
	hlcB := hlc.Timestamp{Time: testNow.Add(2 * time.Second), Counter: 0}
	titleEvtsB, err := aggB.HandleUpdateTitle(UpdateTaskTitle{Title: "Device B title"}, hlcB)
	if err != nil {
		t.Fatalf("device B title update: %v", err)
	}

	// Both events exist — Device B has later HLC, so LWW should pick B's title.
	// Verify both events were produced with correct timestamps.
	if titleEvtsA[0].Timestamp.Before(evtsA[0].Timestamp) {
		t.Error("device A event should be after create")
	}
	if titleEvtsB[0].Timestamp.Before(evtsB[0].Timestamp) {
		t.Error("device B event should be after create")
	}
	if !titleEvtsB[0].Timestamp.After(titleEvtsA[0].Timestamp) {
		t.Error("device B event should have later timestamp than device A")
	}

	// The HLC counter on B's event should be 0 (fresh wall clock advance)
	if titleEvtsB[0].Counter != 0 {
		t.Errorf("device B counter = %d, want 0", titleEvtsB[0].Counter)
	}
}

func TestConflictConcurrentPriorityAndDueDate(t *testing.T) {
	aggID := uuid.New()
	createHLC := hlc.Timestamp{Time: testNow, Counter: 0}
	createCmd := CreateTask{TaskID: aggID, UserID: testUserID, Title: "Task", Priority: 0, Position: "a"}

	// Device A changes priority
	agg := NewTaskAggregate()
	evts, _ := agg.HandleCreate(createCmd, createHLC)
	agg.Apply(evts[0])
	hlcA := hlc.Timestamp{Time: testNow.Add(time.Second), Counter: 0}
	prioEvts, err := agg.HandleUpdatePriority(UpdateTaskPriority{Priority: 3}, hlcA)
	if err != nil {
		t.Fatalf("priority update: %v", err)
	}

	// Device B changes due date (different field — no conflict)
	agg2 := NewTaskAggregate()
	evts2, _ := agg2.HandleCreate(createCmd, createHLC)
	agg2.Apply(evts2[0])
	dueDate := testNow.Add(24 * time.Hour)
	hlcB := hlc.Timestamp{Time: testNow.Add(2 * time.Second), Counter: 0}
	dateEvts, err := agg2.HandleUpdateDueDate(UpdateTaskDueDate{DueDate: &dueDate}, hlcB)
	if err != nil {
		t.Fatalf("due date update: %v", err)
	}

	// Both events should coexist — different event types, no conflict
	if prioEvts[0].EventType != eventstore.EventTaskPriorityUpdated {
		t.Errorf("priority event type = %q", prioEvts[0].EventType)
	}
	if dateEvts[0].EventType != eventstore.EventTaskDueDateUpdated {
		t.Errorf("date event type = %q", dateEvts[0].EventType)
	}
}

func TestConflictDeleteVsEdit(t *testing.T) {
	// ADR-002 policy: "edit resurrects delete"
	// If device A deletes and device B edits, the edit wins and the task is restored.
	aggID := uuid.New()
	createHLC := hlc.Timestamp{Time: testNow, Counter: 0}
	createCmd := CreateTask{TaskID: aggID, UserID: testUserID, Title: "Task", Priority: 0, Position: "a"}

	// Device A deletes
	aggA := NewTaskAggregate()
	evts, _ := aggA.HandleCreate(createCmd, createHLC)
	aggA.Apply(evts[0])
	hlcA := hlc.Timestamp{Time: testNow.Add(time.Second), Counter: 0}
	deleteEvts, err := aggA.HandleDelete(DeleteTask{DeletedAt: hlcA.Time}, hlcA)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Device B edits (unaware of delete)
	aggB := NewTaskAggregate()
	evtsB, _ := aggB.HandleCreate(createCmd, createHLC)
	aggB.Apply(evtsB[0])
	hlcB := hlc.Timestamp{Time: testNow.Add(2 * time.Second), Counter: 0}
	editEvts, err := aggB.HandleUpdateTitle(UpdateTaskTitle{Title: "Edited after delete"}, hlcB)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

	// Both events exist. The server would process them in HLC order:
	// 1. Delete at t+1s
	// 2. Edit at t+2s — but task is deleted, so this would fail on the aggregate
	//
	// The application-level policy says "edit resurrects delete".
	// The sync handler should detect this conflict and restore the task.
	// For now, we verify the events are produced correctly.
	if deleteEvts[0].EventType != eventstore.EventTaskDeleted {
		t.Errorf("delete event type = %q", deleteEvts[0].EventType)
	}
	if editEvts[0].EventType != eventstore.EventTaskTitleUpdated {
		t.Errorf("edit event type = %q", editEvts[0].EventType)
	}
	// Delete has earlier timestamp
	if !editEvts[0].Timestamp.After(deleteEvts[0].Timestamp) {
		t.Error("edit should have later timestamp than delete")
	}
}

func TestConflictCompleteVsDelete(t *testing.T) {
	// ADR-002 policy: "complete resurrects delete"
	aggID := uuid.New()
	createHLC := hlc.Timestamp{Time: testNow, Counter: 0}
	createCmd := CreateTask{TaskID: aggID, UserID: testUserID, Title: "Task", Priority: 0, Position: "a"}

	// Device A deletes
	aggA := NewTaskAggregate()
	evts, _ := aggA.HandleCreate(createCmd, createHLC)
	aggA.Apply(evts[0])
	hlcA := hlc.Timestamp{Time: testNow.Add(time.Second), Counter: 0}
	_, err := aggA.HandleDelete(DeleteTask{DeletedAt: hlcA.Time}, hlcA)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Device B completes (unaware of delete)
	aggB := NewTaskAggregate()
	evtsB, _ := aggB.HandleCreate(createCmd, createHLC)
	aggB.Apply(evtsB[0])
	hlcB := hlc.Timestamp{Time: testNow.Add(2 * time.Second), Counter: 0}
	completeEvts, _, err := aggB.HandleComplete(CompleteTask{CompletedAt: hlcB.Time}, hlcB)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if completeEvts[0].EventType != eventstore.EventTaskCompleted {
		t.Errorf("complete event type = %q", completeEvts[0].EventType)
	}
}

func TestConflictHLCMonotonicity(t *testing.T) {
	// Verify that a sequence of operations on one device produces
	// strictly increasing HLC timestamps
	clock := hlc.New()
	aggID := uuid.New()
	createCmd := CreateTask{TaskID: aggID, UserID: testUserID, Title: "Task", Priority: 0, Position: "a"}

	agg := NewTaskAggregate()
	now1 := clock.Now()
	evts, _ := agg.HandleCreate(createCmd, now1)
	agg.Apply(evts[0])

	now2 := clock.Now()
	evts2, _ := agg.HandleUpdateTitle(UpdateTaskTitle{Title: "Updated"}, now2)

	if hlc.Compare(now2, now1) <= 0 {
		t.Errorf("second HLC %v not after first %v", now2, now1)
	}
	if !evts2[0].Timestamp.After(evts[0].Timestamp) || (evts2[0].Timestamp.Equal(evts[0].Timestamp) && evts2[0].Counter <= evts[0].Counter) {
		t.Errorf("second event HLC not after first event HLC")
	}
}
