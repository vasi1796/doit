package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/domain"
)

// LabelCommander is the interface the label handler needs from the domain.
type LabelCommander interface {
	CreateLabel(ctx context.Context, cmd domain.CreateLabel) error
	DeleteLabel(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.DeleteLabel) error
}

type LabelHandler struct {
	cmds   LabelCommander
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewLabelHandler(cmds LabelCommander, pool *pgxpool.Pool, logger zerolog.Logger) *LabelHandler {
	return &LabelHandler{cmds: cmds, pool: pool, logger: logger}
}

// Request and response types are generated from api/openapi.yaml
// in openapi_types.gen.go. Do not define them here.

// Create handles POST /api/v1/labels
func (h *LabelHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req CreateLabelRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	labelID := uuid.New()
	cmd := domain.CreateLabel{
		LabelID: labelID,
		UserID:  userID,
		Name:    req.Name,
		Colour:  req.Colour,
	}

	if mapDomainError(w, h.logger, h.cmds.CreateLabel(r.Context(), cmd)) {
		return
	}

	writeJSON(w, h.logger, http.StatusCreated, map[string]string{"id": labelID.String()})
}

// List handles GET /api/v1/labels
func (h *LabelHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, colour, created_at FROM labels WHERE user_id = $1 ORDER BY name ASC`,
		userID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("querying labels")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	labels := make([]Label, 0)
	for rows.Next() {
		var l Label
		var colour sql.NullString
		var createdAt sql.NullTime
		if err := rows.Scan(&l.Id, &l.Name, &colour, &createdAt); err != nil {
			h.logger.Error().Err(err).Msg("scanning label row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}
		if colour.Valid {
			l.Colour = &colour.String
		}
		if createdAt.Valid {
			l.CreatedAt = &createdAt.Time
		}
		labels = append(labels, l)
	}
	if err := rows.Err(); err != nil {
		h.logger.Error().Err(err).Msg("iterating label rows")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, h.logger, http.StatusOK, labels)
}

// Delete handles DELETE /api/v1/labels/{id}
func (h *LabelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	labelID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	err := h.cmds.DeleteLabel(r.Context(), labelID, userID, domain.DeleteLabel{
		DeletedAt: time.Now().UTC(),
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
