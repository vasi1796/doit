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
	"github.com/vasi1796/doit/internal/handler"
	"github.com/vasi1796/doit/internal/projection"
)

func TestSyncPushFlow(t *testing.T) {
	// Full pipeline: POST /api/v1/sync with batched CreateTask →
	// CommandHandler writes events + outbox → outbox poller → RabbitMQ →
	// projection worker updates read model → task appears.
	h := setupHarness(t)
	ctx := context.Background()

	hub := handler.NewHub(h.logger)
	snapWriter := projection.NewSnapshotWriter(h.pool, h.logger)
	syncHandler := handler.NewSyncHandler(h.cmdHandler, h.store, h.clock, hub, snapWriter, h.pool, h.logger)

	taskID := uuid.New()
	dueDate := "2026-05-15"

	// Build sync request with a CreateTask operation
	reqBody := map[string]any{
		"operations": []map[string]any{
			{
				"type":         "CreateTask",
				"aggregate_id": taskID.String(),
				"data": map[string]any{
					"title":    "Sync test task",
					"priority": 1,
					"position": "a",
					"due_date": dueDate,
				},
				"hlc_time":    time.Now().UnixMilli(),
				"hlc_counter": 0,
			},
		},
		"cursor": nil,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	// Create HTTP request with user ID in context (simulates JWT middleware)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUserID(req.Context(), h.userID))

	rr := httptest.NewRecorder()
	syncHandler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("sync returned %d: %s", rr.Code, rr.Body.String())
	}

	// Parse response — check no failed ops
	var resp struct {
		Cursor    map[string]any `json:"cursor"`
		Events    []any          `json:"events"`
		FailedOps []int          `json:"failed_ops"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.FailedOps) > 0 {
		t.Fatalf("sync had failed ops: %v", resp.FailedOps)
	}

	// Verify events were stored in event store
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

	// Flush outbox → RabbitMQ
	h.flushOutbox(t)

	// Drain projections → read model
	projected := h.drainProjections(t)
	if projected != 1 {
		t.Fatalf("expected 1 projected event, got %d", projected)
	}

	// Verify task in read model
	var title string
	var readDueDate *time.Time
	err = h.pool.QueryRow(ctx,
		`SELECT title, due_date FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, h.userID,
	).Scan(&title, &readDueDate)
	if err != nil {
		t.Fatalf("reading task from read model: %v", err)
	}

	if title != "Sync test task" {
		t.Errorf("title = %q, want %q", title, "Sync test task")
	}
	if readDueDate == nil || readDueDate.Format("2006-01-02") != dueDate {
		t.Errorf("due_date = %v, want %s", readDueDate, dueDate)
	}
}
