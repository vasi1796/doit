// Reminder worker — sends push notifications for tasks due today.
// Runs on a timer (default: every hour), checks if it's the configured
// reminder hour, and sends one aggregated notification per user.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/vasi1796/doit/internal/config"
)

type pushSubscription struct {
	ID       int64
	Endpoint string
	P256dh   string
	Auth     string
}

type dueTaskSummary struct {
	UserID    uuid.UUID
	TaskCount int
	Titles    []string // up to 3
}

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", "worker-reminder").Logger()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	if cfg.VAPIDPublicKey == "" || cfg.VAPIDPrivateKey == "" {
		logger.Fatal().Msg("VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY are required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	interval := envDuration("REMINDER_INTERVAL", 1*time.Hour)
	reminderHour := envInt("REMINDER_HOUR", 8)
	tz := envString("REMINDER_TZ", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		logger.Warn().Str("tz", tz).Msg("invalid timezone, falling back to UTC")
		loc = time.UTC
	}

	vapidOpts := &webpush.Options{
		VAPIDPublicKey:  cfg.VAPIDPublicKey,
		VAPIDPrivateKey: cfg.VAPIDPrivateKey,
		Subscriber:      cfg.VAPIDSubject,
		TTL:             3600,
	}

	logger.Info().
		Dur("interval", interval).
		Int("reminder_hour", reminderHour).
		Str("tz", tz).
		Msg("reminder worker started")

	// Run immediately on startup, then on interval
	sendReminders(ctx, pool, vapidOpts, reminderHour, loc, logger)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("reminder worker shutting down")
			return
		case <-ticker.C:
			sendReminders(ctx, pool, vapidOpts, reminderHour, loc, logger)
		}
	}
}

func sendReminders(ctx context.Context, pool *pgxpool.Pool, opts *webpush.Options, reminderHour int, loc *time.Location, logger zerolog.Logger) {
	now := time.Now().In(loc)
	if now.Hour() != reminderHour {
		logger.Debug().Int("current_hour", now.Hour()).Int("reminder_hour", reminderHour).Msg("not reminder hour, skipping")
		return
	}

	today := now.Format("2006-01-02")

	// Find users with tasks due today
	rows, err := pool.Query(ctx,
		`SELECT t.user_id, COUNT(*) as task_count,
		        array_agg(t.title ORDER BY t.position ASC) FILTER (WHERE true) as titles
		 FROM tasks t
		 WHERE t.due_date = $1
		   AND NOT t.is_completed
		   AND NOT t.is_deleted
		 GROUP BY t.user_id`,
		today,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to query due tasks")
		return
	}
	defer rows.Close()

	var summaries []dueTaskSummary
	for rows.Next() {
		var s dueTaskSummary
		var titles []string
		if err := rows.Scan(&s.UserID, &s.TaskCount, &titles); err != nil {
			logger.Error().Err(err).Msg("failed to scan due task row")
			continue
		}
		if len(titles) > 3 {
			s.Titles = titles[:3]
		} else {
			s.Titles = titles
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("error iterating due task rows")
		return
	}

	if len(summaries) == 0 {
		logger.Debug().Msg("no tasks due today")
		return
	}

	for _, s := range summaries {
		// Idempotency: check if already sent today
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM reminder_log WHERE user_id = $1 AND sent_date = $2)`,
			s.UserID, today,
		).Scan(&exists)
		if err != nil {
			logger.Error().Err(err).Str("user_id", s.UserID.String()).Msg("failed to check reminder log")
			continue
		}
		if exists {
			logger.Debug().Str("user_id", s.UserID.String()).Msg("reminder already sent today")
			continue
		}

		// Build notification payload
		body := strings.Join(s.Titles, ", ")
		if s.TaskCount > 3 {
			body += fmt.Sprintf(" + %d more", s.TaskCount-3)
		}
		payload := map[string]string{
			"title": fmt.Sprintf("DoIt: %d task%s due today", s.TaskCount, pluralS(s.TaskCount)),
			"body":  body,
			"url":   "/today",
		}
		payloadJSON, _ := json.Marshal(payload)

		// Load subscriptions for this user
		subRows, err := pool.Query(ctx,
			`SELECT id, endpoint, key_p256dh, key_auth FROM push_subscriptions WHERE user_id = $1`,
			s.UserID,
		)
		if err != nil {
			logger.Error().Err(err).Str("user_id", s.UserID.String()).Msg("failed to load subscriptions")
			continue
		}

		var sent int
		for subRows.Next() {
			var sub pushSubscription
			if err := subRows.Scan(&sub.ID, &sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
				logger.Error().Err(err).Msg("failed to scan subscription")
				continue
			}

			resp, err := webpush.SendNotification(payloadJSON, &webpush.Subscription{
				Endpoint: sub.Endpoint,
				Keys: webpush.Keys{
					P256dh: sub.P256dh,
					Auth:   sub.Auth,
				},
			}, opts)
			if err != nil {
				logger.Error().Err(err).Str("endpoint", sub.Endpoint).Msg("failed to send push")
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusGone {
				// Subscription expired — clean up
				if _, err := pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE id = $1`, sub.ID); err != nil {
					logger.Error().Err(err).Int64("sub_id", sub.ID).Msg("failed to delete stale subscription")
				} else {
					logger.Info().Str("endpoint", sub.Endpoint).Msg("deleted stale push subscription")
				}
				continue
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				sent++
			} else {
				logger.Warn().Int("status", resp.StatusCode).Str("endpoint", sub.Endpoint).Msg("push service returned error")
			}
		}
		subRows.Close()

		// Log that we sent reminders for this user today
		if _, err := pool.Exec(ctx,
			`INSERT INTO reminder_log (user_id, sent_date, task_count) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			s.UserID, today, s.TaskCount,
		); err != nil {
			logger.Error().Err(err).Str("user_id", s.UserID.String()).Msg("failed to log reminder")
		}

		logger.Info().
			Str("user_id", s.UserID.String()).
			Int("task_count", s.TaskCount).
			Int("notifications_sent", sent).
			Msg("reminders sent")
	}
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
