package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
)

var (
	testUserID = uuid.New()
	testNow    = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	testHLC    = hlc.Timestamp{Time: testNow, Counter: 0}
)

func validCreateCmd() CreateTask {
	listID := uuid.New()
	return CreateTask{
		TaskID:   uuid.New(),
		UserID:   testUserID,
		Title:    "Buy groceries",
		Priority: 1,
		ListID:   &listID,
		Position: "a",
	}
}

// makeTaskEvent builds a test event and applies it to the aggregate.
func applyEvents(agg *TaskAggregate, events ...eventstore.Event) {
	for _, e := range events {
		agg.Apply(e)
	}
}

func taskEvent(aggID uuid.UUID, et eventstore.EventType, version int, payload any) eventstore.Event {
	data, _ := json.Marshal(payload) //nolint:errcheck
	return eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   aggID,
		AggregateType: eventstore.AggregateTypeTask,
		EventType:     et,
		UserID:        testUserID,
		Data:          data,
		Timestamp:     testNow.Add(time.Duration(version) * time.Second),
		Version:       version,
	}
}

func TestHandleCreate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     CreateTask
		wantErr error
	}{
		{
			name: "valid task",
			cmd:  validCreateCmd(),
		},
		{
			name:    "empty title",
			cmd:     CreateTask{TaskID: uuid.New(), UserID: testUserID, Title: "", Priority: 0, Position: "a"},
			wantErr: ErrEmptyTitle,
		},
		{
			name:    "priority too high",
			cmd:     CreateTask{TaskID: uuid.New(), UserID: testUserID, Title: "x", Priority: 4, Position: "a"},
			wantErr: ErrInvalidPriority,
		},
		{
			name:    "priority negative",
			cmd:     CreateTask{TaskID: uuid.New(), UserID: testUserID, Title: "x", Priority: -1, Position: "a"},
			wantErr: ErrInvalidPriority,
		},
		{
			name: "priority 0 is valid",
			cmd:  CreateTask{TaskID: uuid.New(), UserID: testUserID, Title: "x", Priority: 0, Position: "a"},
		},
		{
			name: "priority 3 is valid",
			cmd:  CreateTask{TaskID: uuid.New(), UserID: testUserID, Title: "x", Priority: 3, Position: "a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agg := NewTaskAggregate()
			events, err := agg.HandleCreate(tc.cmd, testHLC)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}

			e := events[0]
			if e.EventType != eventstore.EventTaskCreated {
				t.Errorf("EventType = %q, want %q", e.EventType, eventstore.EventTaskCreated)
			}
			if e.AggregateType != eventstore.AggregateTypeTask {
				t.Errorf("AggregateType = %q, want %q", e.AggregateType, eventstore.AggregateTypeTask)
			}
			if e.Version != 1 {
				t.Errorf("Version = %d, want 1", e.Version)
			}
			if e.UserID != tc.cmd.UserID {
				t.Errorf("UserID = %v, want %v", e.UserID, tc.cmd.UserID)
			}

			var payload TaskCreatedPayload
			if err := json.Unmarshal(e.Data, &payload); err != nil {
				t.Fatalf("unmarshaling payload: %v", err)
			}
			if payload.Title != tc.cmd.Title {
				t.Errorf("payload.Title = %q, want %q", payload.Title, tc.cmd.Title)
			}
			if payload.Priority != tc.cmd.Priority {
				t.Errorf("payload.Priority = %d, want %d", payload.Priority, tc.cmd.Priority)
			}
		})
	}
}

func TestHandleCreateAlreadyExists(t *testing.T) {
	agg := NewTaskAggregate()
	cmd := validCreateCmd()
	if _, err := agg.HandleCreate(cmd, testHLC); err != nil {
		t.Fatalf("first create: %v", err)
	}
	agg.Apply(taskEvent(cmd.TaskID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: cmd.Title}))

	_, err := agg.HandleCreate(cmd, testHLC)
	if !errors.Is(err, ErrTaskAlreadyCreated) {
		t.Fatalf("got error %v, want %v", err, ErrTaskAlreadyCreated)
	}
}

