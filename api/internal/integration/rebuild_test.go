//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
)

func TestRebuildProjections(t *testing.T) {
	// Full flow: create tasks/labels via CommandHandler → flush + project →
	// verify read models → truncate read models → rebuild from event log →
	// verify read models match original state.
	h := setupHarness(t)
	ctx := context.Background()

	// 1. Create a label
	labelID := uuid.New()
	if err := h.cmdHandler.CreateLabel(ctx, domain.CreateLabel{
		LabelID: labelID,
		UserID:  h.userID,
		Name:    "Urgent",
		Colour:  "#ff3b30",
	}); err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}

	// 2. Create two tasks, one with a label
	task1ID := uuid.New()
	dueDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if err := h.cmdHandler.CreateTask(ctx, domain.CreateTask{
		TaskID:   task1ID,
		UserID:   h.userID,
		Title:    "Task one",
		Priority: domain.PriorityHigh,
		DueDate:  &dueDate,
		Position: "a",
	}); err != nil {
		t.Fatalf("CreateTask 1: %v", err)
	}

	if err := h.cmdHandler.AddLabel(ctx, task1ID, h.userID, domain.AddLabel{LabelID: labelID}); err != nil {
		t.Fatalf("AddLabel: %v", err)
	}

	task2ID := uuid.New()
	if err := h.cmdHandler.CreateTask(ctx, domain.CreateTask{
		TaskID:   task2ID,
		UserID:   h.userID,
		Title:    "Task two",
		Priority: domain.PriorityLow,
		Position: "b",
	}); err != nil {
		t.Fatalf("CreateTask 2: %v", err)
	}

	// Complete task 2
	if err := h.cmdHandler.CompleteTask(ctx, task2ID, h.userID, domain.CompleteTask{
		CompletedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// 3. Flush outbox + project so read models are populated
	h.flushOutbox(t)
	h.drainProjections(t)
	h.drainRecurring(t)

	// 4. Snapshot the read model state before rebuild
	type taskRow struct {
		title       string
		priority    int
		isCompleted bool
	}

	readTask := func(id uuid.UUID) taskRow {
		var r taskRow
		err := h.pool.QueryRow(ctx,
			`SELECT title, priority, is_completed FROM tasks WHERE id = $1`, id,
		).Scan(&r.title, &r.priority, &r.isCompleted)
		if err != nil {
			t.Fatalf("reading task %s: %v", id, err)
		}
		return r
	}

	before1 := readTask(task1ID)
	before2 := readTask(task2ID)

	var labelCountBefore int
	if err := h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM task_labels WHERE task_id = $1`, task1ID).Scan(&labelCountBefore); err != nil {
		t.Fatalf("counting labels: %v", err)
	}

	// 5. Truncate read model tables (simulate disaster / migration)
	if _, err := h.pool.Exec(ctx, "TRUNCATE subtasks, task_labels, tasks, labels, lists, aggregate_snapshots CASCADE"); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}

	// Verify tasks are gone
	var count int
	if err := h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&count); err != nil {
		t.Fatalf("counting tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 tasks after truncate, got %d", count)
	}

	// 6. Rebuild: replay all events through the projector
	store := h.store
	total, err := store.LoadAllBatched(ctx, 100, func(batch []eventstore.Event) error {
		return h.projector.Project(ctx, batch)
	})
	if err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}
	if total == 0 {
		t.Fatal("expected events to replay, got 0")
	}

	// 7. Verify read models match pre-truncation state
	after1 := readTask(task1ID)
	after2 := readTask(task2ID)

	if after1 != before1 {
		t.Errorf("task1 mismatch: before=%+v, after=%+v", before1, after1)
	}
	if after2 != before2 {
		t.Errorf("task2 mismatch: before=%+v, after=%+v", before2, after2)
	}

	var labelCountAfter int
	if err := h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM task_labels WHERE task_id = $1`, task1ID).Scan(&labelCountAfter); err != nil {
		t.Fatalf("counting labels after rebuild: %v", err)
	}
	if labelCountAfter != labelCountBefore {
		t.Errorf("label count mismatch: before=%d, after=%d", labelCountBefore, labelCountAfter)
	}

	// Verify label read model was also rebuilt
	var labelName string
	if err := h.pool.QueryRow(ctx, `SELECT name FROM labels WHERE id = $1`, labelID).Scan(&labelName); err != nil {
		t.Fatalf("reading label after rebuild: %v", err)
	}
	if labelName != "Urgent" {
		t.Errorf("label name = %q, want %q", labelName, "Urgent")
	}
}
