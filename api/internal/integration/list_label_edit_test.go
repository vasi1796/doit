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

func TestListLabelEditFlow(t *testing.T) {
	// Full pipeline: sync push with UpdateList/UpdateLabel ops →
	// granular rename/recolour events → outbox → projections →
	// lists/labels read models reflect the edits.
	h := setupHarness(t)
	ctx := context.Background()

	snapWriter := projection.NewSnapshotWriter(h.pool, h.logger)
	syncHandler := handler.NewSyncHandler(h.cmdHandler, h.store, h.clock, snapWriter, h.pool, h.logger)

	listID := uuid.New()
	labelID := uuid.New()
	now := time.Now().UnixMilli()

	reqBody := map[string]any{
		"operations": []map[string]any{
			{
				"type":         "CreateList",
				"aggregate_id": listID.String(),
				"data":         map[string]any{"name": "Work", "colour": "#ff0000", "position": "a"},
				"hlc_time":     now,
				"hlc_counter":  0,
			},
			{
				"type":         "UpdateList",
				"aggregate_id": listID.String(),
				"data":         map[string]any{"name": "Office", "colour": "#00ff00"},
				"hlc_time":     now,
				"hlc_counter":  1,
			},
			{
				"type":         "CreateLabel",
				"aggregate_id": labelID.String(),
				"data":         map[string]any{"name": "Urgent", "colour": "#ff0000"},
				"hlc_time":     now,
				"hlc_counter":  2,
			},
			{
				"type":         "UpdateLabel",
				"aggregate_id": labelID.String(),
				"data":         map[string]any{"name": "Important", "colour": "#0000ff"},
				"hlc_time":     now,
				"hlc_counter":  3,
			},
		},
		"cursor": nil,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithUserID(req.Context(), h.userID))

	rr := httptest.NewRecorder()
	syncHandler.Sync(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("sync returned %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		FailedOps []int `json:"failed_ops"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.FailedOps) > 0 {
		t.Fatalf("sync had failed ops: %v", resp.FailedOps)
	}

	// Verify granular events were appended
	listEvents, err := h.store.LoadByAggregate(ctx, listID)
	if err != nil {
		t.Fatalf("LoadByAggregate(list): %v", err)
	}
	wantListEvents := []string{"ListCreated", "ListNameUpdated", "ListColourUpdated"}
	if len(listEvents) != len(wantListEvents) {
		t.Fatalf("expected %d list events, got %d", len(wantListEvents), len(listEvents))
	}
	for i, want := range wantListEvents {
		if string(listEvents[i].EventType) != want {
			t.Errorf("list event %d = %s, want %s", i, listEvents[i].EventType, want)
		}
	}

	// Flush outbox → RabbitMQ, drain projections → read models
	h.flushOutbox(t)
	projected := h.drainProjections(t)
	if projected != 6 {
		t.Fatalf("expected 6 projected events, got %d", projected)
	}

	var listName, listColour string
	err = h.pool.QueryRow(ctx,
		`SELECT name, colour FROM lists WHERE id = $1 AND user_id = $2`,
		listID, h.userID,
	).Scan(&listName, &listColour)
	if err != nil {
		t.Fatalf("reading list from read model: %v", err)
	}
	if listName != "Office" {
		t.Errorf("list name = %q, want %q", listName, "Office")
	}
	if listColour != "#00ff00" {
		t.Errorf("list colour = %q, want %q", listColour, "#00ff00")
	}

	var labelName, labelColour string
	err = h.pool.QueryRow(ctx,
		`SELECT name, colour FROM labels WHERE id = $1 AND user_id = $2`,
		labelID, h.userID,
	).Scan(&labelName, &labelColour)
	if err != nil {
		t.Fatalf("reading label from read model: %v", err)
	}
	if labelName != "Important" {
		t.Errorf("label name = %q, want %q", labelName, "Important")
	}
	if labelColour != "#0000ff" {
		t.Errorf("label colour = %q, want %q", labelColour, "#0000ff")
	}
}
