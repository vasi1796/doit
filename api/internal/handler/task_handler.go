package handler

import (
	"context"
	"database/sql"
	"errors"
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
	RestoreTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.RestoreTask) error
	MoveTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.MoveTask) error
	UpdateTaskDescription(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskDescription) error
	UpdateTaskTitle(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskTitle) error
	UpdateTaskPriority(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskPriority) error
	UpdateTaskDueDate(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskDueDate) error
	UpdateTaskDueTime(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskDueTime) error
	AddLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.AddLabel) error
	RemoveLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.RemoveLabel) error
	CreateSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CreateSubtask) error
	CompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.CompleteSubtask) error
	UncompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UncompleteSubtask) error
	UpdateTaskRecurrence(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateTaskRecurrence) error
	UpdateSubtaskTitle(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.UpdateSubtaskTitle) error
}

type TaskHandler struct {
	cmds   TaskCommander
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewTaskHandler(cmds TaskCommander, pool *pgxpool.Pool, logger zerolog.Logger) *TaskHandler {
	return &TaskHandler{cmds: cmds, pool: pool, logger: logger}
}

// Request and response types are generated from api/openapi.yaml
// in openapi_types.gen.go. Do not define them here.

// Create handles POST /api/v1/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req CreateTaskRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			writeError(w, h.logger, http.StatusBadRequest, "invalid due_date format, expected YYYY-MM-DD")
			return
		}
		dueDate = &parsed
	}

	var description string
	if req.Description != nil {
		description = *req.Description
	}

	taskID := uuid.New()
	cmd := domain.CreateTask{
		TaskID:      taskID,
		UserID:      userID,
		Title:       req.Title,
		Description: description,
		Priority:    domain.Priority(req.Priority),
		DueDate:     dueDate,
		DueTime:     req.DueTime,
		ListID:      req.ListId,
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
	var req UpdateTaskRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	ctx := r.Context()

	// Collect commands to dispatch from non-nil fields
	type cmdFn func() error
	var updates []cmdFn

	if req.Title != nil {
		v := *req.Title
		updates = append(updates, func() error { return h.cmds.UpdateTaskTitle(ctx, taskID, userID, domain.UpdateTaskTitle{Title: v}) })
	}
	if req.Description != nil {
		v := *req.Description
		updates = append(updates, func() error { return h.cmds.UpdateTaskDescription(ctx, taskID, userID, domain.UpdateTaskDescription{Description: v}) })
	}
	if req.Priority != nil {
		v := *req.Priority
		updates = append(updates, func() error { return h.cmds.UpdateTaskPriority(ctx, taskID, userID, domain.UpdateTaskPriority{Priority: domain.Priority(v)}) })
	}
	if req.DueDate != nil {
		var dueDate *time.Time
		if *req.DueDate != "" {
			parsed, err := time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				writeError(w, h.logger, http.StatusBadRequest, "invalid due_date format, expected YYYY-MM-DD")
				return
			}
			dueDate = &parsed
		}
		updates = append(updates, func() error { return h.cmds.UpdateTaskDueDate(ctx, taskID, userID, domain.UpdateTaskDueDate{DueDate: dueDate}) })
	}
	if req.DueTime != nil {
		v := req.DueTime
		updates = append(updates, func() error { return h.cmds.UpdateTaskDueTime(ctx, taskID, userID, domain.UpdateTaskDueTime{DueTime: v}) })
	}
	if req.RecurrenceRule != nil {
		v := *req.RecurrenceRule
		updates = append(updates, func() error { return h.cmds.UpdateTaskRecurrence(ctx, taskID, userID, domain.UpdateTaskRecurrence{RecurrenceRule: domain.RecurrenceRule(v)}) })
	}
	if req.ListId != nil && req.Position != nil {
		lid, pos := *req.ListId, *req.Position
		updates = append(updates, func() error { return h.cmds.MoveTask(ctx, taskID, userID, domain.MoveTask{ListID: lid, Position: pos}) })
	}

	for _, fn := range updates {
		if mapDomainError(w, h.logger, fn()) {
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

// Restore handles POST /api/v1/tasks/{id}/restore
func (h *TaskHandler) Restore(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	taskID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	err := h.cmds.RestoreTask(r.Context(), taskID, userID, domain.RestoreTask{})
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

	var req CreateSubtaskRequest
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

// UncompleteSubtask handles POST /api/v1/tasks/{id}/subtasks/{sid}/uncomplete
func (h *TaskHandler) UncompleteSubtask(w http.ResponseWriter, r *http.Request) {
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
	err := h.cmds.UncompleteSubtask(r.Context(), taskID, userID, domain.UncompleteSubtask{
		SubtaskID: subtaskID,
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateSubtaskTitle handles PATCH /api/v1/tasks/{id}/subtasks/{sid}
func (h *TaskHandler) UpdateSubtaskTitle(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateSubtaskRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	err := h.cmds.UpdateSubtaskTitle(r.Context(), taskID, userID, domain.UpdateSubtaskTitle{
		SubtaskID: subtaskID,
		Title:     req.Title,
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

	var req AddLabelRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	err := h.cmds.AddLabel(r.Context(), taskID, userID, domain.AddLabel{
		LabelID: req.LabelId,
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

	showDeleted := r.URL.Query().Get("is_deleted") == "true"
	query := `SELECT id, list_id, title, description, priority, due_date, due_time, position,
		is_completed, completed_at, is_deleted, created_at, updated_at, recurrence_rule
		FROM tasks WHERE user_id = $1`
	if showDeleted {
		query += " AND is_deleted = true"
	} else {
		query += " AND is_deleted = false"
	}
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

	if labelID := r.URL.Query().Get("label_id"); labelID != "" {
		parsed, err := uuid.Parse(labelID)
		if err != nil {
			writeError(w, h.logger, http.StatusBadRequest, "invalid label_id")
			return
		}
		query += fmt.Sprintf(" AND id IN (SELECT task_id FROM task_labels WHERE label_id = $%d)", argIdx)
		args = append(args, parsed)
		argIdx++
	}

	query += " ORDER BY position ASC"

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		h.logger.Error().Err(err).Msg("querying tasks")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		t, err := scanTaskRow(rows)
		if err != nil {
			h.logger.Error().Err(err).Msg("scanning task row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		h.logger.Error().Err(err).Msg("iterating task rows")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.loadLabelsForTasks(r.Context(), tasks); err != nil {
		h.logger.Error().Err(err).Msg("loading labels for tasks")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.loadSubtasksForTasks(r.Context(), tasks); err != nil {
		h.logger.Error().Err(err).Msg("loading subtasks for tasks")
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

	row := h.pool.QueryRow(r.Context(),
		`SELECT id, list_id, title, description, priority, due_date, due_time, position,
			is_completed, completed_at, is_deleted, created_at, updated_at, recurrence_rule
		 FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, userID,
	)

	t, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, h.logger, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error().Err(err).Msg("querying task")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	// Reuse batch loaders with a single-element slice
	tasks := []Task{t}
	if err := h.loadLabelsForTasks(r.Context(), tasks); err != nil {
		h.logger.Error().Err(err).Msg("loading labels for task")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	if err := h.loadSubtasksForTasks(r.Context(), tasks); err != nil {
		h.logger.Error().Err(err).Msg("loading subtasks for task")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, h.logger, http.StatusOK, tasks[0])
}

// scanTaskRow scans a task row from the standard SELECT columns into a Task.
func scanTaskRow(rows interface{ Scan(dest ...any) error }) (Task, error) {
	var t Task
	var listID *uuid.UUID
	var description, dueDate, dueTime, recurrenceRule sql.NullString
	var completedAt sql.NullTime

	err := rows.Scan(
		&t.Id, &listID, &t.Title, &description, &t.Priority, &dueDate, &dueTime,
		&t.Position, &t.IsCompleted, &completedAt, &t.IsDeleted,
		&t.CreatedAt, &t.UpdatedAt, &recurrenceRule,
	)
	if err != nil {
		return t, err
	}

	t.ListId = listID
	if description.Valid {
		t.Description = &description.String
	}
	if dueDate.Valid {
		t.DueDate = &dueDate.String
	}
	if dueTime.Valid {
		t.DueTime = &dueTime.String
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	if recurrenceRule.Valid && recurrenceRule.String != "" {
		t.RecurrenceRule = &recurrenceRule.String
	}
	return t, nil
}

// loadLabelsForTasks batch-loads labels for a slice of tasks.
func (h *TaskHandler) loadLabelsForTasks(ctx context.Context, tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}
	taskIDs := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.Id
	}

	rows, err := h.pool.Query(ctx,
		`SELECT tl.task_id, l.id, l.name, l.colour FROM labels l
		 JOIN task_labels tl ON tl.label_id = l.id
		 WHERE tl.task_id = ANY($1)`, taskIDs)
	if err != nil {
		return fmt.Errorf("batch loading labels: %w", err)
	}
	defer rows.Close()

	labelMap := make(map[uuid.UUID][]Label)
	for rows.Next() {
		var taskID uuid.UUID
		var l Label
		var colour sql.NullString
		if err := rows.Scan(&taskID, &l.Id, &l.Name, &colour); err != nil {
			return fmt.Errorf("scanning batch label: %w", err)
		}
		if colour.Valid {
			l.Colour = &colour.String
		}
		labelMap[taskID] = append(labelMap[taskID], l)
	}
	for i := range tasks {
		if labels, ok := labelMap[tasks[i].Id]; ok {
			tasks[i].Labels = &labels
		}
	}
	return nil
}

// loadSubtasksForTasks batch-loads subtasks for a slice of tasks.
func (h *TaskHandler) loadSubtasksForTasks(ctx context.Context, tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}
	taskIDs := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.Id
	}

	rows, err := h.pool.Query(ctx,
		`SELECT task_id, id, title, is_completed, position FROM subtasks
		 WHERE task_id = ANY($1) ORDER BY position ASC`, taskIDs)
	if err != nil {
		return fmt.Errorf("batch loading subtasks: %w", err)
	}
	defer rows.Close()

	subtaskMap := make(map[uuid.UUID][]Subtask)
	for rows.Next() {
		var taskID uuid.UUID
		var s Subtask
		if err := rows.Scan(&taskID, &s.Id, &s.Title, &s.IsCompleted, &s.Position); err != nil {
			return fmt.Errorf("scanning batch subtask: %w", err)
		}
		subtaskMap[taskID] = append(subtaskMap[taskID], s)
	}
	for i := range tasks {
		if subs, ok := subtaskMap[tasks[i].Id]; ok {
			tasks[i].Subtasks = &subs
		}
	}
	return nil
}