func TestStateTransitions(t *testing.T) {
	aggID := uuid.New()
	labelID := uuid.New()
	subtaskID := uuid.New()

	tests := []struct {
		name       string
		setupEvts  []eventstore.Event
		command    func(agg *TaskAggregate) ([]eventstore.Event, error)
		wantErr    error
		wantEvtType eventstore.EventType
	}{
		{
			name: "complete active task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) {
				evts, _, err := agg.HandleComplete(CompleteTask{CompletedAt: testNow}, testHLC)
				return evts, err
			},
			wantEvtType: eventstore.EventTaskCompleted,
		},
		{
			name: "complete already completed task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventTaskCompleted, 2, TaskCompletedPayload{CompletedAt: testNow}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) {
				evts, _, err := agg.HandleComplete(CompleteTask{CompletedAt: testNow}, testHLC)
				return evts, err
			},
			wantErr: ErrTaskAlreadyCompleted,
		},
		{
			name: "complete deleted task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventTaskDeleted, 2, TaskDeletedPayload{DeletedAt: testNow}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) {
				evts, _, err := agg.HandleComplete(CompleteTask{CompletedAt: testNow}, testHLC)
				return evts, err
			},
			wantErr: ErrTaskAlreadyDeleted,
		},
		{
			name: "uncomplete completed task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventTaskCompleted, 2, TaskCompletedPayload{CompletedAt: testNow}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleUncomplete(UncompleteTask{}, testHLC) },
			wantEvtType: eventstore.EventTaskUncompleted,
		},
		{
			name: "uncomplete active task fails",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleUncomplete(UncompleteTask{}, testHLC) },
			wantErr: ErrTaskNotCompleted,
		},
		{
			name: "delete active task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleDelete(DeleteTask{DeletedAt: testNow}, testHLC) },
			wantEvtType: eventstore.EventTaskDeleted,
		},
		{
			name: "delete already deleted task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventTaskDeleted, 2, TaskDeletedPayload{DeletedAt: testNow}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleDelete(DeleteTask{DeletedAt: testNow}, testHLC) },
			wantErr: ErrTaskAlreadyDeleted,
		},
		{
			name: "move active task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleMove(MoveTask{ListID: uuid.New(), Position: "b"}, testHLC) },
			wantEvtType: eventstore.EventTaskMoved,
		},
		{
			name: "move deleted task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventTaskDeleted, 2, TaskDeletedPayload{DeletedAt: testNow}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleMove(MoveTask{ListID: uuid.New(), Position: "b"}, testHLC) },
			wantErr: ErrTaskAlreadyDeleted,
		},
		{
			name: "update description",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleUpdateDescription(UpdateTaskDescription{Description: "new desc"}, testHLC) },
			wantEvtType: eventstore.EventTaskDescriptionUpdated,
		},
		{
			name: "update description on deleted task",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventTaskDeleted, 2, TaskDeletedPayload{DeletedAt: testNow}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleUpdateDescription(UpdateTaskDescription{Description: "new desc"}, testHLC) },
			wantErr: ErrTaskAlreadyDeleted,
		},
		{
			name: "add label",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleAddLabel(AddLabel{LabelID: labelID}, testHLC) },
			wantEvtType: eventstore.EventLabelAdded,
		},
		{
			name: "add duplicate label",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventLabelAdded, 2, LabelAddedPayload{LabelID: labelID}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleAddLabel(AddLabel{LabelID: labelID}, testHLC) },
			wantErr: ErrLabelAlreadyAttached,
		},
		{
			name: "remove label",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventLabelAdded, 2, LabelAddedPayload{LabelID: labelID}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleRemoveLabel(RemoveLabel{LabelID: labelID}, testHLC) },
			wantEvtType: eventstore.EventLabelRemoved,
		},
		{
			name: "remove label not attached",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleRemoveLabel(RemoveLabel{LabelID: labelID}, testHLC) },
			wantErr: ErrLabelNotAttached,
		},
		{
			name: "create subtask",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleCreateSubtask(CreateSubtask{SubtaskID: subtaskID, Title: "sub", Position: "a"}, testHLC) },
			wantEvtType: eventstore.EventSubtaskCreated,
		},
		{
			name: "create subtask empty title",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleCreateSubtask(CreateSubtask{SubtaskID: subtaskID, Title: "", Position: "a"}, testHLC) },
			wantErr: ErrEmptyTitle,
		},
		{
			name: "complete subtask",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventSubtaskCreated, 2, SubtaskCreatedPayload{SubtaskID: subtaskID, Title: "sub", Position: "a"}),
			},
			command:     func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleCompleteSubtask(CompleteSubtask{SubtaskID: subtaskID, CompletedAt: testNow}, testHLC) },
			wantEvtType: eventstore.EventSubtaskCompleted,
		},
		{
			name: "complete subtask not found",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleCompleteSubtask(CompleteSubtask{SubtaskID: subtaskID, CompletedAt: testNow}, testHLC) },
			wantErr: ErrSubtaskNotFound,
		},
		{
			name: "complete subtask already completed",
			setupEvts: []eventstore.Event{
				taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				taskEvent(aggID, eventstore.EventSubtaskCreated, 2, SubtaskCreatedPayload{SubtaskID: subtaskID, Title: "sub", Position: "a"}),
				taskEvent(aggID, eventstore.EventSubtaskCompleted, 3, SubtaskCompletedPayload{SubtaskID: subtaskID, CompletedAt: testNow}),
			},
			command: func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleCompleteSubtask(CompleteSubtask{SubtaskID: subtaskID, CompletedAt: testNow}, testHLC) },
			wantErr: ErrSubtaskAlreadyCompleted,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agg := NewTaskAggregate()
			applyEvents(agg, tc.setupEvts...)

			events, err := tc.command(agg)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}
			if events[0].EventType != tc.wantEvtType {
				t.Errorf("EventType = %q, want %q", events[0].EventType, tc.wantEvtType)
			}
		})
	}
}

