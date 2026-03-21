package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
)

// EventStore is the interface the domain needs from the event store.
type EventStore interface {
	LoadByAggregate(ctx context.Context, aggregateID uuid.UUID) ([]eventstore.Event, error)
	AppendTx(ctx context.Context, tx pgx.Tx, events []eventstore.Event) error
	InsertOutbox(ctx context.Context, tx pgx.Tx, events []eventstore.Event) error
	Append(ctx context.Context, events []eventstore.Event) error
}

// TxBeginner starts database transactions.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// CommandHandler orchestrates the load-validate-append cycle for all commands.
// Projections are no longer called inline — events flow through the outbox
// to RabbitMQ where workers project them asynchronously.
type CommandHandler struct {
	store EventStore
	pool  TxBeginner
	clock *hlc.Clock
}

func NewCommandHandler(store EventStore, pool TxBeginner, clock *hlc.Clock) *CommandHandler {
	return &CommandHandler{store: store, pool: pool, clock: clock}
}

// appendWithOutbox atomically appends events and creates outbox rows in a single transaction.
func (h *CommandHandler) appendWithOutbox(ctx context.Context, events []eventstore.Event) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := h.store.AppendTx(ctx, tx, events); err != nil {
		if errors.Is(err, eventstore.ErrVersionConflict) {
			return ErrVersionConflict
		}
		return err
	}
	if err := h.store.InsertOutbox(ctx, tx, events); err != nil {
		return fmt.Errorf("inserting outbox: %w", err)
	}
	return tx.Commit(ctx)
}

// Task commands

func (h *CommandHandler) CreateTask(ctx context.Context, cmd CreateTask) error {
	agg := NewTaskAggregate()
	events, err := agg.HandleCreate(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) CompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd CompleteTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleComplete(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UncompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UncompleteTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUncomplete(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) DeleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd DeleteTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleDelete(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) RestoreTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd RestoreTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleRestore(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) MoveTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd MoveTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleMove(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UpdateTaskDescription(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateTaskDescription) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdateDescription(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UpdateTaskTitle(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateTaskTitle) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdateTitle(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UpdateTaskPriority(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateTaskPriority) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdatePriority(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UpdateTaskDueDate(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateTaskDueDate) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdateDueDate(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UpdateTaskDueTime(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateTaskDueTime) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdateDueTime(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UpdateTaskRecurrence(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateTaskRecurrence) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdateRecurrence(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) AddLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd AddLabel) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleAddLabel(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) RemoveLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd RemoveLabel) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleRemoveLabel(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) CreateSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd CreateSubtask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleCreateSubtask(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UpdateSubtaskTitle(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateSubtaskTitle) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdateSubtaskTitle(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) CompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd CompleteSubtask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleCompleteSubtask(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) UncompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UncompleteSubtask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUncompleteSubtask(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

// List commands

func (h *CommandHandler) CreateList(ctx context.Context, cmd CreateList) error {
	agg := NewListAggregate()
	events, err := agg.HandleCreate(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) DeleteList(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd DeleteList) error {
	agg, err := h.loadListAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleDelete(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

// Label commands

func (h *CommandHandler) CreateLabel(ctx context.Context, cmd CreateLabel) error {
	agg := NewLabelAggregate()
	events, err := agg.HandleCreate(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

func (h *CommandHandler) DeleteLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd DeleteLabel) error {
	agg, err := h.loadLabelAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleDelete(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendWithOutbox(ctx, events)
}

// Aggregate loaders

func (h *CommandHandler) loadTaskAggregate(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*TaskAggregate, error) {
	stored, err := h.store.LoadByAggregate(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, ErrTaskNotFound
	}
	agg := NewTaskAggregate()
	for _, e := range stored {
		agg.Apply(e)
	}
	if agg.UserID() != userID {
		return nil, ErrTaskNotFound
	}
	return agg, nil
}

func (h *CommandHandler) loadListAggregate(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*ListAggregate, error) {
	stored, err := h.store.LoadByAggregate(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, ErrListNotFound
	}
	agg := NewListAggregate()
	for _, e := range stored {
		agg.Apply(e)
	}
	if agg.UserID() != userID {
		return nil, ErrListNotFound
	}
	return agg, nil
}

func (h *CommandHandler) loadLabelAggregate(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*LabelAggregate, error) {
	stored, err := h.store.LoadByAggregate(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, ErrLabelNotFound
	}
	agg := NewLabelAggregate()
	for _, e := range stored {
		agg.Apply(e)
	}
	if agg.UserID() != userID {
		return nil, ErrLabelNotFound
	}
	return agg, nil
}
