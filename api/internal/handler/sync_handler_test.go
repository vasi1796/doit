package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"

	"github.com/vasi1796/doit/internal/auth"
)

// --- Test mocks implementing SyncHandler interfaces ---

type mockSyncCommander struct {
	calls  []string
	err    error
}

func (m *mockSyncCommander) CreateTask(_ context.Context, cmd domain.CreateTask) error {
	m.calls = append(m.calls, "CreateTask:"+cmd.Title)
	return m.err
}
func (m *mockSyncCommander) CompleteTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.CompleteTask) error {
	m.calls = append(m.calls, "CompleteTask")
	return m.err
}
func (m *mockSyncCommander) UncompleteTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UncompleteTask) error {
	m.calls = append(m.calls, "UncompleteTask")
	return m.err
}
func (m *mockSyncCommander) DeleteTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.DeleteTask) error {
	m.calls = append(m.calls, "DeleteTask")
	return m.err
}
func (m *mockSyncCommander) RestoreTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.RestoreTask) error {
	m.calls = append(m.calls, "RestoreTask")
	return m.err
}
func (m *mockSyncCommander) MoveTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.MoveTask) error {
	m.calls = append(m.calls, "MoveTask")
	return m.err
}
func (m *mockSyncCommander) ReorderTask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.ReorderTask) error {
	m.calls = append(m.calls, "ReorderTask")
	return m.err
}
func (m *mockSyncCommander) UpdateTaskTitle(_ context.Context, _ uuid.UUID, _ uuid.UUID, cmd domain.UpdateTaskTitle) error {
	m.calls = append(m.calls, "UpdateTaskTitle:"+cmd.Title)
	return m.err
}
func (m *mockSyncCommander) UpdateTaskDescription(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UpdateTaskDescription) error {
	m.calls = append(m.calls, "UpdateTaskDescription")
	return m.err
}
func (m *mockSyncCommander) UpdateTaskPriority(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UpdateTaskPriority) error {
	m.calls = append(m.calls, "UpdateTaskPriority")
	return m.err
}
func (m *mockSyncCommander) UpdateTaskDueDate(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UpdateTaskDueDate) error {
	m.calls = append(m.calls, "UpdateTaskDueDate")
	return m.err
}
func (m *mockSyncCommander) UpdateTaskDueTime(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UpdateTaskDueTime) error {
	m.calls = append(m.calls, "UpdateTaskDueTime")
	return m.err
}
func (m *mockSyncCommander) UpdateTaskRecurrence(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UpdateTaskRecurrence) error {
	m.calls = append(m.calls, "UpdateTaskRecurrence")
	return m.err
}
func (m *mockSyncCommander) AddLabel(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.AddLabel) error {
	m.calls = append(m.calls, "AddLabel")
	return m.err
}
func (m *mockSyncCommander) RemoveLabel(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.RemoveLabel) error {
	m.calls = append(m.calls, "RemoveLabel")
	return m.err
}
func (m *mockSyncCommander) CreateSubtask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.CreateSubtask) error {
	m.calls = append(m.calls, "CreateSubtask")
	return m.err
}
func (m *mockSyncCommander) CompleteSubtask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.CompleteSubtask) error {
	m.calls = append(m.calls, "CompleteSubtask")
	return m.err
}
func (m *mockSyncCommander) UncompleteSubtask(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UncompleteSubtask) error {
	m.calls = append(m.calls, "UncompleteSubtask")
	return m.err
}
func (m *mockSyncCommander) UpdateSubtaskTitle(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.UpdateSubtaskTitle) error {
	m.calls = append(m.calls, "UpdateSubtaskTitle")
	return m.err
}
func (m *mockSyncCommander) CreateList(_ context.Context, cmd domain.CreateList) error {
	m.calls = append(m.calls, "CreateList:"+cmd.Name)
	return m.err
}
func (m *mockSyncCommander) DeleteList(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.DeleteList) error {
	m.calls = append(m.calls, "DeleteList")
	return m.err
}
func (m *mockSyncCommander) CreateLabel(_ context.Context, cmd domain.CreateLabel) error {
	m.calls = append(m.calls, "CreateLabel:"+cmd.Name)
	return m.err
}
func (m *mockSyncCommander) DeleteLabel(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ domain.DeleteLabel) error {
	m.calls = append(m.calls, "DeleteLabel")
	return m.err
}

type mockSyncEventLoader struct {
	events []eventstore.Event
	err    error
}

func (m *mockSyncEventLoader) LoadByUserSince(_ context.Context, _ uuid.UUID, _ time.Time, _ int) ([]eventstore.Event, error) {
	return m.events, m.err
}

type mockSyncClock struct{}

