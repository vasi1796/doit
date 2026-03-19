package handler

import (
	"context"
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
}

type LabelHandler struct {
	cmds   LabelCommander
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewLabelHandler(cmds LabelCommander, pool *pgxpool.Pool, logger zerolog.Logger) *LabelHandler {
	return &LabelHandler{cmds: cmds, pool: pool, logger: logger}
}

type createLabelRequest struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`
}

type labelResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Colour    string     `json:"colour,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Create handles POST /api/v1/labels
func (h *LabelHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req createLabelRequest
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

	labels := make([]labelResponse, 0)
	for rows.Next() {
		var l labelResponse
		if err := rows.Scan(&l.ID, &l.Name, &l.Colour, &l.CreatedAt); err != nil {
			h.logger.Error().Err(err).Msg("scanning label row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
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
