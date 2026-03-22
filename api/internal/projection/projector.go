package projection

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
)

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

// Project processes a batch of events, updating read models for each.
func (p *Projector) Project(ctx context.Context, events []eventstore.Event) error {
	for _, e := range events {
		if err := p.handleEvent(ctx, e); err != nil {
			return fmt.Errorf("projecting event %s (%s): %w", e.ID, e.EventType, err)
		}
	}
	return nil
}

func (p *Projector) handleEvent(ctx context.Context, e eventstore.Event) error {
	switch e.EventType {
	case eventstore.EventTaskCreated:
		return p.handleTaskCreated(ctx, e)
	case eventstore.EventTaskCompleted:
		return p.handleTaskCompleted(ctx, e)
	case eventstore.EventTaskUncompleted:
		return p.handleTaskUncompleted(ctx, e)
	case eventstore.EventTaskDeleted:
		return p.handleTaskDeleted(ctx, e)
	case eventstore.EventTaskRestored:
		return p.handleTaskRestored(ctx, e)
	case eventstore.EventTaskMoved:
		return p.handleTaskMoved(ctx, e)
	case eventstore.EventTaskDescriptionUpdated:
		return p.handleTaskDescriptionUpdated(ctx, e)
	case eventstore.EventTaskTitleUpdated:
		return p.handleTaskTitleUpdated(ctx, e)
	case eventstore.EventTaskPriorityUpdated:
		return p.handleTaskPriorityUpdated(ctx, e)
	case eventstore.EventTaskDueDateUpdated:
		return p.handleTaskDueDateUpdated(ctx, e)
	case eventstore.EventLabelAdded:
		return p.handleLabelAdded(ctx, e)
	case eventstore.EventLabelRemoved:
		return p.handleLabelRemoved(ctx, e)
	case eventstore.EventListCreated:
		return p.handleListCreated(ctx, e)
	case eventstore.EventListDeleted:
		return p.handleListDeleted(ctx, e)
	case eventstore.EventLabelCreated:
		return p.handleLabelCreated(ctx, e)
	case eventstore.EventLabelDeleted:
		return p.handleLabelDeleted(ctx, e)
	case eventstore.EventSubtaskCreated:
		return p.handleSubtaskCreated(ctx, e)
	case eventstore.EventSubtaskTitleUpdated:
		return p.handleSubtaskTitleUpdated(ctx, e)
	case eventstore.EventSubtaskCompleted:
		return p.handleSubtaskCompleted(ctx, e)
	case eventstore.EventSubtaskUncompleted:
		return p.handleSubtaskUncompleted(ctx, e)
	case eventstore.EventTaskRecurrenceUpdated:
		return p.handleTaskRecurrenceUpdated(ctx, e)
	case eventstore.EventTaskDueTimeUpdated:
		return p.handleTaskDueTimeUpdated(ctx, e)
	case eventstore.EventTaskReordered:
		return p.handleTaskReordered(ctx, e)
	default:
		// Unknown event types are silently skipped for forward compatibility.
		return nil
	}
}

func (p *Projector) handleTaskCreated(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskCreatedPayload: %w", err)
	}
	_, err := p.pool.Exec(ctx, upsertTaskSQL,
		e.AggregateID, e.UserID, payload.ListID, payload.Title,
		payload.Description, payload.Priority, payload.DueDate,
		payload.DueTime, payload.Position, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting task: %w", err)
	}
	return nil
}

