package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
)

// executor is satisfied by both *pgxpool.Pool and pgx.Tx, allowing handlers
// to work within a transaction or directly against the pool.
type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

const upsertTaskSQL = `INSERT INTO tasks (id, user_id, list_id, title, description, priority, due_date, due_time, position, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
ON CONFLICT (id) DO UPDATE SET
	user_id = EXCLUDED.user_id, list_id = EXCLUDED.list_id, title = EXCLUDED.title,
	description = EXCLUDED.description, priority = EXCLUDED.priority, due_date = EXCLUDED.due_date,
	due_time = EXCLUDED.due_time, position = EXCLUDED.position, updated_at = EXCLUDED.updated_at`

const updateTaskCompletedSQL = `UPDATE tasks SET is_completed = true, completed_at = $2, updated_at = $3 WHERE id = $1`
const updateTaskUncompletedSQL = `UPDATE tasks SET is_completed = false, completed_at = NULL, updated_at = $2 WHERE id = $1`
const updateTaskDeletedSQL = `UPDATE tasks SET is_deleted = true, deleted_at = $2, updated_at = $3 WHERE id = $1`
const updateTaskRestoredSQL = `UPDATE tasks SET is_deleted = false, deleted_at = NULL, updated_at = $2 WHERE id = $1`
const updateTaskMovedSQL = `UPDATE tasks SET list_id = $2, position = $3, updated_at = $4 WHERE id = $1`
const updateTaskTitleSQL = `UPDATE tasks SET title = $2, updated_at = $3 WHERE id = $1`
const updateTaskPrioritySQL = `UPDATE tasks SET priority = $2, updated_at = $3 WHERE id = $1`
const updateTaskDueDateSQL = `UPDATE tasks SET due_date = $2, updated_at = $3 WHERE id = $1`
const updateTaskDescriptionSQL = `UPDATE tasks SET description = $2, updated_at = $3 WHERE id = $1`
const updateTaskRecurrenceSQL = `UPDATE tasks SET recurrence_rule = $2, updated_at = $3 WHERE id = $1`
const updateTaskDueTimeSQL = `UPDATE tasks SET due_time = $2, updated_at = $3 WHERE id = $1`
const updateTaskPositionSQL = `UPDATE tasks SET position = $2, updated_at = $3 WHERE id = $1`

