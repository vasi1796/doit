// Recurring tasks worker — consumes TaskCompleted events and creates next occurrences.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/broker"
	"github.com/vasi1796/doit/internal/config"
	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "worker-recurring").Logger()

	cfg, err := config.LoadWorker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	if cfg.RabbitMQURL == "" {
		logger.Fatal().Msg("RABBITMQ_URL is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	b, err := broker.New(cfg.RabbitMQURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to RabbitMQ")
	}
	defer b.Close()

	if err := b.Setup(); err != nil {
		logger.Fatal().Err(err).Msg("failed to setup RabbitMQ")
	}

	store := eventstore.New(pool, logger)
	clock := hlc.New()
	cmdHandler := domain.NewCommandHandler(store, pool, clock)

	logger.Info().Msg("recurring tasks worker started")

	for {
		deliveries, err := b.Consume(broker.QueueRecurring, 1)
		if err != nil {
			logger.Error().Err(err).Msg("failed to start consuming, waiting for reconnect")
			select {
			case <-ctx.Done():
				logger.Info().Msg("recurring tasks worker shutting down")
				return
			case <-b.Reconnected():
				continue
			}
		}

		reconnected := b.Reconnected()

		done := false
		for !done {
			select {
			case <-ctx.Done():
				logger.Info().Msg("recurring tasks worker shutting down")
				return
			case <-reconnected:
				logger.Info().Msg("broker reconnected, re-subscribing")
				done = true
			case msg, ok := <-deliveries:
				if !ok {
					logger.Warn().Msg("delivery channel closed, waiting for reconnect")
					select {
					case <-ctx.Done():
						logger.Info().Msg("recurring tasks worker shutting down")
						return
					case <-reconnected:
						done = true
					}
					continue
				}

				var em broker.EventMessage
				if err := json.Unmarshal(msg.Body, &em); err != nil {
					logger.Error().Err(err).Msg("unmarshal event message")
					msg.Nack(false, false)
					continue
				}

				if em.EventType != string(eventstore.EventTaskCompleted) {
					msg.Ack(false)
					continue
				}

				if err := handleRecurring(ctx, store, cmdHandler, em, logger); err != nil {
					logger.Error().Err(err).Str("aggregate_id", em.AggregateID.String()).Msg("recurring task creation failed")
					msg.Nack(false, false)
					continue
				}

				msg.Ack(false)
			}
		}
	}
}

func handleRecurring(ctx context.Context, store *eventstore.Store, cmdHandler *domain.CommandHandler, em broker.EventMessage, logger zerolog.Logger) error {
	// Load the completed task's aggregate to check recurrence
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

	if agg.RecurrenceRule() == "" || agg.DueDate() == nil {
		return nil // Not a recurring task
	}

	nextDue := domain.NextDueDate(*agg.DueDate(), agg.RecurrenceRule())

	cmd := domain.CreateTask{
		TaskID:      domain.NewID(),
		UserID:      em.UserID,
		Title:       agg.Title(),
		Description: agg.Description(),
		Priority:    agg.Priority(),
		DueDate:     &nextDue,
		DueTime:     agg.DueTime(),
		ListID:      agg.ListID(),
		Position:    agg.Position(),
	}

	if err := cmdHandler.CreateTask(ctx, cmd); err != nil {
		return err
	}

	// Set recurrence rule on the new task
	if err := cmdHandler.UpdateTaskRecurrence(ctx, cmd.TaskID, em.UserID, domain.UpdateTaskRecurrence{
		RecurrenceRule: agg.RecurrenceRule(),
	}); err != nil {
		logger.Warn().Err(err).Msg("failed to set recurrence on new task")
	}

	// Copy labels from original task
	for _, labelID := range agg.Labels() {
		if err := cmdHandler.AddLabel(ctx, cmd.TaskID, em.UserID, domain.AddLabel{LabelID: labelID}); err != nil {
			logger.Warn().Err(err).Str("label_id", labelID.String()).Msg("failed to copy label to new task")
		}
	}

	logger.Info().
		Str("original_task", em.AggregateID.String()).
		Str("new_task", cmd.TaskID.String()).
		Time("next_due", nextDue).
		Msg("recurring task created")

	return nil
}
