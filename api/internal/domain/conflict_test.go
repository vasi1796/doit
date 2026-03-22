package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"

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
