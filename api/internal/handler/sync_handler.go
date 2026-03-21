package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
)

// SyncCommander is the interface the sync handler needs from the domain command layer.
type SyncCommander interface {
	CreateTask(ctx context.Context, cmd domain.CreateTask) error
	CompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CompleteTask) error
	UncompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UncompleteTask) error
	DeleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.DeleteTask) error
	RestoreTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.RestoreTask) error
	MoveTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.MoveTask) error
	UpdateTaskTitle(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskTitle) error
	UpdateTaskDescription(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskDescription) error
	UpdateTaskPriority(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskPriority) error
	UpdateTaskDueDate(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskDueDate) error
	UpdateTaskDueTime(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskDueTime) error
	UpdateTaskRecurrence(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskRecurrence) error
	AddLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.AddLabel) error
	RemoveLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.RemoveLabel) error
	CreateSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CreateSubtask) error
	CompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CompleteSubtask) error
	UncompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UncompleteSubtask) error
	UpdateSubtaskTitle(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateSubtaskTitle) error
	CreateList(ctx context.Context, cmd domain.CreateList) error
	DeleteList(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.DeleteList) error
	CreateLabel(ctx context.Context, cmd domain.CreateLabel) error
	DeleteLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.DeleteLabel) error
}

// SyncEventLoader is the interface the sync handler needs from the event store.
type SyncEventLoader interface {
	LoadByUserSince(ctx context.Context, userID uuid.UUID, since time.Time, sinceCounter int) ([]eventstore.Event, error)
}

// SyncClock is the interface the sync handler needs from the HLC clock.
type SyncClock interface {
	Now() hlc.Timestamp
	Update(remote hlc.Timestamp) hlc.Timestamp
}

// SyncSnapshotWriter is the interface the sync handler needs from the projection layer.
type SyncSnapshotWriter interface {
	SaveTaskSnapshot(ctx context.Context, taskID, userID uuid.UUID) error
	SaveListSnapshot(ctx context.Context, listID, userID uuid.UUID) error
	SaveLabelSnapshot(ctx context.Context, labelID, userID uuid.UUID) error
}

// SyncHandler processes batched sync operations from clients.
type SyncHandler struct {
	cmds      SyncCommander
	store     SyncEventLoader
	clock     SyncClock
	hub       *Hub
	snapshots SyncSnapshotWriter
	pool      *pgxpool.Pool
	logger    zerolog.Logger
}

func NewSyncHandler(cmds SyncCommander, store SyncEventLoader, clock SyncClock, hub *Hub, snapshots SyncSnapshotWriter, pool *pgxpool.Pool, logger zerolog.Logger) *SyncHandler {
	return &SyncHandler{cmds: cmds, store: store, clock: clock, hub: hub, snapshots: snapshots, pool: pool, logger: logger}
}

type syncRequest struct {
	Operations []syncOperation `json:"operations"`
	Cursor     *syncCursor     `json:"cursor"`
}

type syncOperation struct {
	Type        string         `json:"type"`
	AggregateID string         `json:"aggregate_id"`
	Data        map[string]any `json:"data"`
	HLCTime     int64          `json:"hlc_time"`
	HLCCounter  int            `json:"hlc_counter"`
}

type syncCursor struct {
	HLCTime    int64 `json:"hlc_time"`
	HLCCounter int   `json:"hlc_counter"`
}

type syncResponse struct {
	Cursor    syncCursor         `json:"cursor"`
	Events    []eventstore.Event `json:"events"`
	FailedOps []int              `json:"failed_ops,omitempty"`
}

// Sync handles POST /api/v1/sync
func (h *SyncHandler) Sync(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req syncRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	// Process each operation through the CommandHandler.
	// Operations are processed sequentially to maintain ordering.
	var failedOps []int
	for i, op := range req.Operations {
		// Update server HLC with client timestamp for causal ordering
		clientHLC := hlc.Timestamp{
			Time:    time.UnixMilli(op.HLCTime),
			Counter: op.HLCCounter,
		}
		h.clock.Update(clientHLC)

		aggID, err := uuid.Parse(op.AggregateID)
		if err != nil {
			h.logger.Warn().Str("aggregate_id", op.AggregateID).Msg("sync: invalid aggregate ID, skipping")
			failedOps = append(failedOps, i)
			continue
		}

		if err := h.dispatchOp(r, userID, aggID, op); err != nil {
			h.logger.Warn().Err(err).Str("type", op.Type).Str("aggregate_id", op.AggregateID).Msg("sync: operation failed, skipping")
			failedOps = append(failedOps, i)
			continue
		}

		// Save snapshot for the affected aggregate
		h.saveSnapshot(r.Context(), op.Type, aggID, userID)
	}

	// Build response with current server HLC as new cursor
	serverNow := h.clock.Now()
	resp := syncResponse{
		Cursor: syncCursor{
			HLCTime:    serverNow.Time.UnixMilli(),
			HLCCounter: serverNow.Counter,
		},
		Events:    []eventstore.Event{},
		FailedOps: failedOps,
	}

	// If client sent a cursor, include events since that point (for pull)
	if req.Cursor != nil {
		since := time.UnixMilli(req.Cursor.HLCTime)
		events, err := h.store.LoadByUserSince(r.Context(), userID, since, req.Cursor.HLCCounter)
		if err != nil {
			h.logger.Error().Err(err).Msg("sync: failed to load events for pull")
		} else {
			resp.Events = events
			// Push events to OTHER connected WS clients (not the one that synced).
			// The sync response already includes these events for the pushing client.
			// Passing nil as sender broadcasts to all WS clients — the pushing client
			// is an HTTP client, not a WS client, so nil correctly excludes no one.
			// This is acceptable: the client-side merge is idempotent (LWW comparison).
			h.hub.Broadcast(userID, events, nil)
		}
	}

	writeJSON(w, h.logger, http.StatusOK, resp)
}

