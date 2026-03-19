package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/auth"
	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
)

func writeJSON(w http.ResponseWriter, logger zerolog.Logger, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error().Err(err).Msg("failed to encode JSON response")
	}
}

func writeError(w http.ResponseWriter, logger zerolog.Logger, status int, msg string) {
	writeJSON(w, logger, status, map[string]string{"error": msg})
}

// readJSON decodes a JSON request body into dst.
// Returns false and writes a 400 response if decoding fails.
// Limits request body to 1MB to prevent abuse.
func readJSON(w http.ResponseWriter, logger zerolog.Logger, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, logger, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// parseUUID extracts a chi URL parameter and parses it as a UUID.
// Returns uuid.Nil and writes a 400 response if invalid.
func parseUUID(w http.ResponseWriter, logger zerolog.Logger, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, logger, http.StatusBadRequest, "invalid "+param)
		return uuid.Nil, false
	}
	return id, true
}

// requireUserID extracts the authenticated user ID from the request context.
// Returns uuid.Nil and writes a 401 response if not present.
func requireUserID(w http.ResponseWriter, logger zerolog.Logger, r *http.Request) (uuid.UUID, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, logger, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	return userID, true
}

// mapDomainError maps domain and eventstore errors to HTTP status codes.
// Returns true if an error was handled, false if err is nil.
func mapDomainError(w http.ResponseWriter, logger zerolog.Logger, err error) bool {
	if err == nil {
		return false
	}

	switch {
	// 404 Not Found
	case errors.Is(err, domain.ErrTaskNotFound),
		errors.Is(err, domain.ErrListNotFound),
		errors.Is(err, domain.ErrLabelNotFound),
		errors.Is(err, domain.ErrSubtaskNotFound):
		writeError(w, logger, http.StatusNotFound, err.Error())

	// 400 Bad Request
	case errors.Is(err, domain.ErrEmptyTitle),
		errors.Is(err, domain.ErrInvalidPriority):
		writeError(w, logger, http.StatusBadRequest, err.Error())

	// 409 Conflict
	case errors.Is(err, domain.ErrTaskAlreadyCompleted),
		errors.Is(err, domain.ErrTaskAlreadyDeleted),
		errors.Is(err, domain.ErrTaskNotCompleted),
		errors.Is(err, domain.ErrTaskAlreadyCreated),
		errors.Is(err, domain.ErrListAlreadyCreated),
		errors.Is(err, domain.ErrLabelAlreadyCreated),
		errors.Is(err, domain.ErrLabelAlreadyAttached),
		errors.Is(err, domain.ErrLabelNotAttached),
		errors.Is(err, domain.ErrSubtaskAlreadyCompleted),
		errors.Is(err, eventstore.ErrVersionConflict):
		writeError(w, logger, http.StatusConflict, err.Error())

	// 500 Internal Server Error
	default:
		logger.Error().Err(err).Msg("internal error")
		writeError(w, logger, http.StatusInternalServerError, "internal error")
	}

	return true
}
