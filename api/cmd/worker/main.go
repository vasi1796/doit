// Projection worker — consumes events from RabbitMQ and updates read models.
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
	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/projection"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "worker-projection").Logger()

	cfg, err := config.Load()
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

	projector := projection.New(pool, logger)

	deliveries, err := b.Consume(broker.QueueProjections)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start consuming")
	}

	logger.Info().Msg("projection worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("projection worker shutting down")
			return
		case msg, ok := <-deliveries:
			if !ok {
				logger.Warn().Msg("delivery channel closed, reconnecting")
				return
			}

			var em broker.EventMessage
			if err := json.Unmarshal(msg.Body, &em); err != nil {
				logger.Error().Err(err).Msg("unmarshal event message")
				msg.Nack(false, false) // to DLQ
				continue
			}

			event := eventstore.Event{
				ID:            em.EventID,
				AggregateID:   em.AggregateID,
				AggregateType: eventstore.AggregateType(em.AggregateType),
				EventType:     eventstore.EventType(em.EventType),
				UserID:        em.UserID,
				Data:          em.Data,
				Timestamp:     em.Timestamp,
				Counter:       em.Counter,
				Version:       em.Version,
			}

			if err := projector.Project(ctx, []eventstore.Event{event}); err != nil {
				logger.Error().Err(err).
					Str("event_type", em.EventType).
					Str("event_id", em.EventID.String()).
					Msg("projection failed")
				msg.Nack(false, false) // to DLQ
				continue
			}

			if err := msg.Ack(false); err != nil {
				logger.Error().Err(err).Msg("ack failed")
			}
		}
	}
}
