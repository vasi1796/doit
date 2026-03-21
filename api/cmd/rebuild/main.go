// Projection rebuilder — replays the entire event log to reconstruct all read model tables.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/projection"
)

const batchSize = 1000

// readModelTables lists tables to truncate before rebuilding, in FK-safe order.
var readModelTables = []string{
	"subtasks",
	"task_labels",
	"tasks",
	"labels",
	"lists",
	"aggregate_snapshots",
}

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "rebuild").Logger()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Fatal().Msg("DATABASE_URL is required")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	store := eventstore.New(pool, logger)
	projector := projection.New(pool, logger)

	if err := rebuild(ctx, pool, store, projector, logger); err != nil {
		logger.Fatal().Err(err).Msg("rebuild failed")
	}
}

func rebuild(ctx context.Context, pool *pgxpool.Pool, store *eventstore.Store, projector *projection.Projector, logger zerolog.Logger) error {
	// Count events for progress reporting
	eventCount, err := store.CountEvents(ctx)
	if err != nil {
		return fmt.Errorf("counting events: %w", err)
	}
	logger.Info().Int("total_events", eventCount).Msg("starting projection rebuild")

	if eventCount == 0 {
		logger.Info().Msg("no events to replay — nothing to do")
		return nil
	}

	// Truncate read model tables
	for _, table := range readModelTables {
		if _, err := pool.Exec(ctx, "TRUNCATE "+table+" CASCADE"); err != nil {
			return fmt.Errorf("truncating %s: %w", table, err)
		}
	}
	logger.Info().Int("tables", len(readModelTables)).Msg("read model tables truncated")

	// Replay all events through the projector
	start := time.Now()
	total, err := store.LoadAllBatched(ctx, batchSize, func(batch []eventstore.Event) error {
		return projector.Project(ctx, batch)
	})
	if err != nil {
		return fmt.Errorf("replaying events: %w", err)
	}

	elapsed := time.Since(start)
	logger.Info().
		Int("events_processed", total).
		Dur("duration", elapsed).
		Float64("events_per_sec", float64(total)/elapsed.Seconds()).
		Msg("projection rebuild complete")

	return nil
}
