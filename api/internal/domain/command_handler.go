package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

// EventLoader is the interface the domain needs from the event store.
// Defined here (consumer-side) per project conventions.
type EventLoader interface {
	LoadByAggregate(ctx context.Context, aggregateID uuid.UUID) ([]eventstore.Event, error)
	Append(ctx context.Context, events []eventstore.Event) error
}

// EventProjector updates read models from events.
type EventProjector interface {
	Project(ctx context.Context, events []eventstore.Event) error
}

// CommandHandler orchestrates the load-validate-append-project cycle for all commands.
type CommandHandler struct {
	store     EventLoader
	projector EventProjector
}

func NewCommandHandler(store EventLoader, projector EventProjector) *CommandHandler {
	return &CommandHandler{store: store, projector: projector}
}

func (h *CommandHandler) appendAndProject(ctx context.Context, events []eventstore.Event) error {
	if err := h.store.Append(ctx, events); err != nil {
		return err
	}
	if err := h.projector.Project(ctx, events); err != nil {
		// Events are stored but read models are stale.
		// Recovery: rebuild projections from the event store.
		return fmt.Errorf("projection failed (events stored, read models may be stale): %w", err)
	}
	return nil
}

// Task commands

func (h *CommandHandler) CreateTask(ctx context.Context, cmd CreateTask) error {
	agg := NewTaskAggregate()
	events, err := agg.HandleCreate(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) CompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd CompleteTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleComplete(cmd)
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) UncompleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UncompleteTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUncomplete(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) DeleteTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd DeleteTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleDelete(cmd)
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) MoveTask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd MoveTask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleMove(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) UpdateTaskDescription(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd UpdateTaskDescription) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleUpdateDescription(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) AddLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd AddLabel) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleAddLabel(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) RemoveLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd RemoveLabel) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleRemoveLabel(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) CreateSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd CreateSubtask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleCreateSubtask(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

func (h *CommandHandler) CompleteSubtask(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd CompleteSubtask) error {
	agg, err := h.loadTaskAggregate(ctx, aggregateID, userID)
	if err != nil {
		return err
	}
	events, err := agg.HandleCompleteSubtask(cmd)
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

// List commands

func (h *CommandHandler) CreateList(ctx context.Context, cmd CreateList) error {
	agg := NewListAggregate()
	events, err := agg.HandleCreate(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
}

// Label commands

func (h *CommandHandler) CreateLabel(ctx context.Context, cmd CreateLabel) error {
	agg := NewLabelAggregate()
	events, err := agg.HandleCreate(cmd, time.Now().UTC())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
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
	// Don't leak existence of tasks belonging to other users.
	if agg.UserID() != userID {
		return nil, ErrTaskNotFound
	}
	return agg, nil
}
