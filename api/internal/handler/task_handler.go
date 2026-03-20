package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
)

// TaskCommander is the interface the task handler needs from the domain.
type TaskCommander interface {
	CreateTask(ctx context.Context, cmd domain.CreateTask) error
	CompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CompleteTask) error
	UncompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UncompleteTask) error
	DeleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.DeleteTask) error
	MoveTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.MoveTask) error
	UpdateTaskDescription(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskDescription) error
	AddLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.AddLabel) error
	RemoveLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.RemoveLabel) error
	CreateSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CreateSubtask) error
	CompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CompleteSubtask) error
}

type TaskHandler struct {
	cmds   TaskCommander
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewTaskHandler(cmds TaskCommander, pool *pgxpool.Pool, logger zerolog.Logger) *TaskHandler {
	return &TaskHandler{cmds: cmds, pool: pool, logger: logger}
}

// Request types

type createTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    int        `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	ListID      *uuid.UUID `json:"list_id,omitempty"`
	Position    string     `json:"position"`
}

type updateTaskRequest struct {
	Description *string    `json:"description,omitempty"`
	ListID      *uuid.UUID `json:"list_id,omitempty"`
	Position    *string    `json:"position,omitempty"`
}

type createSubtaskRequest struct {
	Title    string `json:"title"`
	Position string `json:"position"`
}

type addLabelRequest struct {
	LabelID uuid.UUID `json:"label_id"`
}

// Response types

type taskResponse struct {
	ID          uuid.UUID         `json:"id"`
	ListID      *uuid.UUID        `json:"list_id,omitempty"`
	Title       string            `json:"title"`
	Description *string           `json:"description,omitempty"`
	Priority    int               `json:"priority"`
	DueDate     *string           `json:"due_date,omitempty"`
	Position    string            `json:"position"`
	IsCompleted bool              `json:"is_completed"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	IsDeleted   bool              `json:"is_deleted"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Subtasks    []subtaskResponse `json:"subtasks,omitempty"`
	Labels      []labelResponse   `json:"labels,omitempty"`
}

type subtaskResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	IsCompleted bool      `json:"is_completed"`
	Position    string    `json:"position"`
}

// Create handles POST /api/v1/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req createTaskRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	taskID := uuid.New()
	cmd := domain.CreateTask{
		TaskID:      taskID,
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		ListID:      req.ListID,
		Position:    req.Position,
	}

	if mapDomainError(w, h.logger, h.cmds.CreateTask(r.Context(), cmd)) {
		return
	}

	writeJSON(w, h.logger, http.StatusCreated, map[string]string{"id": taskID.String()})
}

// Update handles PATCH /api/v1/tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	var req updateTaskRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	ctx := r.Context()

	if req.Description != nil {
		err := h.cmds.UpdateTaskDescription(ctx, taskID, userID, domain.UpdateTaskDescription{
			Description: *req.Description,
		})
		if mapDomainError(w, h.logger, err) {
			return
		}
	}

	if req.ListID != nil && req.Position != nil {
		err := h.cmds.MoveTask(ctx, taskID, userID, domain.MoveTask{
			ListID:   *req.ListID,
			Position: *req.Position,
		})
		if mapDomainError(w, h.logger, err) {
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete handles DELETE /api/v1/tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	err := h.cmds.DeleteTask(r.Context(), taskID, userID, domain.DeleteTask{
		DeletedAt: time.Now().UTC(),
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Complete handles POST /api/v1/tasks/{id}/complete
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	err := h.cmds.CompleteTask(r.Context(), taskID, userID, domain.CompleteTask{
		CompletedAt: time.Now().UTC(),
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Uncomplete handles POST /api/v1/tasks/{id}/uncomplete
func (h *TaskHandler) Uncomplete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	err := h.cmds.UncompleteTask(r.Context(), taskID, userID, domain.UncompleteTask{})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateSubtask handles POST /api/v1/tasks/{id}/subtasks
func (h *TaskHandler) CreateSubtask(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	var req createSubtaskRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	subtaskID := uuid.New()
	err := h.cmds.CreateSubtask(r.Context(), taskID, userID, domain.CreateSubtask{
		SubtaskID: subtaskID,
		Title:     req.Title,
		Position:  req.Position,
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	writeJSON(w, h.logger, http.StatusCreated, map[string]string{"id": subtaskID.String()})
}

// CompleteSubtask handles POST /api/v1/tasks/{id}/subtasks/{sid}/complete
func (h *TaskHandler) CompleteSubtask(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}
	subtaskID, ok := parseUUID(w, h.logger, r, "sid")
	if !ok {
		return
	}
	err := h.cmds.CompleteSubtask(r.Context(), taskID, userID, domain.CompleteSubtask{
		SubtaskID:   subtaskID,
		CompletedAt: time.Now().UTC(),
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddLabel handles POST /api/v1/tasks/{id}/labels
func (h *TaskHandler) AddLabel(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	var req addLabelRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	err := h.cmds.AddLabel(r.Context(), taskID, userID, domain.AddLabel{
		LabelID: req.LabelID,
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveLabel handles DELETE /api/v1/tasks/{id}/labels/{lid}
func (h *TaskHandler) RemoveLabel(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}
	labelID, ok := parseUUID(w, h.logger, r, "lid")
	if !ok {
		return
	}

	err := h.cmds.RemoveLabel(r.Context(), taskID, userID, domain.RemoveLabel{
		LabelID: labelID,
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List handles GET /api/v1/tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	query := `SELECT id, list_id, title, description, priority, due_date, position,
		is_completed, completed_at, is_deleted, created_at, updated_at
		FROM tasks WHERE user_id = $1 AND is_deleted = false`
	args := []any{userID}
	argIdx := 2

	if listID := r.URL.Query().Get("list_id"); listID != "" {
		parsed, err := uuid.Parse(listID)
		if err != nil {
			writeError(w, h.logger, http.StatusBadRequest, "invalid list_id")
			return
		}
		query += fmt.Sprintf(" AND list_id = $%d", argIdx)
		args = append(args, parsed)
		argIdx++
	}

	if completed := r.URL.Query().Get("is_completed"); completed == "true" {
		query += fmt.Sprintf(" AND is_completed = $%d", argIdx)
		args = append(args, true)
		argIdx++
	} else if completed == "false" {
		query += fmt.Sprintf(" AND is_completed = $%d", argIdx)
		args = append(args, false)
		argIdx++
	}

	// Tasks with no list (inbox)
	if r.URL.Query().Get("inbox") == "true" {
		query += " AND list_id IS NULL"
	}

	query += " ORDER BY position ASC"

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		h.logger.Error().Err(err).Msg("querying tasks")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	tasks := make([]taskResponse, 0)
	for rows.Next() {
		var t taskResponse
		var listID *uuid.UUID
		var description sql.NullString
		var dueDate sql.NullString
		var completedAt sql.NullTime

		if err := rows.Scan(
			&t.ID, &listID, &t.Title, &description, &t.Priority, &dueDate,
			&t.Position, &t.IsCompleted, &completedAt, &t.IsDeleted,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			h.logger.Error().Err(err).Msg("scanning task row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}

		t.ListID = listID
		if description.Valid {
			t.Description = &description.String
		}
		if dueDate.Valid {
			t.DueDate = &dueDate.String
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}

		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		h.logger.Error().Err(err).Msg("iterating task rows")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, h.logger, http.StatusOK, tasks)
}

// Get handles GET /api/v1/tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}
	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	ctx := r.Context()

	// Load task
	var t taskResponse
	var listID *uuid.UUID
	var description sql.NullString
	var dueDate sql.NullString
	var completedAt sql.NullTime

	err := h.pool.QueryRow(ctx,
		`SELECT id, list_id, title, description, priority, due_date, position,
			is_completed, completed_at, is_deleted, created_at, updated_at
		 FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, userID,
	).Scan(
		&t.ID, &listID, &t.Title, &description, &t.Priority, &dueDate,
		&t.Position, &t.IsCompleted, &completedAt, &t.IsDeleted,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, h.logger, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error().Err(err).Msg("querying task")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	t.ListID = listID
	if description.Valid {
		t.Description = &description.String
	}
	if dueDate.Valid {
		t.DueDate = &dueDate.String
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}

	// Load subtasks
	subtaskRows, err := h.pool.Query(ctx,
		`SELECT id, title, is_completed, position FROM subtasks WHERE task_id = $1 ORDER BY position ASC`,
		taskID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("querying subtasks")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer subtaskRows.Close()

	t.Subtasks = make([]subtaskResponse, 0)
	for subtaskRows.Next() {
		var s subtaskResponse
		if err := subtaskRows.Scan(&s.ID, &s.Title, &s.IsCompleted, &s.Position); err != nil {
			h.logger.Error().Err(err).Msg("scanning subtask row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}
		t.Subtasks = append(t.Subtasks, s)
	}
	if err := subtaskRows.Err(); err != nil {
		h.logger.Error().Err(err).Msg("iterating subtask rows")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	// Load labels
	labelRows, err := h.pool.Query(ctx,
		`SELECT l.id, l.name, l.colour FROM labels l
		 JOIN task_labels tl ON tl.label_id = l.id
		 WHERE tl.task_id = $1`,
		taskID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("querying task labels")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer labelRows.Close()

	t.Labels = make([]labelResponse, 0)
	for labelRows.Next() {
		var l labelResponse
		if err := labelRows.Scan(&l.ID, &l.Name, &l.Colour); err != nil {
			h.logger.Error().Err(err).Msg("scanning label row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}
		t.Labels = append(t.Labels, l)
	}
	if err := labelRows.Err(); err != nil {
		h.logger.Error().Err(err).Msg("iterating label rows")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, h.logger, http.StatusOK, t)
}