func TestVersionTracking(t *testing.T) {
	agg := NewTaskAggregate()
	aggID := uuid.New()

	// Apply 3 events
	applyEvents(agg,
		taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
		taskEvent(aggID, eventstore.EventTaskCompleted, 2, TaskCompletedPayload{CompletedAt: testNow}),
		taskEvent(aggID, eventstore.EventTaskUncompleted, 3, TaskUncompletedPayload{}),
	)

	if agg.Version() != 3 {
		t.Fatalf("Version = %d, want 3", agg.Version())
	}

	// Next event should be version 4
	events, _, err := agg.HandleComplete(CompleteTask{CompletedAt: testNow}, testHLC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events[0].Version != 4 {
		t.Errorf("next event Version = %d, want 4", events[0].Version)
	}
}

func TestCommandOnNonexistentTask(t *testing.T) {
	tests := []struct {
		name    string
		command func(agg *TaskAggregate) ([]eventstore.Event, error)
	}{
		{"complete", func(agg *TaskAggregate) ([]eventstore.Event, error) { e, _, err := agg.HandleComplete(CompleteTask{CompletedAt: testNow}, testHLC); return e, err }},
		{"delete", func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleDelete(DeleteTask{DeletedAt: testNow}, testHLC) }},
		{"move", func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleMove(MoveTask{}, testHLC) }},
		{"update description", func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleUpdateDescription(UpdateTaskDescription{}, testHLC) }},
		{"add label", func(agg *TaskAggregate) ([]eventstore.Event, error) { return agg.HandleAddLabel(AddLabel{}, testHLC) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agg := NewTaskAggregate()
			_, err := tc.command(agg)
			if !errors.Is(err, ErrTaskNotFound) {
				t.Fatalf("got error %v, want %v", err, ErrTaskNotFound)
			}
		})
	}
}
