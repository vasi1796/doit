package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// SnapshotHandler serves aggregate snapshots for client rehydration.
type SnapshotHandler struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewSnapshotHandler(pool *pgxpool.Pool, logger zerolog.Logger) *SnapshotHandler {
	return &SnapshotHandler{pool: pool, logger: logger}
}

type snapshotEntry struct {
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Version       int             `json:"version"`
	Data          json.RawMessage `json:"data"`
}

// List handles GET /api/v1/snapshots — returns all snapshots for the user.
func (h *SnapshotHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT aggregate_id, aggregate_type, version, data
		 FROM aggregate_snapshots WHERE user_id = $1`, userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("querying snapshots")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	snapshots := make([]snapshotEntry, 0)
	for rows.Next() {
		var s snapshotEntry
		if err := rows.Scan(&s.AggregateID, &s.AggregateType, &s.Version, &s.Data); err != nil {
			h.logger.Error().Err(err).Msg("scanning snapshot row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}
		snapshots = append(snapshots, s)
	}

	writeJSON(w, h.logger, http.StatusOK, snapshots)
}