func (m *mockSyncClock) Now() hlc.Timestamp {
	return hlc.Timestamp{Time: time.Now().UTC(), Counter: 0}
}
func (m *mockSyncClock) Update(_ hlc.Timestamp) hlc.Timestamp {
	return hlc.Timestamp{Time: time.Now().UTC(), Counter: 0}
}

type mockSyncSnapshotWriter struct {
	calls []string
}

func (m *mockSyncSnapshotWriter) SaveTaskSnapshot(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	m.calls = append(m.calls, "task")
	return nil
}
func (m *mockSyncSnapshotWriter) SaveListSnapshot(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	m.calls = append(m.calls, "list")
	return nil
}
func (m *mockSyncSnapshotWriter) SaveLabelSnapshot(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	m.calls = append(m.calls, "label")
	return nil
}

func newTestSyncHandler(cmds SyncCommander) *SyncHandler {
	logger := zerolog.Nop()
	return NewSyncHandler(
		cmds,
		&mockSyncEventLoader{},
		&mockSyncClock{},
		NewHub(logger),
		&mockSyncSnapshotWriter{},
		nil, // pool not needed for sync tests
		logger,
	)
}

func doSyncRequest(t *testing.T, handler *SyncHandler, userID uuid.UUID, body syncRequest) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithUserID(req.Context(), userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.Sync(w, req)
	return w
}

func TestSyncCreateTask(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()
	taskID := uuid.New()

	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{
			{
				Type:        "CreateTask",
				AggregateID: taskID.String(),
				Data:        map[string]any{"title": "Test task", "priority": 1, "position": "a"},
				HLCTime:     time.Now().UnixMilli(),
				HLCCounter:  0,
			},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(cmds.calls) != 1 || cmds.calls[0] != "CreateTask:Test task" {
		t.Errorf("calls = %v, want [CreateTask:Test task]", cmds.calls)
	}
}

func TestSyncMultipleOperations(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()

	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{
			{Type: "CreateTask", AggregateID: uuid.New().String(), Data: map[string]any{"title": "Task 1", "priority": 0, "position": "a"}, HLCTime: time.Now().UnixMilli()},
			{Type: "CompleteTask", AggregateID: uuid.New().String(), Data: map[string]any{}, HLCTime: time.Now().UnixMilli()},
			{Type: "CreateList", AggregateID: uuid.New().String(), Data: map[string]any{"name": "Work", "colour": "#ff0000", "position": "a"}, HLCTime: time.Now().UnixMilli()},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(cmds.calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(cmds.calls))
	}
}

func TestSyncInvalidAggregateID(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()

	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{
			{Type: "CreateTask", AggregateID: "not-a-uuid", Data: map[string]any{"title": "Bad", "priority": 0, "position": "a"}, HLCTime: time.Now().UnixMilli()},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (continue on failure)", w.Code)
	}

	var resp syncResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.FailedOps) != 1 || resp.FailedOps[0] != 0 {
		t.Errorf("FailedOps = %v, want [0]", resp.FailedOps)
	}
	if len(cmds.calls) != 0 {
		t.Errorf("calls = %v, want none (operation should be skipped)", cmds.calls)
	}
}

func TestSyncUnknownOperationType(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()

	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{
			{Type: "UnknownOp", AggregateID: uuid.New().String(), Data: map[string]any{}, HLCTime: time.Now().UnixMilli()},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(cmds.calls) != 0 {
		t.Errorf("calls = %v, want none", cmds.calls)
	}
}

func TestSyncUpdateTaskDispatchesFieldCommands(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()
	taskID := uuid.New()

	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{
			{
				Type:        "UpdateTask",
				AggregateID: taskID.String(),
				Data:        map[string]any{"title": "New title", "priority": float64(2)},
				HLCTime:     time.Now().UnixMilli(),
			},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(cmds.calls) != 2 {
		t.Fatalf("calls = %v, want 2 (title + priority)", cmds.calls)
	}
}

func TestSyncResponseIncludesCursor(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()

	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{},
	})

	var resp syncResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Cursor.HLCTime == 0 {
		t.Error("cursor HLC time should be non-zero")
	}
}

