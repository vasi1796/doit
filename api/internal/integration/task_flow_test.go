//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/domain"
)

func TestCreateTaskFlow(t *testing.T) {
	// Full pipeline: CommandHandler → event store + outbox → outbox poller →
	// RabbitMQ → projection worker → task appears in read model.
	h := setupHarness(t)
	ctx := context.Background()

	taskID := uuid.New()
	dueDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	dueTime := "14:30"

	// 1. Create task via CommandHandler (writes events + outbox atomically)
	err := h.cmdHandler.CreateTask(ctx, domain.CreateTask{
		TaskID:   taskID,
		UserID:   h.userID,
		Title:    "Buy groceries",
		Priority: domain.PriorityMedium,
		DueDate:  &dueDate,
		DueTime:  &dueTime,
		Position: "a",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// 2. Verify events were stored
	events, err := h.store.LoadByAggregate(ctx, taskID)
	if err != nil {
		t.Fatalf("LoadByAggregate: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != "TaskCreated" {
		t.Errorf("expected TaskCreated, got %s", events[0].EventType)
	}

	// 3. Flush outbox → publishes to RabbitMQ
	h.flushOutbox(t)

	// 4. Drain projections queue → projector updates read model
	projected := h.drainProjections(t)
	if projected != 1 {
		t.Fatalf("expected 1 projected event, got %d", projected)
	}

	// 5. Verify task appears in read model
	var title string
	var priority int
	var readDueDate *time.Time
	var readDueTime *string
	var isCompleted bool
	var isDeleted bool
	err = h.pool.QueryRow(ctx,
		`SELECT title, priority, due_date, due_time, is_completed, is_deleted FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, h.userID,
	).Scan(&title, &priority, &readDueDate, &readDueTime, &isCompleted, &isDeleted)
	if err != nil {
		t.Fatalf("reading task from read model: %v", err)
	}

	if title != "Buy groceries" {
		t.Errorf("title = %q, want %q", title, "Buy groceries")
	}
	if priority != int(domain.PriorityMedium) {
		t.Errorf("priority = %d, want %d", priority, domain.PriorityMedium)
	}
	if readDueDate == nil || readDueDate.Format("2006-01-02") != "2026-04-01" {
		t.Errorf("due_date = %v, want 2026-04-01", readDueDate)
	}
	if readDueTime == nil || (*readDueTime)[:5] != dueTime {
		got := "<nil>"
		if readDueTime != nil {
			got = *readDueTime
		}
		t.Errorf("due_time = %s, want %s", got, dueTime)
	}
	if isCompleted {
		t.Error("expected is_completed = false")
	}
	if isDeleted {
		t.Error("expected is_deleted = false")
	}
}

func TestRecurringTaskFlow(t *testing.T) {
	// Full pipeline: Create recurring task with labels → complete it →
	// outbox → RabbitMQ → recurring worker creates next occurrence →
	// outbox → RabbitMQ → projection worker updates read model →
	// next task has recurrence_rule + labels.
	h := setupHarness(t)
	ctx := context.Background()

	// Setup: create a label first
	labelID := uuid.New()
	err := h.cmdHandler.CreateLabel(ctx, domain.CreateLabel{
		LabelID: labelID,
		UserID:  h.userID,
		Name:    "Errands",
		Colour:  "#ff9500",
	})
	if err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}

	// 1. Create a recurring task with a label
	taskID := uuid.New()
	dueDate := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)

	err = h.cmdHandler.CreateTask(ctx, domain.CreateTask{
		TaskID:   taskID,
		UserID:   h.userID,
		Title:    "Water plants",
		Priority: domain.PriorityLow,
		DueDate:  &dueDate,
		Position: "a",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Set recurrence rule
	err = h.cmdHandler.UpdateTaskRecurrence(ctx, taskID, h.userID, domain.UpdateTaskRecurrence{
		RecurrenceRule: domain.RecurrenceWeekly,
	})
	if err != nil {
		t.Fatalf("UpdateTaskRecurrence: %v", err)
	}

	// Add label to task
	err = h.cmdHandler.AddLabel(ctx, taskID, h.userID, domain.AddLabel{LabelID: labelID})
	if err != nil {
		t.Fatalf("AddLabel: %v", err)
	}

	// Flush and project the creation events so read models are current
	h.flushOutbox(t)
	h.drainProjections(t)

	// Also drain recurring queue for creation events (they won't match TaskCompleted)
	h.drainRecurring(t)

	// 2. Complete the task
	err = h.cmdHandler.CompleteTask(ctx, taskID, h.userID, domain.CompleteTask{
		CompletedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// 3. Flush outbox → publishes TaskCompleted to RabbitMQ
	h.flushOutbox(t)

	// 4. Drain recurring queue → worker creates next occurrence via CommandHandler
	//    (this also writes new events + outbox rows)
	recurring := h.drainRecurring(t)
	if recurring != 1 {
		t.Fatalf("expected 1 recurring event processed, got %d", recurring)
	}

	// 5. Flush outbox again → new task's events get published
	h.flushOutbox(t)

	// 6. Drain projections → all events (completion + new task creation) projected
	h.drainProjections(t)

	// 7. Verify: the original task is completed
	var isCompleted bool
	err = h.pool.QueryRow(ctx,
		`SELECT is_completed FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, h.userID,
	).Scan(&isCompleted)
	if err != nil {
		t.Fatalf("reading original task: %v", err)
	}
	if !isCompleted {
		t.Error("original task should be completed")
	}

	// 8. Verify: a new task exists with the next due date, recurrence rule, and label
	var newTaskID uuid.UUID
	var newTitle string
	var newDueDate *time.Time
	var newRecurrence *string
	err = h.pool.QueryRow(ctx,
		`SELECT id, title, due_date, recurrence_rule FROM tasks
		 WHERE user_id = $1 AND id != $2 AND is_completed = false AND is_deleted = false`,
		h.userID, taskID,
	).Scan(&newTaskID, &newTitle, &newDueDate, &newRecurrence)
	if err != nil {
		t.Fatalf("reading new recurring task: %v", err)
	}

	if newTitle != "Water plants" {
		t.Errorf("new task title = %q, want %q", newTitle, "Water plants")
	}

	expectedNextDue := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
	if newDueDate == nil || newDueDate.Format("2006-01-02") != expectedNextDue.Format("2006-01-02") {
		t.Errorf("new task due_date = %v, want %s", newDueDate, expectedNextDue.Format("2006-01-02"))
	}

	if newRecurrence == nil || *newRecurrence != string(domain.RecurrenceWeekly) {
		t.Errorf("new task recurrence_rule = %v, want %q", newRecurrence, domain.RecurrenceWeekly)
	}

	// Verify labels were copied
	var labelCount int
	err = h.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM task_labels WHERE task_id = $1`,
		newTaskID,
	).Scan(&labelCount)
	if err != nil {
		t.Fatalf("counting labels on new task: %v", err)
	}
	if labelCount != 1 {
		t.Errorf("new task has %d labels, want 1", labelCount)
	}

	var foundLabelID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT label_id FROM task_labels WHERE task_id = $1`,
		newTaskID,
	).Scan(&foundLabelID)
	if err != nil {
		t.Fatalf("reading label on new task: %v", err)
	}
	if foundLabelID != labelID {
		t.Errorf("label_id = %s, want %s", foundLabelID, labelID)
	}
}