func (p *Projector) handleTaskCompleted(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskCompletedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskCompletedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskCompletedSQL, e.AggregateID, payload.CompletedAt, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task completed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskCompleted affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskUncompleted(ctx context.Context, e eventstore.Event) error {
	tag, err := p.pool.Exec(ctx, updateTaskUncompletedSQL, e.AggregateID, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task uncompleted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskUncompleted affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskDeleted(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskDeletedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskDeletedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskDeletedSQL, e.AggregateID, payload.DeletedAt, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task deleted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskDeleted affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskRestored(ctx context.Context, e eventstore.Event) error {
	tag, err := p.pool.Exec(ctx, updateTaskRestoredSQL, e.AggregateID, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task restored: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskRestored affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskMoved(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskMovedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskMovedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskMovedSQL, e.AggregateID, payload.ListID, payload.Position, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task moved: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskMoved affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskDescriptionUpdated(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskDescriptionUpdatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskDescriptionUpdatedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskDescriptionSQL, e.AggregateID, payload.Description, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task description: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskDescriptionUpdated affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskTitleUpdated(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskTitleUpdatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskTitleUpdatedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskTitleSQL, e.AggregateID, payload.Title, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task title: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskTitleUpdated affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskPriorityUpdated(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskPriorityUpdatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskPriorityUpdatedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskPrioritySQL, e.AggregateID, payload.Priority, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task priority: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskPriorityUpdated affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskDueDateUpdated(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskDueDateUpdatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskDueDateUpdatedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskDueDateSQL, e.AggregateID, payload.DueDate, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task due date: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskDueDateUpdated affected 0 rows")
	}
	return nil
}

func (p *Projector) handleLabelAdded(ctx context.Context, e eventstore.Event) error {
	var payload domain.LabelAddedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling LabelAddedPayload: %w", err)
	}
	_, err := p.pool.Exec(ctx, upsertTaskLabelSQL, e.AggregateID, payload.LabelID)
	if err != nil {
		return fmt.Errorf("upserting task label: %w", err)
	}
	return nil
}

func (p *Projector) handleLabelRemoved(ctx context.Context, e eventstore.Event) error {
	var payload domain.LabelRemovedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling LabelRemovedPayload: %w", err)
	}
	_, err := p.pool.Exec(ctx, deleteTaskLabelSQL, e.AggregateID, payload.LabelID)
	if err != nil {
		return fmt.Errorf("deleting task label: %w", err)
	}
	return nil
}

func (p *Projector) handleListCreated(ctx context.Context, e eventstore.Event) error {
	var payload domain.ListCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling ListCreatedPayload: %w", err)
	}
	_, err := p.pool.Exec(ctx, upsertListSQL,
		e.AggregateID, e.UserID, payload.Name, payload.Colour,
		payload.Icon, payload.Position, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting list: %w", err)
	}
	return nil
}

func (p *Projector) handleLabelCreated(ctx context.Context, e eventstore.Event) error {
	var payload domain.LabelCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling LabelCreatedPayload: %w", err)
	}
	_, err := p.pool.Exec(ctx, upsertLabelSQL,
		e.AggregateID, e.UserID, payload.Name, payload.Colour, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting label: %w", err)
	}
	return nil
}

func (p *Projector) handleSubtaskCreated(ctx context.Context, e eventstore.Event) error {
	var payload domain.SubtaskCreatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling SubtaskCreatedPayload: %w", err)
	}
	_, err := p.pool.Exec(ctx, upsertSubtaskSQL,
		payload.SubtaskID, e.AggregateID, payload.Title, payload.Position, e.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("upserting subtask: %w", err)
	}
	return nil
}

func (p *Projector) handleSubtaskCompleted(ctx context.Context, e eventstore.Event) error {
	var payload domain.SubtaskCompletedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling SubtaskCompletedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateSubtaskCompletedSQL, payload.SubtaskID)
	if err != nil {
		return fmt.Errorf("updating subtask completed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("subtask_id", payload.SubtaskID).Msg("projection: SubtaskCompleted affected 0 rows")
	}
	return nil
}

func (p *Projector) handleSubtaskUncompleted(ctx context.Context, e eventstore.Event) error {
	var payload domain.SubtaskUncompletedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling SubtaskUncompletedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateSubtaskUncompletedSQL, payload.SubtaskID)
	if err != nil {
		return fmt.Errorf("updating subtask uncompleted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("subtask_id", payload.SubtaskID).Msg("projection: SubtaskUncompleted affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskRecurrenceUpdated(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskRecurrenceUpdatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskRecurrenceUpdatedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskRecurrenceSQL, e.AggregateID, payload.RecurrenceRule, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task recurrence: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskRecurrenceUpdated affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskDueTimeUpdated(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskDueTimeUpdatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskDueTimeUpdatedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskDueTimeSQL, e.AggregateID, payload.DueTime, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task due time: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskDueTimeUpdated affected 0 rows")
	}
	return nil
}

func (p *Projector) handleTaskReordered(ctx context.Context, e eventstore.Event) error {
	var payload domain.TaskReorderedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling TaskReorderedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateTaskPositionSQL, e.AggregateID, payload.Position, e.Timestamp)
	if err != nil {
		return fmt.Errorf("updating task position: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("aggregate_id", e.AggregateID).Msg("projection: TaskReordered affected 0 rows")
	}
	return nil
}

func (p *Projector) handleSubtaskTitleUpdated(ctx context.Context, e eventstore.Event) error {
	var payload domain.SubtaskTitleUpdatedPayload
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return fmt.Errorf("unmarshaling SubtaskTitleUpdatedPayload: %w", err)
	}
	tag, err := p.pool.Exec(ctx, updateSubtaskTitleSQL, payload.SubtaskID, payload.Title)
	if err != nil {
		return fmt.Errorf("updating subtask title: %w", err)
	}
	if tag.RowsAffected() == 0 {
		p.logger.Warn().Stringer("subtask_id", payload.SubtaskID).Msg("projection: SubtaskTitleUpdated affected 0 rows")
	}
	return nil
}

func (p *Projector) handleListDeleted(ctx context.Context, e eventstore.Event) error {
	// Move tasks belonging to this list to inbox before deleting
	if _, err := p.pool.Exec(ctx, moveTasksToInboxSQL, e.AggregateID); err != nil {
		return fmt.Errorf("moving tasks to inbox on list delete: %w", err)
	}
	if _, err := p.pool.Exec(ctx, deleteListSQL, e.AggregateID); err != nil {
		return fmt.Errorf("deleting list: %w", err)
	}
	return nil
}

func (p *Projector) handleLabelDeleted(ctx context.Context, e eventstore.Event) error {
	// Remove label associations before deleting the label
	if _, err := p.pool.Exec(ctx, deleteTaskLabelsByLabelSQL, e.AggregateID); err != nil {
		return fmt.Errorf("deleting task label associations: %w", err)
	}
	if _, err := p.pool.Exec(ctx, deleteLabelSQL, e.AggregateID); err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}
	return nil
}
