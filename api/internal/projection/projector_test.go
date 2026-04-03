//go:build integration

package projection_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/projection"
)

var testUserID uuid.UUID

func setupTest(t *testing.T) (*projection.Projector, *pgxpool.Pool) {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://doit:changeme@localhost:5432/doit?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	ctx := context.Background()

	// Clean all read model tables in a single statement to avoid deadlocks
	if _, err := pool.Exec(ctx, "TRUNCATE subtasks, task_labels, tasks, labels, lists, aggregate_snapshots, ical_tokens, push_subscriptions, user_config, users, events CASCADE"); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}

	// Insert a test user for FK constraints (unique per test)
	testUserID = uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, google_id, email, name, allowed) VALUES ($1, $2, $3, $4, true)`,
		testUserID, "google-"+testUserID.String(), testUserID.String()+"@test.com", "Test User",
	)
	if err != nil {
		t.Fatalf("inserting test user: %v", err)
	}

	return projection.New(pool, zerolog.Nop()), pool
}

func makeEvent(t *testing.T, aggID uuid.UUID, et eventstore.EventType, aggType eventstore.AggregateType, version int, payload any, ts time.Time) eventstore.Event {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshaling test payload: %v", err)
	}
	return eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   aggID,
		AggregateType: aggType,
		EventType:     et,
		UserID:        testUserID,
		Data:          data,
		Timestamp:     ts,
		Version:       version,
	}
}

func TestProjectTaskCreated(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	taskID := uuid.New()
	listID := uuid.New()

	// Create a list first (FK constraint)
	listEvt := makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
		domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Position: "a"}, now)
	if err := proj.Project(ctx, []eventstore.Event{listEvt}); err != nil {
		t.Fatalf("projecting list: %v", err)
	}

	evt := makeEvent(t, taskID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
		domain.TaskCreatedPayload{Title: "Buy milk", Priority: 2, ListID: &listID, Position: "a"}, now)

	if err := proj.Project(ctx, []eventstore.Event{evt}); err != nil {
		t.Fatalf("projecting: %v", err)
	}

	// Verify
	var title string
	var priority int
	var isCompleted, isDeleted bool
	err := pool.QueryRow(ctx, "SELECT title, priority, is_completed, is_deleted FROM tasks WHERE id = $1", taskID).
		Scan(&title, &priority, &isCompleted, &isDeleted)
	if err != nil {
		t.Fatalf("querying task: %v", err)
	}
	if title != "Buy milk" {
		t.Errorf("title = %q, want %q", title, "Buy milk")
	}
	if priority != 2 {
		t.Errorf("priority = %d, want 2", priority)
	}
	if isCompleted {
		t.Error("is_completed = true, want false")
	}
	if isDeleted {
		t.Error("is_deleted = true, want false")
	}
}

func TestProjectTaskCreatedIdempotent(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	taskID := uuid.New()
	listID := uuid.New()

	listEvt := makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
		domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Position: "a"}, now)
	if err := proj.Project(ctx, []eventstore.Event{listEvt}); err != nil {
		t.Fatalf("projecting list: %v", err)
	}

	evt := makeEvent(t, taskID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
		domain.TaskCreatedPayload{Title: "Buy milk", Priority: 1, ListID: &listID, Position: "a"}, now)

	// Project twice
	if err := proj.Project(ctx, []eventstore.Event{evt}); err != nil {
		t.Fatalf("first projection: %v", err)
	}
	if err := proj.Project(ctx, []eventstore.Event{evt}); err != nil {
		t.Fatalf("second projection (idempotent): %v", err)
	}

	// Verify only one row
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE id = $1", taskID).Scan(&count); err != nil {
		t.Fatalf("counting tasks: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestProjectTaskCompleteFlow(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	taskID := uuid.New()
	listID := uuid.New()

	events := []eventstore.Event{
		makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
			domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Position: "a"}, now),
		makeEvent(t, taskID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
			domain.TaskCreatedPayload{Title: "Buy milk", Priority: 1, ListID: &listID, Position: "a"}, now),
		makeEvent(t, taskID, eventstore.EventTaskCompleted, eventstore.AggregateTypeTask, 2,
			domain.TaskCompletedPayload{CompletedAt: now.Add(time.Hour)}, now.Add(time.Hour)),
	}

	if err := proj.Project(ctx, events); err != nil {
		t.Fatalf("projecting: %v", err)
	}

	var isCompleted bool
	var completedAt time.Time
	err := pool.QueryRow(ctx, "SELECT is_completed, completed_at FROM tasks WHERE id = $1", taskID).
		Scan(&isCompleted, &completedAt)
	if err != nil {
		t.Fatalf("querying task: %v", err)
	}
	if !isCompleted {
		t.Error("is_completed = false, want true")
	}
}

func TestProjectListCreated(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	listID := uuid.New()

	evt := makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
		domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Icon: "briefcase", Position: "a"}, now)

	if err := proj.Project(ctx, []eventstore.Event{evt}); err != nil {
		t.Fatalf("projecting: %v", err)
	}

	var name, colour string
	err := pool.QueryRow(ctx, "SELECT name, colour FROM lists WHERE id = $1", listID).Scan(&name, &colour)
	if err != nil {
		t.Fatalf("querying list: %v", err)
	}
	if name != "Work" {
		t.Errorf("name = %q, want %q", name, "Work")
	}
	if colour != "#ff0000" {
		t.Errorf("colour = %q, want %q", colour, "#ff0000")
	}
}

func TestProjectLabelCreated(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	labelID := uuid.New()

	evt := makeEvent(t, labelID, eventstore.EventLabelCreated, eventstore.AggregateTypeLabel, 1,
		domain.LabelCreatedPayload{Name: "urgent", Colour: "#ff0000"}, now)

	if err := proj.Project(ctx, []eventstore.Event{evt}); err != nil {
		t.Fatalf("projecting: %v", err)
	}

	var name string
	err := pool.QueryRow(ctx, "SELECT name FROM labels WHERE id = $1", labelID).Scan(&name)
	if err != nil {
		t.Fatalf("querying label: %v", err)
	}
	if name != "urgent" {
		t.Errorf("name = %q, want %q", name, "urgent")
	}
}

func TestProjectLabelAddedRemoved(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	taskID := uuid.New()
	listID := uuid.New()
	labelID := uuid.New()

	// Setup: list, label, task
	setup := []eventstore.Event{
		makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
			domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Position: "a"}, now),
		makeEvent(t, labelID, eventstore.EventLabelCreated, eventstore.AggregateTypeLabel, 1,
			domain.LabelCreatedPayload{Name: "urgent", Colour: "#ff0000"}, now),
		makeEvent(t, taskID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
			domain.TaskCreatedPayload{Title: "Test", ListID: &listID, Position: "a"}, now),
	}
	if err := proj.Project(ctx, setup); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Add label
	addEvt := makeEvent(t, taskID, eventstore.EventLabelAdded, eventstore.AggregateTypeTask, 2,
		domain.LabelAddedPayload{LabelID: labelID}, now)
	if err := proj.Project(ctx, []eventstore.Event{addEvt}); err != nil {
		t.Fatalf("adding label: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM task_labels WHERE task_id = $1 AND label_id = $2", taskID, labelID).Scan(&count); err != nil {
		t.Fatalf("querying: %v", err)
	}
	if count != 1 {
		t.Errorf("after add: count = %d, want 1", count)
	}

	// Add again (idempotent)
	if err := proj.Project(ctx, []eventstore.Event{addEvt}); err != nil {
		t.Fatalf("adding label again (idempotent): %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM task_labels WHERE task_id = $1 AND label_id = $2", taskID, labelID).Scan(&count); err != nil {
		t.Fatalf("querying: %v", err)
	}
	if count != 1 {
		t.Errorf("after idempotent add: count = %d, want 1", count)
	}

	// Remove label
	removeEvt := makeEvent(t, taskID, eventstore.EventLabelRemoved, eventstore.AggregateTypeTask, 3,
		domain.LabelRemovedPayload{LabelID: labelID}, now)
	if err := proj.Project(ctx, []eventstore.Event{removeEvt}); err != nil {
		t.Fatalf("removing label: %v", err)
	}

	if err := pool.QueryRow(ctx, "SELECT count(*) FROM task_labels WHERE task_id = $1 AND label_id = $2", taskID, labelID).Scan(&count); err != nil {
		t.Fatalf("querying: %v", err)
	}
	if count != 0 {
		t.Errorf("after remove: count = %d, want 0", count)
	}
}

func TestProjectSubtaskFlow(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	taskID := uuid.New()
	listID := uuid.New()
	subtaskID := uuid.New()

	setup := []eventstore.Event{
		makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
			domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Position: "a"}, now),
		makeEvent(t, taskID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
			domain.TaskCreatedPayload{Title: "Test", ListID: &listID, Position: "a"}, now),
		makeEvent(t, taskID, eventstore.EventSubtaskCreated, eventstore.AggregateTypeTask, 2,
			domain.SubtaskCreatedPayload{SubtaskID: subtaskID, Title: "Sub item", Position: "a"}, now),
	}
	if err := proj.Project(ctx, setup); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Verify subtask created
	var title string
	var isCompleted bool
	err := pool.QueryRow(ctx, "SELECT title, is_completed FROM subtasks WHERE id = $1", subtaskID).Scan(&title, &isCompleted)
	if err != nil {
		t.Fatalf("querying subtask: %v", err)
	}
	if title != "Sub item" {
		t.Errorf("title = %q, want %q", title, "Sub item")
	}
	if isCompleted {
		t.Error("is_completed = true, want false")
	}

	// Complete subtask
	completeEvt := makeEvent(t, taskID, eventstore.EventSubtaskCompleted, eventstore.AggregateTypeTask, 3,
		domain.SubtaskCompletedPayload{SubtaskID: subtaskID, CompletedAt: now}, now)
	if err := proj.Project(ctx, []eventstore.Event{completeEvt}); err != nil {
		t.Fatalf("completing subtask: %v", err)
	}

	err = pool.QueryRow(ctx, "SELECT is_completed FROM subtasks WHERE id = $1", subtaskID).Scan(&isCompleted)
	if err != nil {
		t.Fatalf("querying subtask: %v", err)
	}
	if !isCompleted {
		t.Error("is_completed = false, want true")
	}
}

func TestProjectUnknownEventType(t *testing.T) {
	proj, _ := setupTest(t)
	ctx := context.Background()

	evt := eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "unknown",
		EventType:     "SomeFutureEvent",
		UserID:        testUserID,
		Data:          json.RawMessage(`{}`),
		Timestamp:     time.Now().UTC(),
		Version:       1,
	}

	// Should not error — silently skip unknown events
	if err := proj.Project(ctx, []eventstore.Event{evt}); err != nil {
		t.Fatalf("unexpected error for unknown event: %v", err)
	}
}

func TestProjectListDeletedMovesTasksToInbox(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	listID := uuid.New()
	task1ID := uuid.New()
	task2ID := uuid.New()

	// Setup: create list with two tasks
	setup := []eventstore.Event{
		makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
			domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Position: "a"}, now),
		makeEvent(t, task1ID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
			domain.TaskCreatedPayload{Title: "Task 1", ListID: &listID, Position: "a"}, now),
		makeEvent(t, task2ID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
			domain.TaskCreatedPayload{Title: "Task 2", ListID: &listID, Position: "b"}, now),
	}
	if err := proj.Project(ctx, setup); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Verify tasks are in the list
	var listIDResult *uuid.UUID
	err := pool.QueryRow(ctx, "SELECT list_id FROM tasks WHERE id = $1", task1ID).Scan(&listIDResult)
	if err != nil {
		t.Fatalf("querying task1 list_id: %v", err)
	}
	if listIDResult == nil || *listIDResult != listID {
		t.Fatalf("task1 list_id = %v, want %v", listIDResult, listID)
	}

	// Delete the list
	deleteEvt := makeEvent(t, listID, eventstore.EventListDeleted, eventstore.AggregateTypeList, 2,
		domain.ListDeletedPayload{DeletedAt: now.Add(time.Hour)}, now.Add(time.Hour))
	if err := proj.Project(ctx, []eventstore.Event{deleteEvt}); err != nil {
		t.Fatalf("projecting list delete: %v", err)
	}

	// Tasks should have NULL list_id (moved to inbox)
	for _, taskID := range []uuid.UUID{task1ID, task2ID} {
		err := pool.QueryRow(ctx, "SELECT list_id FROM tasks WHERE id = $1", taskID).Scan(&listIDResult)
		if err != nil {
			t.Fatalf("querying task list_id after delete: %v", err)
		}
		if listIDResult != nil {
			t.Errorf("task %s list_id = %v, want nil (inbox)", taskID, listIDResult)
		}
	}

	// List should be deleted
	var listCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM lists WHERE id = $1", listID).Scan(&listCount); err != nil {
		t.Fatalf("querying list count: %v", err)
	}
	if listCount != 0 {
		t.Errorf("list count = %d, want 0", listCount)
	}
}

func TestProjectLabelDeletedCleansUpAssociations(t *testing.T) {
	proj, pool := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	listID := uuid.New()
	labelID := uuid.New()
	task1ID := uuid.New()
	task2ID := uuid.New()

	// Setup: create list, label, two tasks, attach label to both
	setup := []eventstore.Event{
		makeEvent(t, listID, eventstore.EventListCreated, eventstore.AggregateTypeList, 1,
			domain.ListCreatedPayload{Name: "Work", Colour: "#ff0000", Position: "a"}, now),
		makeEvent(t, labelID, eventstore.EventLabelCreated, eventstore.AggregateTypeLabel, 1,
			domain.LabelCreatedPayload{Name: "urgent", Colour: "#ff0000"}, now),
		makeEvent(t, task1ID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
			domain.TaskCreatedPayload{Title: "Task 1", ListID: &listID, Position: "a"}, now),
		makeEvent(t, task2ID, eventstore.EventTaskCreated, eventstore.AggregateTypeTask, 1,
			domain.TaskCreatedPayload{Title: "Task 2", ListID: &listID, Position: "b"}, now),
		makeEvent(t, task1ID, eventstore.EventLabelAdded, eventstore.AggregateTypeTask, 2,
			domain.LabelAddedPayload{LabelID: labelID}, now),
		makeEvent(t, task2ID, eventstore.EventLabelAdded, eventstore.AggregateTypeTask, 2,
			domain.LabelAddedPayload{LabelID: labelID}, now),
	}
	if err := proj.Project(ctx, setup); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Verify label associations exist
	var assocCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM task_labels WHERE label_id = $1", labelID).Scan(&assocCount); err != nil {
		t.Fatalf("querying task_labels: %v", err)
	}
	if assocCount != 2 {
		t.Fatalf("task_labels count = %d, want 2", assocCount)
	}

	// Delete the label
	deleteEvt := makeEvent(t, labelID, eventstore.EventLabelDeleted, eventstore.AggregateTypeLabel, 2,
		domain.LabelDeletedPayload{DeletedAt: now.Add(time.Hour)}, now.Add(time.Hour))
	if err := proj.Project(ctx, []eventstore.Event{deleteEvt}); err != nil {
		t.Fatalf("projecting label delete: %v", err)
	}

	// task_labels associations should be removed
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM task_labels WHERE label_id = $1", labelID).Scan(&assocCount); err != nil {
		t.Fatalf("querying task_labels after delete: %v", err)
	}
	if assocCount != 0 {
		t.Errorf("task_labels count after delete = %d, want 0", assocCount)
	}

	// Label should be deleted
	var labelCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM labels WHERE id = $1", labelID).Scan(&labelCount); err != nil {
		t.Fatalf("querying label count: %v", err)
	}
	if labelCount != 0 {
		t.Errorf("label count = %d, want 0", labelCount)
	}

	// Tasks should still exist
	var taskCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE id IN ($1, $2)", task1ID, task2ID).Scan(&taskCount); err != nil {
		t.Fatalf("querying task count: %v", err)
	}
	if taskCount != 2 {
		t.Errorf("task count = %d, want 2", taskCount)
	}
}