func (h *SyncHandler) dispatchOp(r *http.Request, userID, aggID uuid.UUID, op syncOperation) error {
	ctx := r.Context()
	data := op.Data

	switch op.Type {
	case "CreateTask":
		cmd := domain.CreateTask{
			TaskID:   aggID,
			UserID:   userID,
			Title:    strVal(data, "title"),
			Priority: domain.Priority(intVal(data, "priority")),
			Position: strVal(data, "position"),
		}
		if v, ok := data["description"]; ok && v != nil {
			cmd.Description = strVal(data, "description")
		}
		if v, ok := data["due_date"]; ok && v != nil {
			s := strVal(data, "due_date")
			if t, err := time.Parse("2006-01-02", s); err == nil {
				cmd.DueDate = &t
			}
		}
		if v, ok := data["due_time"]; ok && v != nil {
			s := strVal(data, "due_time")
			cmd.DueTime = &s
		}
		if v, ok := data["list_id"]; ok && v != nil {
			if id, err := uuid.Parse(strVal(data, "list_id")); err == nil {
				cmd.ListID = &id
			}
		}
		return h.cmds.CreateTask(ctx, cmd)

	case "UpdateTask":
		return h.dispatchUpdateTask(ctx, aggID, userID, data)

	case "CompleteTask":
		return h.cmds.CompleteTask(ctx, aggID, userID, domain.CompleteTask{CompletedAt: time.Now().UTC()})

	case "UncompleteTask":
		return h.cmds.UncompleteTask(ctx, aggID, userID, domain.UncompleteTask{})

	case "DeleteTask":
		return h.cmds.DeleteTask(ctx, aggID, userID, domain.DeleteTask{DeletedAt: time.Now().UTC()})

	case "RestoreTask":
		return h.cmds.RestoreTask(ctx, aggID, userID, domain.RestoreTask{})

	case "AddLabel":
		labelID, err := uuid.Parse(strVal(data, "label_id"))
		if err != nil {
			return err
		}
		return h.cmds.AddLabel(ctx, aggID, userID, domain.AddLabel{LabelID: labelID})

	case "RemoveLabel":
		labelID, err := uuid.Parse(strVal(data, "label_id"))
		if err != nil {
			return err
		}
		return h.cmds.RemoveLabel(ctx, aggID, userID, domain.RemoveLabel{LabelID: labelID})

	case "CreateSubtask":
		subtaskID, err := uuid.Parse(strVal(data, "subtask_id"))
		if err != nil {
			return err
		}
		return h.cmds.CreateSubtask(ctx, aggID, userID, domain.CreateSubtask{
			SubtaskID: subtaskID,
			Title:     strVal(data, "title"),
			Position:  strVal(data, "position"),
		})

	case "CompleteSubtask":
		subtaskID, err := uuid.Parse(strVal(data, "subtask_id"))
		if err != nil {
			return err
		}
		return h.cmds.CompleteSubtask(ctx, aggID, userID, domain.CompleteSubtask{
			SubtaskID:   subtaskID,
			CompletedAt: time.Now().UTC(),
		})

	case "UncompleteSubtask":
		subtaskID, err := uuid.Parse(strVal(data, "subtask_id"))
		if err != nil {
			return err
		}
		return h.cmds.UncompleteSubtask(ctx, aggID, userID, domain.UncompleteSubtask{SubtaskID: subtaskID})

	case "UpdateSubtaskTitle":
		subtaskID, err := uuid.Parse(strVal(data, "subtask_id"))
		if err != nil {
			return err
		}
		return h.cmds.UpdateSubtaskTitle(ctx, aggID, userID, domain.UpdateSubtaskTitle{
			SubtaskID: subtaskID,
			Title:     strVal(data, "title"),
		})

	case "CreateList":
		return h.cmds.CreateList(ctx, domain.CreateList{
			ListID:   aggID,
			UserID:   userID,
			Name:     strVal(data, "name"),
			Colour:   strVal(data, "colour"),
			Icon:     strVal(data, "icon"),
			Position: strVal(data, "position"),
		})

	case "DeleteList":
		return h.cmds.DeleteList(ctx, aggID, userID, domain.DeleteList{DeletedAt: time.Now().UTC()})

	case "CreateLabel":
		return h.cmds.CreateLabel(ctx, domain.CreateLabel{
			LabelID: aggID,
			UserID:  userID,
			Name:    strVal(data, "name"),
			Colour:  strVal(data, "colour"),
		})

	case "DeleteLabel":
		return h.cmds.DeleteLabel(ctx, aggID, userID, domain.DeleteLabel{DeletedAt: time.Now().UTC()})

	default:
		h.logger.Warn().Str("type", op.Type).Msg("sync: unknown operation type")
		return nil
	}
}