const upsertTaskLabelSQL = `INSERT INTO task_labels (task_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
const deleteTaskLabelSQL = `DELETE FROM task_labels WHERE task_id = $1 AND label_id = $2`

const upsertListSQL = `INSERT INTO lists (id, user_id, name, colour, icon, position, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
ON CONFLICT (id) DO UPDATE SET
	user_id = EXCLUDED.user_id, name = EXCLUDED.name, colour = EXCLUDED.colour,
	icon = EXCLUDED.icon, position = EXCLUDED.position, updated_at = EXCLUDED.updated_at`

const upsertLabelSQL = `INSERT INTO labels (id, user_id, name, colour, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
	user_id = EXCLUDED.user_id, name = EXCLUDED.name, colour = EXCLUDED.colour`

const upsertSubtaskSQL = `INSERT INTO subtasks (id, task_id, title, position, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
	title = EXCLUDED.title, position = EXCLUDED.position`

const updateSubtaskTitleSQL = `UPDATE subtasks SET title = $2 WHERE id = $1`
const updateSubtaskCompletedSQL = `UPDATE subtasks SET is_completed = true WHERE id = $1`
const updateSubtaskUncompletedSQL = `UPDATE subtasks SET is_completed = false WHERE id = $1`

const deleteListSQL = `DELETE FROM lists WHERE id = $1`
const moveTasksToInboxSQL = `UPDATE tasks SET list_id = NULL WHERE list_id = $1`
const deleteLabelSQL = `DELETE FROM labels WHERE id = $1`
const deleteTaskLabelsByLabelSQL = `DELETE FROM task_labels WHERE label_id = $1`

// Projector consumes events and updates read model tables.
type Projector struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// New creates a new Projector backed by the given connection pool.
func New(pool *pgxpool.Pool, logger zerolog.Logger) *Projector {
	return &Projector{pool: pool, logger: logger}
}

// execProjection is a generic helper that encapsulates the common projection pattern:
// unmarshal JSON payload into a typed struct, execute SQL, and warn if 0 rows affected.
func execProjection[T any](ctx context.Context, exec executor, logger zerolog.Logger, e eventstore.Event, sql string, argsFn func(T) []any, label string) error {
	var payload T
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling %s: %w", label, err)
	}
	tag, err := exec.Exec(ctx, sql, argsFn(payload)...)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if tag.RowsAffected() == 0 {
		logger.Warn().Stringer("aggregate_id", e.AggregateID).Msgf("projection: %s affected 0 rows", label)
	}
	return nil
}

// execProjectionDirect handles the simple case where no payload needs unmarshaling.
func execProjectionDirect(ctx context.Context, exec executor, logger zerolog.Logger, e eventstore.Event, sql string, args []any, label string) error {
	tag, err := exec.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if tag.RowsAffected() == 0 {
		logger.Warn().Stringer("aggregate_id", e.AggregateID).Msgf("projection: %s affected 0 rows", label)
	}
	return nil
}

// Project processes a batch of events within a single transaction,
// updating read models for each. The transaction ensures atomicity —
// either all projections in the batch succeed or none do.
func (p *Projector) Project(ctx context.Context, events []eventstore.Event) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning projection tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			p.logger.Error().Err(rbErr).Msg("projection: rollback failed")
		}
	}()

	for _, e := range events {
		if err := p.handleEvent(ctx, tx, e); err != nil {
			return fmt.Errorf("projecting event %s (%s): %w", e.ID, e.EventType, err)
		}
	}

	return tx.Commit(ctx)
}

func (p *Projector) handleEvent(ctx context.Context, tx pgx.Tx, e eventstore.Event) error {
	switch e.EventType {
	case eventstore.EventTaskCreated:
		return p.handleTaskCreated(ctx, tx, e)
	case eventstore.EventTaskCompleted:
		return execProjection(ctx, tx, p.logger, e, updateTaskCompletedSQL, func(pl domain.TaskCompletedPayload) []any {
			return []any{e.AggregateID, pl.CompletedAt, e.Timestamp}
		}, "TaskCompleted")
	case eventstore.EventTaskUncompleted:
		return execProjectionDirect(ctx, tx, p.logger, e, updateTaskUncompletedSQL, []any{e.AggregateID, e.Timestamp}, "TaskUncompleted")
	case eventstore.EventTaskDeleted:
		return execProjection(ctx, tx, p.logger, e, updateTaskDeletedSQL, func(pl domain.TaskDeletedPayload) []any {
			return []any{e.AggregateID, pl.DeletedAt, e.Timestamp}
		}, "TaskDeleted")
	case eventstore.EventTaskRestored:
		return execProjectionDirect(ctx, tx, p.logger, e, updateTaskRestoredSQL, []any{e.AggregateID, e.Timestamp}, "TaskRestored")
	case eventstore.EventTaskMoved:
		return execProjection(ctx, tx, p.logger, e, updateTaskMovedSQL, func(pl domain.TaskMovedPayload) []any {
			return []any{e.AggregateID, pl.ListID, pl.Position, e.Timestamp}
		}, "TaskMoved")
	case eventstore.EventTaskDescriptionUpdated:
		return execProjection(ctx, tx, p.logger, e, updateTaskDescriptionSQL, func(pl domain.TaskDescriptionUpdatedPayload) []any {
			return []any{e.AggregateID, pl.Description, e.Timestamp}
		}, "TaskDescriptionUpdated")
	case eventstore.EventTaskTitleUpdated:
		return execProjection(ctx, tx, p.logger, e, updateTaskTitleSQL, func(pl domain.TaskTitleUpdatedPayload) []any {
			return []any{e.AggregateID, pl.Title, e.Timestamp}
		}, "TaskTitleUpdated")
	case eventstore.EventTaskPriorityUpdated:
		return execProjection(ctx, tx, p.logger, e, updateTaskPrioritySQL, func(pl domain.TaskPriorityUpdatedPayload) []any {
			return []any{e.AggregateID, pl.Priority, e.Timestamp}
		}, "TaskPriorityUpdated")
	case eventstore.EventTaskDueDateUpdated:
		return execProjection(ctx, tx, p.logger, e, updateTaskDueDateSQL, func(pl domain.TaskDueDateUpdatedPayload) []any {
			return []any{e.AggregateID, pl.DueDate, e.Timestamp}
		}, "TaskDueDateUpdated")
	case eventstore.EventLabelAdded:
		return execProjection(ctx, tx, p.logger, e, upsertTaskLabelSQL, func(pl domain.LabelAddedPayload) []any {
			return []any{e.AggregateID, pl.LabelID}
		}, "LabelAdded")
	case eventstore.EventLabelRemoved:
		return execProjection(ctx, tx, p.logger, e, deleteTaskLabelSQL, func(pl domain.LabelRemovedPayload) []any {
			return []any{e.AggregateID, pl.LabelID}
		}, "LabelRemoved")
	case eventstore.EventListCreated:
		return p.handleListCreated(ctx, tx, e)
	case eventstore.EventListDeleted:
		return p.handleListDeleted(ctx, tx, e)
	case eventstore.EventLabelCreated:
		return p.handleLabelCreated(ctx, tx, e)
	case eventstore.EventLabelDeleted:
		return p.handleLabelDeleted(ctx, tx, e)
	case eventstore.EventSubtaskCreated:
		return p.handleSubtaskCreated(ctx, tx, e)
	case eventstore.EventSubtaskTitleUpdated:
		return execProjection(ctx, tx, p.logger, e, updateSubtaskTitleSQL, func(pl domain.SubtaskTitleUpdatedPayload) []any {
			return []any{pl.SubtaskID, pl.Title}
		}, "SubtaskTitleUpdated")
	case eventstore.EventSubtaskCompleted:
		return execProjection(ctx, tx, p.logger, e, updateSubtaskCompletedSQL, func(pl domain.SubtaskCompletedPayload) []any {
			return []any{pl.SubtaskID}
		}, "SubtaskCompleted")
	case eventstore.EventSubtaskUncompleted:
		return execProjection(ctx, tx, p.logger, e, updateSubtaskUncompletedSQL, func(pl domain.SubtaskUncompletedPayload) []any {
			return []any{pl.SubtaskID}
		}, "SubtaskUncompleted")
	case eventstore.EventTaskRecurrenceUpdated:
		return execProjection(ctx, tx, p.logger, e, updateTaskRecurrenceSQL, func(pl domain.TaskRecurrenceUpdatedPayload) []any {
			return []any{e.AggregateID, pl.RecurrenceRule, e.Timestamp}
		}, "TaskRecurrenceUpdated")
	case eventstore.EventTaskDueTimeUpdated:
		return execProjection(ctx, tx, p.logger, e, updateTaskDueTimeSQL, func(pl domain.TaskDueTimeUpdatedPayload) []any {
			return []any{e.AggregateID, pl.DueTime, e.Timestamp}
		}, "TaskDueTimeUpdated")
	case eventstore.EventTaskReordered:
		return execProjection(ctx, tx, p.logger, e, updateTaskPositionSQL, func(pl domain.TaskReorderedPayload) []any {
			return []any{e.AggregateID, pl.Position, e.Timestamp}
		}, "TaskReordered")
	default:
		// Unknown event types are silently skipped for forward compatibility.
		return nil
	}
}

func (p *Projector) handleTaskCreated(ctx context.Context, tx pgx.Tx, e eventstore.Event) error {
	var payload domain.TaskCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskCreatedPayload: %w", err)
	}
	_, err := tx.Exec(ctx, upsertTaskSQL,
		e.AggregateID, e.UserID, payload.ListID, payload.Title,
		payload.Description, payload.Priority, payload.DueDate,
		payload.DueTime, payload.Position, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting task: %w", err)
	}
	return nil
}

func (p *Projector) handleListCreated(ctx context.Context, tx pgx.Tx, e eventstore.Event) error {
	var payload domain.ListCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling ListCreatedPayload: %w", err)
	}
	_, err := tx.Exec(ctx, upsertListSQL,
		e.AggregateID, e.UserID, payload.Name, payload.Colour,
		payload.Icon, payload.Position, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting list: %w", err)
	}
	return nil
}

func (p *Projector) handleLabelCreated(ctx context.Context, tx pgx.Tx, e eventstore.Event) error {
	var payload domain.LabelCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling LabelCreatedPayload: %w", err)
	}
	_, err := tx.Exec(ctx, upsertLabelSQL,
		e.AggregateID, e.UserID, payload.Name, payload.Colour, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting label: %w", err)
	}
	return nil
}

func (p *Projector) handleSubtaskCreated(ctx context.Context, tx pgx.Tx, e eventstore.Event) error {
	var payload domain.SubtaskCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling SubtaskCreatedPayload: %w", err)
	}
	_, err := tx.Exec(ctx, upsertSubtaskSQL,
		payload.SubtaskID, e.AggregateID, payload.Title, payload.Position, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting subtask: %w", err)
	}
	return nil
}

func (p *Projector) handleListDeleted(ctx context.Context, tx pgx.Tx, e eventstore.Event) error {
	// Move tasks belonging to this list to inbox before deleting — atomic within the batch tx.
	if _, err := tx.Exec(ctx, moveTasksToInboxSQL, e.AggregateID); err != nil {
		return fmt.Errorf("moving tasks to inbox on list delete: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteListSQL, e.AggregateID); err != nil {
		return fmt.Errorf("deleting list: %w", err)
	}
	return nil
}

func (p *Projector) handleLabelDeleted(ctx context.Context, tx pgx.Tx, e eventstore.Event) error {
	// Remove label associations before deleting the label — atomic within the batch tx.
	if _, err := tx.Exec(ctx, deleteTaskLabelsByLabelSQL, e.AggregateID); err != nil {
		return fmt.Errorf("deleting task label associations: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteLabelSQL, e.AggregateID); err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}
	return nil
}
