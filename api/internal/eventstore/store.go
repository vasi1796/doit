package eventstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const pgUniqueViolation = "23505"

const insertSQL = `INSERT INTO events (id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, counter, version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const insertOutboxSQL = `INSERT INTO outbox (event_id, aggregate_id, aggregate_type, event_type, user_id, data)
VALUES ($1, $2, $3, $4, $5, $6)`

const claimOutboxSQL = `SELECT id, event_id, aggregate_id, aggregate_type, event_type, user_id, data, created_at
FROM outbox WHERE published = false ORDER BY created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED`

const markPublishedSQL = `UPDATE outbox SET published = true WHERE id = ANY($1)`

const loadByAggregateSQL = `SELECT id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, counter, version
FROM events WHERE aggregate_id = $1 ORDER BY version ASC`

const loadByAggregateFromVersionSQL = `SELECT id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, counter, version
FROM events WHERE aggregate_id = $1 AND version >= $2 ORDER BY version ASC`

const loadByUserSinceSQL = `SELECT id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, counter, version
FROM events WHERE user_id = $1 AND (timestamp > $2 OR (timestamp = $2 AND counter > $3)) ORDER BY timestamp ASC, counter ASC, version ASC`

// Store provides append and query operations on the event store.
type Store struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// Pool exposes the connection pool for transaction management by callers.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func New(pool *pgxpool.Pool, logger zerolog.Logger) *Store {
	return &Store{pool: pool, logger: logger}
}

func validateSameAggregate(events []Event) error {
	if len(events) == 0 {
		return ErrNoEvents
	}
	aggregateID := events[0].AggregateID
	for _, e := range events[1:] {
		if e.AggregateID != aggregateID {
			return fmt.Errorf("eventstore: all events must belong to the same aggregate")
		}
	}
	return nil
}

// AppendTx inserts events into an existing transaction.
// Used by the transactional outbox pattern where events and outbox rows
// must be committed atomically.
func (s *Store) AppendTx(ctx context.Context, tx pgx.Tx, events []Event) error {
	if err := validateSameAggregate(events); err != nil {
		return err
	}
	for _, e := range events {
		_, err := tx.Exec(ctx, insertSQL,
			e.ID, e.AggregateID, e.AggregateType, e.EventType,
			e.UserID, e.Data, e.Timestamp, e.Counter, e.Version,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				return ErrVersionConflict
			}
			return fmt.Errorf("inserting event: %w", err)
		}
	}
	return nil
}

// InsertOutbox writes outbox rows into an existing transaction.
func (s *Store) InsertOutbox(ctx context.Context, tx pgx.Tx, events []Event) error {
	for _, e := range events {
		_, err := tx.Exec(ctx, insertOutboxSQL,
			e.ID, e.AggregateID, e.AggregateType, e.EventType, e.UserID, e.Data,
		)
		if err != nil {
			return fmt.Errorf("inserting outbox entry: %w", err)
		}
	}
	return nil
}

// Append writes events atomically in a self-managed transaction.
// Backward-compatible wrapper around AppendTx.
func (s *Store) Append(ctx context.Context, events []Event) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			s.logger.Error().Err(rbErr).Msg("eventstore: rollback failed")
		}
	}()

	if err := s.AppendTx(ctx, tx, events); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ClaimOutbox selects unpublished outbox entries with row-level locking.
// Concurrent pollers skip locked rows via FOR UPDATE SKIP LOCKED.
func (s *Store) ClaimOutbox(ctx context.Context, tx pgx.Tx, batchSize int) ([]OutboxEntry, error) {
	rows, err := tx.Query(ctx, claimOutboxSQL, batchSize)
	if err != nil {
		return nil, fmt.Errorf("claiming outbox entries: %w", err)
	}
	defer rows.Close()

	var entries []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		if err := rows.Scan(&e.ID, &e.EventID, &e.AggregateID, &e.AggregateType,
			&e.EventType, &e.UserID, &e.Data, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning outbox entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// MarkPublished updates outbox entries as published within the given transaction.
func (s *Store) MarkPublished(ctx context.Context, tx pgx.Tx, ids []int64) error {
	_, err := tx.Exec(ctx, markPublishedSQL, ids)
	return err
}

// LoadByAggregate returns all events for the given aggregate, ordered by version.
func (s *Store) LoadByAggregate(ctx context.Context, aggregateID uuid.UUID) ([]Event, error) {
	rows, err := s.pool.Query(ctx, loadByAggregateSQL, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("querying events by aggregate: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// LoadByAggregateFromVersion returns events starting from the specified version.
func (s *Store) LoadByAggregateFromVersion(ctx context.Context, aggregateID uuid.UUID, fromVersion int) ([]Event, error) {
	rows, err := s.pool.Query(ctx, loadByAggregateFromVersionSQL, aggregateID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("querying events by aggregate from version: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// LoadByUserSince returns all events for a user since the given HLC timestamp.
func (s *Store) LoadByUserSince(ctx context.Context, userID uuid.UUID, since time.Time, sinceCounter int) ([]Event, error) {
	rows, err := s.pool.Query(ctx, loadByUserSinceSQL, userID, since, sinceCounter)
	if err != nil {
		return nil, fmt.Errorf("querying events by user since: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func scanEvents(rows pgx.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(
			&e.ID, &e.AggregateID, &e.AggregateType, &e.EventType,
			&e.UserID, &e.Data, &e.Timestamp, &e.Counter, &e.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning event row: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating event rows: %w", err)
	}
	return events, nil
}
