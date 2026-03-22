//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/auth"
	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/handler"
	"github.com/vasi1796/doit/internal/projection"
)

func TestReorderTaskSyncsPosition(t *testing.T) {
	// Full pipeline: create two tasks → sync a position-only update →
	// outbox → RabbitMQ → projection worker → verify position changed in read model.
	h := setupHarness(t)
	ctx := context.Background()

	// Create two tasks with known positions
	task1ID := uuid.New()
	task2ID := uuid.New()

	for _, tc := range []struct {
		id  uuid.UUID
		pos string
	}{
		{task1ID, "a"},
		{task2ID, "b"},
	} {
		if err := h.cmdHandler.CreateTask(ctx, domain.CreateTask{
			TaskID:   tc.id,
			UserID:   h.userID,
			Title:    "Task " + tc.pos,
			Priority: domain.PriorityNone,
			Position: tc.pos,
		}); err != nil {
			t.Fatalf("CreateTask %s: %v", tc.pos, err)
		}
	}

	// Flush and project creation events
	h.flushOutbox(t)
	h.drainProjections(t)
	h.drainRecurring(t)

	// Verify initial positions
	var pos1Before, pos2Before string
	if err := h.pool.QueryRow(ctx, `SELECT position FROM tasks WHERE id = $1 AND user_id = $2`, task1ID, h.userID).Scan(&pos1Before); err != nil {
		t.Fatalf("reading task1 position: %v", err)
	}
	if pos1Before != "a" {
		t.Fatalf("task1 position = %q, want %q", pos1Before, "a")
	}

	// Sync a position-only update for task1 (reorder it after task2)
	hub := handler.NewHub(h.logger)
	snapWriter := projection.NewSnapshotWriter(h.pool, h.logger)
	syncHandler := handler.NewSyncHandler(h.cmdHandler, h.store, h.clock, hub, snapWriter, h.pool, h.logger)

	reqBody := map[string]any{
		"operations": []map[string]any{
			{
				"type":         "UpdateTask",
				"aggregate_id": task1ID.String(),
				"data":         map[string]any{"position": "c"},
				"hlc_time":     time.Now().UnixMilli(),
				"hlc_counter":  0,
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUserID(req.Context(), h.userID))
	rr := httptest.NewRecorder()
	syncHandler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("sync returned %d: %s", rr.Code, rr.Body.String())
	}

	// Flush outbox and drain projections
	h.flushOutbox(t)
	h.drainProjections(t)

	// Verify task1 position changed in read model
	var pos1After string
	if err := h.pool.QueryRow(ctx, `SELECT position FROM tasks WHERE id = $1 AND user_id = $2`, task1ID, h.userID).Scan(&pos1After); err != nil {
		t.Fatalf("reading task1 position after reorder: %v", err)
	}
	if pos1After != "c" {
		t.Errorf("task1 position after reorder = %q, want %q", pos1After, "c")
	}

	// Verify task2 position unchanged
	if err := h.pool.QueryRow(ctx, `SELECT position FROM tasks WHERE id = $1 AND user_id = $2`, task2ID, h.userID).Scan(&pos2Before); err != nil {
		t.Fatalf("reading task2 position: %v", err)
	}
	if pos2Before != "b" {
		t.Errorf("task2 position = %q, want %q", pos2Before, "b")
	}
}
