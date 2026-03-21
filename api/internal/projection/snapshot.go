package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const upsertSnapshotSQL = `
INSERT INTO aggregate_snapshots (aggregate_id, aggregate_type, user_id, version, data, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (aggregate_id, aggregate_type)
DO UPDATE SET version = $4, data = $5, updated_at = $6`

// SnapshotWriter saves aggregate state snapshots for client rehydration.
type SnapshotWriter struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewSnapshotWriter(pool *pgxpool.Pool, logger zerolog.Logger) *SnapshotWriter {
	return &SnapshotWriter{pool: pool, logger: logger}
}

// SaveTaskSnapshot reads the current task from the read model and upserts a snapshot.
func (s *SnapshotWriter) SaveTaskSnapshot(ctx context.Context, taskID, userID uuid.UUID) error {
	jsonRow := s.pool.QueryRow(ctx,
		`SELECT row_to_json(t) FROM (
			SELECT id, list_id, title, description, priority, due_date, due_time, position,
				is_completed, completed_at, is_deleted, created_at, updated_at, recurrence_rule
			FROM tasks WHERE id = $1 AND user_id = $2
		) t`, taskID, userID)

	var jsonData json.RawMessage
	if err := jsonRow.Scan(&jsonData); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // Task may have been deleted
		}
		return fmt.Errorf("snapshot task json: %w", err)
	}

	_, err := s.pool.Exec(ctx, upsertSnapshotSQL, taskID, "task", userID, 0, jsonData, time.Now().UTC())
	return err
}

// SaveListSnapshot reads the current list and upserts a snapshot.
func (s *SnapshotWriter) SaveListSnapshot(ctx context.Context, listID, userID uuid.UUID) error {
	jsonRow := s.pool.QueryRow(ctx,
		`SELECT row_to_json(t) FROM (
			SELECT id, name, colour, icon, position, created_at, updated_at
			FROM lists WHERE id = $1 AND user_id = $2
		) t`, listID, userID)

	var jsonData json.RawMessage
	if err := jsonRow.Scan(&jsonData); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // List may have been deleted
		}
		return fmt.Errorf("snapshot list json: %w", err)
	}

	_, err := s.pool.Exec(ctx, upsertSnapshotSQL, listID, "list", userID, 0, jsonData, time.Now().UTC())
	return err
}

// SaveLabelSnapshot reads the current label and upserts a snapshot.
func (s *SnapshotWriter) SaveLabelSnapshot(ctx context.Context, labelID, userID uuid.UUID) error {
	jsonRow := s.pool.QueryRow(ctx,
		`SELECT row_to_json(t) FROM (
			SELECT id, name, colour, created_at
			FROM labels WHERE id = $1 AND user_id = $2
		) t`, labelID, userID)

	var jsonData json.RawMessage
	if err := jsonRow.Scan(&jsonData); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // Label may have been deleted
		}
		return fmt.Errorf("snapshot label json: %w", err)
	}

	_, err := s.pool.Exec(ctx, upsertSnapshotSQL, labelID, "label", userID, 0, jsonData, time.Now().UTC())
	return err
}
