package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
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
	clock     *hlc.Clock
}

func NewCommandHandler(store EventLoader, projector EventProjector, clock *hlc.Clock) *CommandHandler {
	return &CommandHandler{store: store, projector: projector, clock: clock}
}

func (h *CommandHandler) appendAndProject(ctx context.Context, events []eventstore.Event) error {
	if err := h.store.Append(ctx, events); err != nil {
		if errors.Is(err, eventstore.ErrVersionConflict) {
			return ErrVersionConflict
		}
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
	events, err := agg.HandleCreate(cmd, h.clock.Now())
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
	events, recurring, err := agg.HandleComplete(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	if err := h.appendAndProject(ctx, events); err != nil {
		return err
	}
	// Append recurring task events separately (different aggregate)
	if recurring != nil {
		if err := h.appendAndProject(ctx, recurring.Events); err != nil {
			return fmt.Errorf("creating recurring task: %w", err)
		}
	}
	return nil
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
}

// List commands

func (h *CommandHandler) CreateList(ctx context.Context, cmd CreateList) error {
	agg := NewListAggregate()
	events, err := agg.HandleCreate(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
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
	return h.appendAndProject(ctx, events)
}

// Label commands

func (h *CommandHandler) CreateLabel(ctx context.Context, cmd CreateLabel) error {
	agg := NewLabelAggregate()
	events, err := agg.HandleCreate(cmd, h.clock.Now())
	if err != nil {
		return err
	}
	return h.appendAndProject(ctx, events)
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
