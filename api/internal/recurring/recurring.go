// Package recurring creates the next occurrence of a completed recurring task.
package recurring

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/broker"
	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
)

// EventLoader loads the event history for an aggregate.
type EventLoader interface {
	LoadByAggregate(ctx context.Context, aggregateID uuid.UUID) ([]eventstore.Event, error)
}

// TaskCreator dispatches task creation commands.
type TaskCreator interface {
	CreateTask(ctx context.Context, cmd domain.CreateTask) error
}

// taskIDNamespace seeds deterministic next-occurrence IDs. Never change it:
// a different namespace would break redelivery deduplication for in-flight events.
var taskIDNamespace = uuid.MustParse("9e2f3f9a-1b6c-4b8e-9f0d-7c2a5d4e6f81")

// NextTaskID derives the next occurrence's task ID from the TaskCompleted
// event that triggered it, so a redelivered message replays into the same
// aggregate ID and is rejected by the event store's version constraint.
func NextTaskID(completedEventID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(taskIDNamespace, completedEventID[:])
}

// Handle creates the next occurrence for a completed recurring task as a
// single atomic command. Redeliveries are detected via the deterministic task
// ID and acknowledged as already processed.
func Handle(ctx context.Context, store EventLoader, cmds TaskCreator, em broker.EventMessage, logger zerolog.Logger) error {
	events, err := store.LoadByAggregate(ctx, em.AggregateID)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	agg := domain.NewTaskAggregate()
	for _, e := range events {
		agg.Apply(e)
	}

	if agg.RecurrenceRule() == domain.RecurrenceNone || agg.DueDate() == nil {
		return nil
	}

	nextDue := domain.NextDueDate(*agg.DueDate(), agg.RecurrenceRule())

	cmd := domain.CreateTask{
		TaskID:         NextTaskID(em.EventID),
		UserID:         em.UserID,
		Title:          agg.Title(),
		Description:    agg.Description(),
		Priority:       agg.Priority(),
		DueDate:        &nextDue,
		DueTime:        agg.DueTime(),
		ListID:         agg.ListID(),
		Position:       agg.Position(),
		RecurrenceRule: agg.RecurrenceRule(),
		Labels:         agg.Labels(),
	}

	if err := cmds.CreateTask(ctx, cmd); err != nil {
		if errors.Is(err, domain.ErrVersionConflict) {
			logger.Debug().
				Str("original_task", em.AggregateID.String()).
				Str("new_task", cmd.TaskID.String()).
				Msg("next occurrence already created, skipping redelivery")
			return nil
		}
		return err
	}

	logger.Info().
		Str("original_task", em.AggregateID.String()).
		Str("new_task", cmd.TaskID.String()).
		Time("next_due", nextDue).
		Msg("recurring task created")

	return nil
}
