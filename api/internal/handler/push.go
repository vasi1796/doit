package handler

import (
	"encoding/json"
	"io"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PushHandler manages Web Push subscription endpoints.
type PushHandler struct {
	pool      *pgxpool.Pool
	vapidOpts *webpush.Options
	vapidPub  string
	logger    zerolog.Logger
}

func NewPushHandler(pool *pgxpool.Pool, vapidPublicKey, vapidPrivateKey, vapidSubject string, logger zerolog.Logger) *PushHandler {
	var opts *webpush.Options
	if vapidPublicKey != "" && vapidPrivateKey != "" {
		opts = &webpush.Options{
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			Subscriber:      vapidSubject,
			TTL:             3600,
		}
	}
	return &PushHandler{pool: pool, vapidOpts: opts, vapidPub: vapidPublicKey, logger: logger}
}

// GetVAPIDKey returns the VAPID public key for PushManager.subscribe().
func (h *PushHandler) GetVAPIDKey(w http.ResponseWriter, r *http.Request) {
	if h.vapidPub == "" {
		writeError(w, h.logger, http.StatusServiceUnavailable, "push notifications not configured")
		return
	}
	writeJSON(w, h.logger, http.StatusOK, map[string]string{"vapid_public_key": h.vapidPub})
}

type subscribeRequest struct {
	Endpoint string        `json:"endpoint"`
	Keys     subscribeKeys `json:"keys"`
}

type subscribeKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// Subscribe stores a push subscription for the authenticated user.
func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req subscribeRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		writeError(w, h.logger, http.StatusBadRequest, "endpoint, keys.p256dh, and keys.auth are required")
		return
	}

	_, err := h.pool.Exec(r.Context(),
		`INSERT INTO push_subscriptions (user_id, endpoint, key_p256dh, key_auth)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (endpoint) DO UPDATE SET
		   user_id = EXCLUDED.user_id,
		   key_p256dh = EXCLUDED.key_p256dh,
		   key_auth = EXCLUDED.key_auth,
		   created_at = NOW()`,
		userID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("push: failed to store subscription")
		writeError(w, h.logger, http.StatusInternalServerError, "failed to store subscription")
		return
	}

	h.logger.Info().Str("user_id", userID.String()).Msg("push subscription stored")
	w.WriteHeader(http.StatusCreated)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

// Unsubscribe removes a push subscription for the authenticated user.
func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	var req unsubscribeRequest
	if !readJSON(w, h.logger, r, &req) {
		return
	}

	if req.Endpoint == "" {
		writeError(w, h.logger, http.StatusBadRequest, "endpoint is required")
		return
	}

	_, err := h.pool.Exec(r.Context(),
		`DELETE FROM push_subscriptions WHERE endpoint = $1 AND user_id = $2`,
		req.Endpoint, userID,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("push: failed to delete subscription")
		writeError(w, h.logger, http.StatusInternalServerError, "failed to delete subscription")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Test sends a test push notification to all subscriptions for the authenticated user.
func (h *PushHandler) Test(w http.ResponseWriter, r *http.Request) {
	if h.vapidOpts == nil {
		writeError(w, h.logger, http.StatusServiceUnavailable, "push notifications not configured")
		return
	}

	userID, ok := requireUserID(w, h.logger, r)
	if !ok {
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, endpoint, key_p256dh, key_auth FROM push_subscriptions WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		writeError(w, h.logger, http.StatusInternalServerError, "failed to load subscriptions")
		return
	}
	defer rows.Close()

	payload, _ := json.Marshal(map[string]string{
		"title": "DoIt: Test Notification",
		"body":  "Push notifications are working!",
		"url":   "/today",
	})

	var sent, failed int
	for rows.Next() {
		var sub struct {
			ID       int64
			Endpoint string
			P256dh   string
			Auth     string
		}
		if err := rows.Scan(&sub.ID, &sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			failed++
			continue
		}

		resp, err := webpush.SendNotification(payload, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, h.vapidOpts)
		if err != nil {
			h.logger.Error().Err(err).Str("endpoint", sub.Endpoint).Msg("test push failed")
			failed++
			continue
		}
		if resp.StatusCode == http.StatusGone {
			resp.Body.Close()
			if _, err := h.pool.Exec(r.Context(), `DELETE FROM push_subscriptions WHERE id = $1`, sub.ID); err != nil {
				h.logger.Error().Err(err).Int64("sub_id", sub.ID).Msg("failed to delete stale subscription")
			}
			failed++
		} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			sent++
		} else {
			body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for diagnostic logging
			resp.Body.Close()
			h.logger.Error().Int("status", resp.StatusCode).Str("body", string(body)).Str("endpoint", sub.Endpoint).Msg("push service rejected notification")
			failed++
		}
	}

	writeJSON(w, h.logger, http.StatusOK, map[string]int{"sent": sent, "failed": failed})
}