func TestSyncUpdateTaskPositionOnlyDispatchesReorder(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()
	taskID := uuid.New()

	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{
			{
				Type:        "UpdateTask",
				AggregateID: taskID.String(),
				Data:        map[string]any{"position": "b"},
				HLCTime:     time.Now().UnixMilli(),
			},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(cmds.calls) != 1 || cmds.calls[0] != "ReorderTask" {
		t.Errorf("calls = %v, want [ReorderTask]", cmds.calls)
	}
}

func TestSyncDispatchOpAllTypes(t *testing.T) {
	labelID := uuid.New()
	subtaskID := uuid.New()

	tests := []struct {
		name        string
		opType      string
		data        map[string]any
		wantCalls   []string
		wantFailed  bool
	}{
		{
			name:      "DeleteTask dispatches DeleteTask",
			opType:    "DeleteTask",
			data:      map[string]any{},
			wantCalls: []string{"DeleteTask"},
		},
		{
			name:      "RestoreTask dispatches RestoreTask",
			opType:    "RestoreTask",
			data:      map[string]any{},
			wantCalls: []string{"RestoreTask"},
		},
		{
			name:      "UncompleteTask dispatches UncompleteTask",
			opType:    "UncompleteTask",
			data:      map[string]any{},
			wantCalls: []string{"UncompleteTask"},
		},
		{
			name:      "AddLabel dispatches AddLabel with label_id",
			opType:    "AddLabel",
			data:      map[string]any{"label_id": labelID.String()},
			wantCalls: []string{"AddLabel"},
		},
		{
			name:       "AddLabel fails with invalid label_id",
			opType:     "AddLabel",
			data:       map[string]any{"label_id": "not-a-uuid"},
			wantCalls:  []string{},
			wantFailed: true,
		},
		{
			name:      "RemoveLabel dispatches RemoveLabel with label_id",
			opType:    "RemoveLabel",
			data:      map[string]any{"label_id": labelID.String()},
			wantCalls: []string{"RemoveLabel"},
		},
		{
			name:       "RemoveLabel fails with invalid label_id",
			opType:     "RemoveLabel",
			data:       map[string]any{"label_id": "bad"},
			wantCalls:  []string{},
			wantFailed: true,
		},
		{
			name:      "CreateSubtask dispatches with subtask_id, title, position",
			opType:    "CreateSubtask",
			data:      map[string]any{"subtask_id": subtaskID.String(), "title": "Sub 1", "position": "a"},
			wantCalls: []string{"CreateSubtask"},
		},
		{
			name:       "CreateSubtask fails with invalid subtask_id",
			opType:     "CreateSubtask",
			data:       map[string]any{"subtask_id": "bad", "title": "Sub 1", "position": "a"},
			wantCalls:  []string{},
			wantFailed: true,
		},
		{
			name:      "CompleteSubtask dispatches with subtask_id",
			opType:    "CompleteSubtask",
			data:      map[string]any{"subtask_id": subtaskID.String()},
			wantCalls: []string{"CompleteSubtask"},
		},
		{
			name:       "CompleteSubtask fails with invalid subtask_id",
			opType:     "CompleteSubtask",
			data:       map[string]any{"subtask_id": "bad"},
			wantCalls:  []string{},
			wantFailed: true,
		},
		{
			name:      "UncompleteSubtask dispatches with subtask_id",
			opType:    "UncompleteSubtask",
			data:      map[string]any{"subtask_id": subtaskID.String()},
			wantCalls: []string{"UncompleteSubtask"},
		},
		{
			name:       "UncompleteSubtask fails with invalid subtask_id",
			opType:     "UncompleteSubtask",
			data:       map[string]any{"subtask_id": "bad"},
			wantCalls:  []string{},
			wantFailed: true,
		},
		{
			name:      "UpdateSubtaskTitle dispatches with subtask_id and title",
			opType:    "UpdateSubtaskTitle",
			data:      map[string]any{"subtask_id": subtaskID.String(), "title": "Updated sub"},
			wantCalls: []string{"UpdateSubtaskTitle"},
		},
		{
			name:       "UpdateSubtaskTitle fails with invalid subtask_id",
			opType:     "UpdateSubtaskTitle",
			data:       map[string]any{"subtask_id": "bad", "title": "Updated sub"},
			wantCalls:  []string{},
			wantFailed: true,
		},
		{
			name:      "CreateList dispatches with name, colour, position",
			opType:    "CreateList",
			data:      map[string]any{"name": "Work", "colour": "#ff0000", "position": "a"},
			wantCalls: []string{"CreateList:Work"},
		},
		{
			name:      "DeleteList dispatches DeleteList",
			opType:    "DeleteList",
			data:      map[string]any{},
			wantCalls: []string{"DeleteList"},
		},
		{
			name:      "CreateLabel dispatches with name, colour",
			opType:    "CreateLabel",
			data:      map[string]any{"name": "Urgent", "colour": "#ff0000"},
			wantCalls: []string{"CreateLabel:Urgent"},
		},
		{
			name:      "DeleteLabel dispatches DeleteLabel",
			opType:    "DeleteLabel",
			data:      map[string]any{},
			wantCalls: []string{"DeleteLabel"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmds := &mockSyncCommander{}
			h := newTestSyncHandler(cmds)
			userID := uuid.New()
			aggID := uuid.New()

			w := doSyncRequest(t, h, userID, syncRequest{
				Operations: []syncOperation{
					{
						Type:        tc.opType,
						AggregateID: aggID.String(),
						Data:        tc.data,
						HLCTime:     time.Now().UnixMilli(),
					},
				},
			})

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}

			var resp syncResponse
			json.NewDecoder(w.Body).Decode(&resp)

			if tc.wantFailed {
				if len(resp.FailedOps) == 0 {
					t.Errorf("expected operation to fail, but FailedOps is empty")
				}
				if len(cmds.calls) != 0 {
					t.Errorf("calls = %v, want none for failed operation", cmds.calls)
				}
				return
			}

			if len(resp.FailedOps) != 0 {
				t.Errorf("FailedOps = %v, want none", resp.FailedOps)
			}

			if len(cmds.calls) != len(tc.wantCalls) {
				t.Fatalf("calls = %v, want %v", cmds.calls, tc.wantCalls)
			}
			for i, want := range tc.wantCalls {
				if cmds.calls[i] != want {
					t.Errorf("calls[%d] = %q, want %q", i, cmds.calls[i], want)
				}
			}
		})
	}
}

