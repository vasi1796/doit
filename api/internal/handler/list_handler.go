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

// ListCommander is the interface the list handler needs from the domain.
type ListCommander interface {
	CreateList(ctx context.Context, cmd domain.CreateList) error
	DeleteList(ctx context.Context, aggregateID uuid.UUID, userID uuid.UUID, cmd domain.DeleteList) error
}

type ListHandler struct {
	cmds   ListCommander
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewListHandler(cmds ListCommander, pool *pgxpool.Pool, logger zerolog.Logger) *ListHandler {
	return &ListHandler{cmds: cmds, pool: pool, logger: logger}
}

// Request and response types are generated from api/openapi.yaml
// in openapi_types.gen.go. Do not define them here.

// Create handles POST /api/v1/lists
func (h *ListHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req CreateListRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	var icon string
	if req.Icon != nil {
		icon = *req.Icon
	}

	listID := uuid.New()
	cmd := domain.CreateList{
		ListID:   listID,
		UserID:   userID,
		Name:     req.Name,
		Colour:   req.Colour,
		Icon:     icon,
		Position: req.Position,
	}

	if mapDomainError(w, h.logger, h.cmds.CreateList(r.Context(), cmd)) {
		return
	}

	writeJSON(w, h.logger, http.StatusCreated, map[string]string{"id": listID.String()})
}

// List handles GET /api/v1/lists
func (h *ListHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, name, colour, icon, position, created_at, updated_at
		 FROM lists WHERE user_id = $1 ORDER BY position ASC`,
		userID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("querying lists")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	lists := make([]List, 0)
	for rows.Next() {
		var l List
		var colour, icon sql.NullString
		if err := rows.Scan(&l.Id, &l.Name, &colour, &icon, &l.Position, &l.CreatedAt, &l.UpdatedAt); err != nil {
			h.logger.Error().Err(err).Msg("scanning list row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}
		if colour.Valid {
			l.Colour = &colour.String
		}
		if icon.Valid {
			l.Icon = &icon.String
		}
		lists = append(lists, l)
	}
	if err := rows.Err(); err != nil {
		h.logger.Error().Err(err).Msg("iterating list rows")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, h.logger, http.StatusOK, lists)
}

// Delete handles DELETE /api/v1/lists/{id}
func (h *ListHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	listID, ok := parseUUID(w, h.logger, r, "id")
	if !ok {
		return
	}

	err := h.cmds.DeleteList(r.Context(), listID, userID, domain.DeleteList{
		DeletedAt: time.Now().UTC(),
	})
	if mapDomainError(w, h.logger, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
