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

// ListCommander is the interface the list handler needs from the domain.
type ListCommander interface {
	CreateList(ctx context.Context, cmd domain.CreateList) error
}

type ListHandler struct {
	cmds   ListCommander
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewListHandler(cmds ListCommander, pool *pgxpool.Pool, logger zerolog.Logger) *ListHandler {
	return &ListHandler{cmds: cmds, pool: pool, logger: logger}
}

type createListRequest struct {
	Name     string `json:"name"`
	Colour   string `json:"colour"`
	Icon     string `json:"icon,omitempty"`
	Position string `json:"position"`
}

type listResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Colour    string    `json:"colour,omitempty"`
	Icon      string    `json:"icon,omitempty"`
	Position  string    `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Create handles POST /api/v1/lists
func (h *ListHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req createListRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	listID := uuid.New()
	cmd := domain.CreateList{
		ListID:   listID,
		UserID:   userID,
		Name:     req.Name,
		Colour:   req.Colour,
		Icon:     req.Icon,
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

	lists := make([]listResponse, 0)
	for rows.Next() {
		var l listResponse
		if err := rows.Scan(&l.ID, &l.Name, &l.Colour, &l.Icon, &l.Position, &l.CreatedAt, &l.UpdatedAt); err != nil {
			h.logger.Error().Err(err).Msg("scanning list row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
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

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("beginning transaction for list delete")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer func() {
		if err := tx.Rollback(r.Context()); err != nil && err.Error() != "tx is closed" {
			h.logger.Error().Err(err).Msg("rolling back list delete transaction")
		}
	}()

	// Move tasks belonging to this list to Inbox (list_id = NULL)
	if _, err := tx.Exec(r.Context(),
		`UPDATE tasks SET list_id = NULL WHERE list_id = $1 AND user_id = $2`,
		listID, userID,
	); err != nil {
		h.logger.Error().Err(err).Msg("moving tasks to inbox on list delete")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	// Delete the list
	tag, err := tx.Exec(r.Context(),
		`DELETE FROM lists WHERE id = $1 AND user_id = $2`,
		listID, userID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("deleting list")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	if tag.RowsAffected() == 0 {
		writeError(w, h.logger, http.StatusNotFound, "list not found")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		h.logger.Error().Err(err).Msg("committing list delete transaction")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