func TestSyncUpdateTaskFieldCombinations(t *testing.T) {
	listID := uuid.New()

	tests := []struct {
		name      string
		data      map[string]any
		wantCalls []string
	}{
		{
			name:      "description only",
			data:      map[string]any{"description": "A detailed description"},
			wantCalls: []string{"UpdateTaskDescription"},
		},
		{
			name:      "due_date only with valid date",
			data:      map[string]any{"due_date": "2026-04-15"},
			wantCalls: []string{"UpdateTaskDueDate"},
		},
		{
			name:      "due_time only",
			data:      map[string]any{"due_time": "14:30"},
			wantCalls: []string{"UpdateTaskDueTime"},
		},
		{
			name:      "recurrence_rule only",
			data:      map[string]any{"recurrence_rule": "FREQ=DAILY;INTERVAL=1"},
			wantCalls: []string{"UpdateTaskRecurrence"},
		},
		{
			name:      "list_id + position dispatches MoveTask",
			data:      map[string]any{"list_id": listID.String(), "position": "c"},
			wantCalls: []string{"MoveTask"},
		},
		{
			name:      "clearing due_date with null value",
			data:      map[string]any{"due_date": nil},
			wantCalls: []string{"UpdateTaskDueDate"},
		},
		{
			name:      "clearing due_time with null value",
			data:      map[string]any{"due_time": nil},
			wantCalls: []string{"UpdateTaskDueTime"},
		},
		{
			name:      "description + due_date + priority together",
			data:      map[string]any{"description": "desc", "due_date": "2026-05-01", "priority": float64(3)},
			wantCalls: []string{"UpdateTaskDescription", "UpdateTaskPriority", "UpdateTaskDueDate"},
		},
		{
			name:      "title + recurrence_rule + due_time",
			data:      map[string]any{"title": "Updated", "recurrence_rule": "FREQ=WEEKLY", "due_time": "09:00"},
			wantCalls: []string{"UpdateTaskTitle:Updated", "UpdateTaskDueTime", "UpdateTaskRecurrence"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmds := &mockSyncCommander{}
			h := newTestSyncHandler(cmds)
			userID := uuid.New()
			taskID := uuid.New()

			w := doSyncRequest(t, h, userID, syncRequest{
				Operations: []syncOperation{
					{
						Type:        "UpdateTask",
						AggregateID: taskID.String(),
						Data:        tc.data,
						HLCTime:     time.Now().UnixMilli(),
					},
				},
			})

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}

			var resp syncResponse
			json.NewDecoder(w.Body).Decode(&resp)

			if len(resp.FailedOps) != 0 {
				t.Errorf("FailedOps = %v, want none", resp.FailedOps)
			}

			if len(cmds.calls) != len(tc.wantCalls) {
				t.Fatalf("calls = %v, want %v", cmds.calls, tc.wantCalls)
			}

			// Check that all expected calls are present (order may vary for some fields)
			callSet := make(map[string]bool)
			for _, c := range cmds.calls {
				callSet[c] = true
			}
			for _, want := range tc.wantCalls {
				if !callSet[want] {
					t.Errorf("expected call %q not found in %v", want, cmds.calls)
				}
			}
		})
	}
}

func TestSyncMalformedDataDoesNotPanic(t *testing.T) {
	cmds := &mockSyncCommander{}
	h := newTestSyncHandler(cmds)
	userID := uuid.New()

	// Send a CreateTask with priority as string instead of number — should not panic
	w := doSyncRequest(t, h, userID, syncRequest{
		Operations: []syncOperation{
			{
				Type:        "CreateTask",
				AggregateID: uuid.New().String(),
				Data:        map[string]any{"title": 12345, "priority": "not-a-number", "position": nil},
				HLCTime:     time.Now().UnixMilli(),
			},
		},
	})

	// Should not panic — strVal/intVal handle wrong types gracefully
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (should not panic)", w.Code)
	}
}