func (h *SyncHandler) dispatchUpdateTask(ctx context.Context, aggID, userID uuid.UUID, data map[string]any) error {
	if _, ok := data["title"]; ok {
		title := strVal(data, "title")
		if title != "" {
			if err := h.cmds.UpdateTaskTitle(ctx, aggID, userID, domain.UpdateTaskTitle{Title: title}); err != nil {
				return err
			}
		}
	}
	if _, ok := data["description"]; ok {
		desc := strVal(data, "description")
		if err := h.cmds.UpdateTaskDescription(ctx, aggID, userID, domain.UpdateTaskDescription{Description: desc}); err != nil {
			return err
		}
	}
	if _, ok := data["priority"]; ok && data["priority"] != nil {
		if err := h.cmds.UpdateTaskPriority(ctx, aggID, userID, domain.UpdateTaskPriority{Priority: domain.Priority(intVal(data, "priority"))}); err != nil {
			return err
		}
	}
	if _, ok := data["due_date"]; ok {
		var dueDate *time.Time
		if data["due_date"] != nil {
			s := strVal(data, "due_date")
			if t, err := time.Parse("2006-01-02", s); err == nil {
				dueDate = &t
			}
		}
		if err := h.cmds.UpdateTaskDueDate(ctx, aggID, userID, domain.UpdateTaskDueDate{DueDate: dueDate}); err != nil {
			return err
		}
	}
	if _, ok := data["due_time"]; ok {
		var dueTime *string
		if data["due_time"] != nil {
			s := strVal(data, "due_time")
			dueTime = &s
		}
		if err := h.cmds.UpdateTaskDueTime(ctx, aggID, userID, domain.UpdateTaskDueTime{DueTime: dueTime}); err != nil {
			return err
		}
	}
	if _, ok := data["recurrence_rule"]; ok {
		rule := strVal(data, "recurrence_rule")
		if err := h.cmds.UpdateTaskRecurrence(ctx, aggID, userID, domain.UpdateTaskRecurrence{RecurrenceRule: domain.RecurrenceRule(rule)}); err != nil {
			return err
		}
	}
	if v, ok := data["list_id"]; ok && v != nil {
		lid, err := uuid.Parse(strVal(data, "list_id"))
		if err != nil {
			return err
		}
		pos := strVal(data, "position")
		if pos == "" {
			pos = time.Now().String() // fallback position
		}
		if err := h.cmds.MoveTask(ctx, aggID, userID, domain.MoveTask{ListID: lid, Position: pos}); err != nil {
			return err
		}
	}
	return nil
}

func strVal(data map[string]any, key string) string {
	if v, ok := data[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intVal(data map[string]any, key string) int {
	if v, ok := data[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n) // JSON numbers are float64
		case int:
			return n
		}
	}
	return 0
}

func (h *SyncHandler) saveSnapshot(ctx context.Context, opType string, aggID, userID uuid.UUID) {
	var err error
	switch opType {
	case "CreateList", "DeleteList":
		err = h.snapshots.SaveListSnapshot(ctx, aggID, userID)
	case "CreateLabel", "DeleteLabel":
		err = h.snapshots.SaveLabelSnapshot(ctx, aggID, userID)
	case "CreateTask", "UpdateTask", "CompleteTask", "UncompleteTask",
		"DeleteTask", "RestoreTask", "AddLabel", "RemoveLabel",
		"CreateSubtask", "CompleteSubtask", "UncompleteSubtask", "UpdateSubtaskTitle":
		err = h.snapshots.SaveTaskSnapshot(ctx, aggID, userID)
	default:
		h.logger.Warn().Str("type", opType).Msg("sync: unknown operation type for snapshot")
		return
	}
	if err != nil {
		h.logger.Warn().Err(err).Str("type", opType).Msg("sync: snapshot save failed")
	}
}
