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

// https://www.postgresql.org/docs/current/errcodes-appendix.html
const pgUniqueViolation = "23505"

const insertSQL = `INSERT INTO events (id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

const loadByAggregateSQL = `SELECT id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, version
FROM events WHERE aggregate_id = $1 ORDER BY version ASC`

const loadByAggregateFromVersionSQL = `SELECT id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, version
FROM events WHERE aggregate_id = $1 AND version >= $2 ORDER BY version ASC`

const loadByUserSinceSQL = `SELECT id, aggregate_id, aggregate_type, event_type, user_id, data, timestamp, version
FROM events WHERE user_id = $1 AND timestamp > $2 ORDER BY timestamp ASC, version ASC`

// Store provides append and query operations on the event store.
type Store struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// New creates a new Store backed by the given connection pool.
func New(pool *pgxpool.Pool, logger zerolog.Logger) *Store {
	return &Store{pool: pool, logger: logger}
}

// Append writes one or more events atomically. All events must belong
// to the same aggregate. Returns ErrVersionConflict if the unique
// (aggregate_id, version) constraint is violated.
func (s *Store) Append(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return ErrNoEvents
	}

	aggregateID := events[0].AggregateID
	for _, e := range events[1:] {
		if e.AggregateID != aggregateID {
			return fmt.Errorf("eventstore: all events must belong to the same aggregate")
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			s.logger.Error().Err(rbErr).Msg("eventstore: rollback failed")
		}
	}()

	for _, e := range events {
		_, err := tx.Exec(ctx, insertSQL,
			e.ID, e.AggregateID, e.AggregateType, e.EventType,
			e.UserID, e.Data, e.Timestamp, e.Version,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				return ErrVersionConflict
			}
			return fmt.Errorf("inserting event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// LoadByAggregate returns all events for the given aggregate, ordered
// by version ascending.
func (s *Store) LoadByAggregate(ctx context.Context, aggregateID uuid.UUID) ([]Event, error) {
	rows, err := s.pool.Query(ctx, loadByAggregateSQL, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("querying events by aggregate: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

// LoadByAggregateFromVersion returns events for the given aggregate
// starting from the specified version (inclusive), ordered by version.
func (s *Store) LoadByAggregateFromVersion(ctx context.Context, aggregateID uuid.UUID, fromVersion int) ([]Event, error) {
	rows, err := s.pool.Query(ctx, loadByAggregateFromVersionSQL, aggregateID, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("querying events by aggregate from version: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

// LoadByUserSince returns all events for a user since the given
// timestamp, ordered by timestamp then version. Used for sync.
func (s *Store) LoadByUserSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]Event, error) {
	rows, err := s.pool.Query(ctx, loadByUserSinceSQL, userID, since)
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
			&e.UserID, &e.Data, &e.Timestamp, &e.Version,
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
