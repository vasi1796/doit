package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ICalHandler manages iCal calendar feed endpoints.
type ICalHandler struct {
	pool    *pgxpool.Pool
	logger  zerolog.Logger
	baseURL string
}

func NewICalHandler(pool *pgxpool.Pool, logger zerolog.Logger, baseURL string) *ICalHandler {
	return &ICalHandler{pool: pool, logger: logger, baseURL: baseURL}
}

// ServeCalendar handles GET /ical/{token}/calendar.ics (unauthenticated, token-based).
func (h *ICalHandler) ServeCalendar(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, h.logger, http.StatusBadRequest, "missing token")
		return
	}

	// Look up user by token
	var userID uuid.UUID
	err := h.pool.QueryRow(r.Context(),
		`SELECT user_id FROM ical_tokens WHERE token = $1`, token,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, h.logger, http.StatusNotFound, "invalid token")
			return
		}
		h.logger.Error().Err(err).Msg("ical: failed to look up token")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	// Query tasks with due dates that are not deleted and not completed
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, title, description, due_date, due_time, recurrence_rule, updated_at
		 FROM tasks
		 WHERE user_id = $1 AND is_deleted = false AND is_completed = false AND due_date IS NOT NULL`,
		userID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("ical: failed to query tasks")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	type icalTask struct {
		ID             uuid.UUID
		Title          string
		Description    sql.NullString
		DueDate        string
		DueTime        sql.NullString
		RecurrenceRule sql.NullString
		UpdatedAt      time.Time
	}

	var tasks []icalTask
	for rows.Next() {
		var t icalTask
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.DueDate, &t.DueTime, &t.RecurrenceRule, &t.UpdatedAt); err != nil {
			h.logger.Error().Err(err).Msg("ical: failed to scan task row")
			writeError(w, h.logger, http.StatusInternalServerError, "internal error")
			return
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		h.logger.Error().Err(err).Msg("ical: error iterating task rows")
		writeError(w, h.logger, http.StatusInternalServerError, "internal error")
		return
	}

	// Generate iCal output
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"doit.ics\"")
	w.WriteHeader(http.StatusOK)

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//DoIt//DoIt Task Manager//EN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	b.WriteString("X-WR-CALNAME:DoIt Tasks\r\n")

	for _, t := range tasks {
		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString(fmt.Sprintf("UID:%s@doit\r\n", t.ID.String()))
		b.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(t.Title)))

		if t.Description.Valid && t.Description.String != "" {
			b.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(t.Description.String)))
		}

		// DTSTART: date-only or date+time
		if t.DueTime.Valid && t.DueTime.String != "" {
			// Parse date and time
			dtStart, err := parseDateAndTime(t.DueDate, t.DueTime.String)
			if err == nil {
				b.WriteString(fmt.Sprintf("DTSTART:%s\r\n", dtStart.UTC().Format("20060102T150405Z")))
			} else {
				// Fallback to date-only
				b.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", formatICalDate(t.DueDate)))
			}
		} else {
			b.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", formatICalDate(t.DueDate)))
		}

		b.WriteString(fmt.Sprintf("LAST-MODIFIED:%s\r\n", t.UpdatedAt.UTC().Format("20060102T150405Z")))

		// RRULE if recurrence is set
		if t.RecurrenceRule.Valid && t.RecurrenceRule.String != "" {
			rrule := mapRecurrenceToRRule(t.RecurrenceRule.String)
			if rrule != "" {
				b.WriteString(fmt.Sprintf("RRULE:%s\r\n", rrule))
			}
		}

		b.WriteString("END:VEVENT\r\n")
	}

	b.WriteString("END:VCALENDAR\r\n")

	if _, err := w.Write([]byte(b.String())); err != nil {
		h.logger.Error().Err(err).Msg("ical: failed to write response")
	}
}

// GenerateToken handles POST /api/v1/ical/token — generates a new iCal feed token.
func (h *ICalHandler) GenerateToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	// Generate 32 random bytes, hex-encoded (64 chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		h.logger.Error().Err(err).Msg("ical: failed to generate random token")
		writeError(w, h.logger, http.StatusInternalServerError, "failed to generate token")
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Upsert token
	_, err := h.pool.Exec(r.Context(),
		`INSERT INTO ical_tokens (user_id, token)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET token = EXCLUDED.token, created_at = NOW()`,
		userID, token,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("ical: failed to upsert token")
		writeError(w, h.logger, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Build the feed URL
	base := h.resolveBaseURL(r)
	feedURL := fmt.Sprintf("%s/ical/%s/calendar.ics", base, token)

	h.logger.Info().Str("user_id", userID.String()).Msg("ical token generated")
	writeJSON(w, h.logger, http.StatusOK, map[string]string{"url": feedURL})
}

// RevokeToken handles DELETE /api/v1/ical/token — revokes the user's iCal feed token.
func (h *ICalHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	_, err := h.pool.Exec(r.Context(),
		`DELETE FROM ical_tokens WHERE user_id = $1`, userID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("ical: failed to delete token")
		writeError(w, h.logger, http.StatusInternalServerError, "failed to revoke token")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTokenStatus handles GET /api/v1/ical/token — checks if a token exists.
func (h *ICalHandler) GetTokenStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var token string
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(),
		`SELECT token, created_at FROM ical_tokens WHERE user_id = $1`, userID,
	).Scan(&token, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, h.logger, http.StatusOK, map[string]any{"enabled": false})
			return
		}
		h.logger.Error().Err(err).Msg("ical: failed to check token status")
		writeError(w, h.logger, http.StatusInternalServerError, "failed to check token status")
		return
	}

	base := h.resolveBaseURL(r)
	feedURL := fmt.Sprintf("%s/ical/%s/calendar.ics", base, token)

	writeJSON(w, h.logger, http.StatusOK, map[string]any{
		"enabled":    true,
		"url":        feedURL,
		"created_at": createdAt.Format(time.RFC3339),
	})
}

// resolveBaseURL returns the base URL for building feed URLs.
func (h *ICalHandler) resolveBaseURL(r *http.Request) string {
	if h.baseURL != "" {
		return strings.TrimRight(h.baseURL, "/")
	}
	scheme := "https"
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// escapeICalText escapes special characters in iCal text fields.
func escapeICalText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// formatICalDate converts a "2006-01-02" date string to iCal DATE format "20060102".
func formatICalDate(dateStr string) string {
	return strings.ReplaceAll(dateStr, "-", "")
}

// parseDateAndTime parses a date string "2006-01-02" and time string "HH:MM" into a time.Time.
func parseDateAndTime(dateStr, timeStr string) (time.Time, error) {
	combined := dateStr + "T" + timeStr
	t, err := time.Parse("2006-01-02T15:04", combined)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing date+time: %w", err)
	}
	return t, nil
}

// mapRecurrenceToRRule maps DoIt recurrence rules to iCal RRULE values.
func mapRecurrenceToRRule(rule string) string {
	switch strings.ToLower(rule) {
	case "daily":
		return "FREQ=DAILY"
	case "weekly":
		return "FREQ=WEEKLY"
	case "monthly":
		return "FREQ=MONTHLY"
	case "yearly":
		return "FREQ=YEARLY"
	default:
		return ""
	}
}
