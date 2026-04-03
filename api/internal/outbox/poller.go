// Package outbox polls unpublished events and publishes them to RabbitMQ.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/broker"
	"github.com/vasi1796/doit/internal/eventstore"
)

// Publisher sends messages to the broker.
type Publisher interface {
	Publish(routingKey string, body []byte) error
}

// OutboxStore provides outbox query methods.
type OutboxStore interface {
	ClaimOutbox(ctx context.Context, tx pgx.Tx, batchSize int) ([]eventstore.OutboxEntry, error)
	MarkPublished(ctx context.Context, tx pgx.Tx, ids []int64) error
	CleanupOutbox(ctx context.Context) (int64, error)
}

// Poller reads unpublished outbox entries and publishes them to RabbitMQ.
type Poller struct {
	pool      *pgxpool.Pool
	store     OutboxStore
	publisher Publisher
	logger    zerolog.Logger
	batchSize int
}

func NewPoller(pool *pgxpool.Pool, store OutboxStore, publisher Publisher, logger zerolog.Logger) *Poller {
	return &Poller{
		pool:      pool,
		store:     store,
		publisher: publisher,
		logger:    logger,
		batchSize: 50,
	}
}

// Run polls the outbox at the given interval until the context is cancelled.
// It also runs periodic cleanup of published entries (once per hour).
func (p *Poller) Run(ctx context.Context, interval time.Duration) {
	p.logger.Info().Dur("interval", interval).Msg("outbox poller started")
	pollTicker := time.NewTicker(interval)
	defer pollTicker.Stop()

	cleanupTicker := time.NewTicker(1 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Msg("outbox poller stopped")
			return
		case <-pollTicker.C:
			if err := p.Poll(ctx); err != nil {
				p.logger.Error().Err(err).Msg("outbox poll failed")
			}
		case <-cleanupTicker.C:
			if deleted, err := p.store.CleanupOutbox(ctx); err != nil {
				p.logger.Error().Err(err).Msg("outbox cleanup failed")
			} else if deleted > 0 {
				p.logger.Info().Int64("deleted", deleted).Msg("outbox cleanup completed")
			}
		}
	}
}

// Poll claims a batch of unpublished outbox entries, publishes them, and marks them as published.
func (p *Poller) Poll(ctx context.Context) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	entries, err := p.store.ClaimOutbox(ctx, tx, p.batchSize)
	if err != nil {
		return fmt.Errorf("claim outbox: %w", err)
	}
	if len(entries) == 0 {
		return tx.Commit(ctx)
	}

	var publishedIDs []int64
	for _, entry := range entries {
		msg := broker.EventMessage{
			EventID:       entry.EventID,
			AggregateID:   entry.AggregateID,
			AggregateType: string(entry.AggregateType),
			EventType:     string(entry.EventType),
			UserID:        entry.UserID,
			Data:          entry.Data,
			Timestamp:     entry.Timestamp,
			Counter:       entry.Counter,
			Version:       entry.Version,
		}
		body, err := json.Marshal(msg)
		if err != nil {
			p.logger.Error().Err(err).Str("event_id", entry.EventID.String()).Msg("marshal outbox entry")
			continue
		}

		if err := p.publisher.Publish(string(entry.EventType), body); err != nil {
			p.logger.Error().Err(err).Str("event_id", entry.EventID.String()).Msg("publish outbox entry")
			// Stop publishing this batch — entries stay unpublished for retry
			break
		}
		publishedIDs = append(publishedIDs, entry.ID)
	}

	if len(publishedIDs) > 0 {
		if err := p.store.MarkPublished(ctx, tx, publishedIDs); err != nil {
			return fmt.Errorf("mark published: %w", err)
		}
		p.logger.Debug().Int("count", len(publishedIDs)).Msg("outbox entries published")
	}

	return tx.Commit(ctx)
}
