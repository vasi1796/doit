// Reminder worker — sends push notifications for due tasks.
//
// Two notification strategies run every tick (default: 10m):
//  1. Morning digest — at REMINDER_HOUR, one aggregated notification per user
//     for tasks with a due_date but no due_time.
//  2. Due-time alerts — individual notifications for tasks whose due_time
//     falls within the current polling window.
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

type dueTimeTask struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Title   string
	DueDate time.Time
}

// SQL queries — extracted for readability.
const (
	queryDigestTasks = `
		SELECT t.user_id, COUNT(*) as task_count,
		       array_agg(t.title ORDER BY t.position ASC) FILTER (WHERE true) as titles
		FROM tasks t
		WHERE t.due_date = $1
		  AND t.due_time IS NULL
		  AND NOT t.is_completed
		  AND NOT t.is_deleted
		GROUP BY t.user_id`

	queryDigestSent = `
		SELECT EXISTS(SELECT 1 FROM reminder_log WHERE user_id = $1 AND sent_date = $2)`

	execLogDigest = `
		INSERT INTO reminder_log (user_id, sent_date, task_count) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`

	queryDueTimeTasks = `
		SELECT t.id, t.user_id, t.title, t.due_date
		FROM tasks t
		WHERE t.due_date = $1
		  AND t.due_time IS NOT NULL
		  AND t.due_time >= $2::time
		  AND t.due_time < $3::time
		  AND NOT t.is_completed
		  AND NOT t.is_deleted
		  AND NOT EXISTS (
		      SELECT 1 FROM task_reminder_log r
		      WHERE r.task_id = t.id AND r.due_date = $1
		  )`

	queryDueTimeTasksMidnight = `
		SELECT t.id, t.user_id, t.title, t.due_date
		FROM tasks t
		WHERE NOT t.is_completed
		  AND NOT t.is_deleted
		  AND t.due_time IS NOT NULL
		  AND (
		      (t.due_date = $1 AND t.due_time >= $2::time)
		      OR
		      (t.due_date = $3 AND t.due_time < $4::time)
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM task_reminder_log r
		      WHERE r.task_id = t.id AND r.due_date = t.due_date
		  )`

	execLogTaskReminder = `
		INSERT INTO task_reminder_log (task_id, due_date) VALUES ($1, $2) ON CONFLICT DO NOTHING`

	queryUserSubscriptions = `
		SELECT id, endpoint, key_p256dh, key_auth FROM push_subscriptions WHERE user_id = $1`

	execDeleteSubscription = `
		DELETE FROM push_subscriptions WHERE id = $1`
)

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

	interval := envDuration("REMINDER_INTERVAL", 10*time.Minute)
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
	now := time.Now().In(loc)
	tick(ctx, pool, vapidOpts, interval, reminderHour, now, logger)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("reminder worker shutting down")
			return
		case <-ticker.C:
			now := time.Now().In(loc)
			tick(ctx, pool, vapidOpts, interval, reminderHour, now, logger)
		}
	}
}

// tick runs both reminder strategies on each interval.
func tick(ctx context.Context, pool *pgxpool.Pool, opts *webpush.Options, interval time.Duration, reminderHour int, now time.Time, logger zerolog.Logger) {

	// Strategy 1: morning digest for tasks with due_date but no due_time
	if now.Hour() == reminderHour {
		sendMorningDigest(ctx, pool, opts, now, logger)
	}

	// Strategy 2: per-task alerts for tasks with a due_time in the current window
	sendDueTimeAlerts(ctx, pool, opts, now, interval, logger)
}

// sendMorningDigest sends one aggregated notification per user for tasks
// that have a due_date of today but no due_time set.
func sendMorningDigest(ctx context.Context, pool *pgxpool.Pool, opts *webpush.Options, now time.Time, logger zerolog.Logger) {
	today := now.Format("2006-01-02")

	rows, err := pool.Query(ctx, queryDigestTasks, today)
	if err != nil {
		logger.Error().Err(err).Msg("failed to query morning digest tasks")
		return
	}
	defer rows.Close()

	var summaries []dueTaskSummary
	for rows.Next() {
		var s dueTaskSummary
		var titles []string
		if err := rows.Scan(&s.UserID, &s.TaskCount, &titles); err != nil {
			logger.Error().Err(err).Msg("failed to scan morning digest row")
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
		logger.Error().Err(err).Msg("error iterating morning digest rows")
		return
	}

	if len(summaries) == 0 {
		logger.Debug().Msg("no date-only tasks due today")
		return
	}

	for _, s := range summaries {
		// Idempotency: check if already sent today
		var exists bool
		err := pool.QueryRow(ctx, queryDigestSent, s.UserID, today).Scan(&exists)
		if err != nil {
			logger.Error().Err(err).Str("user_id", s.UserID.String()).Msg("failed to check reminder log")
			continue
		}
		if exists {
			logger.Debug().Str("user_id", s.UserID.String()).Msg("morning digest already sent today")
			continue
		}

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

		sent := sendToUser(ctx, pool, opts, s.UserID, payloadJSON, logger)

		if _, err := pool.Exec(ctx, execLogDigest, s.UserID, today, s.TaskCount); err != nil {
			logger.Error().Err(err).Str("user_id", s.UserID.String()).Msg("failed to log morning digest")
		}

		logger.Info().
			Str("user_id", s.UserID.String()).
			Int("task_count", s.TaskCount).
			Int("notifications_sent", sent).
			Msg("morning digest sent")
	}
}

// sendDueTimeAlerts sends individual notifications for tasks whose due_time
// falls within [now-interval, now).
func sendDueTimeAlerts(ctx context.Context, pool *pgxpool.Pool, opts *webpush.Options, now time.Time, interval time.Duration, logger zerolog.Logger) {
	today := now.Format("2006-01-02")
	windowStart := now.Add(-interval)
	windowStartTime := windowStart.Format("15:04:05")
	windowEndTime := now.Format("15:04:05")

	// Handle midnight crossing: if the window spans midnight (e.g. 23:55–00:05),
	// use OR to match both sides. Also check yesterday's date for the pre-midnight part.
	var query string
	var args []any
	if windowStartTime > windowEndTime {
		yesterday := windowStart.Format("2006-01-02")
		query = queryDueTimeTasksMidnight
		args = []any{yesterday, windowStartTime, today, windowEndTime}
	} else {
		query = queryDueTimeTasks
		args = []any{today, windowStartTime, windowEndTime}
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		logger.Error().Err(err).Msg("failed to query due-time tasks")
		return
	}
	defer rows.Close()

	var tasks []dueTimeTask
	for rows.Next() {
		var t dueTimeTask
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.DueDate); err != nil {
			logger.Error().Err(err).Msg("failed to scan due-time task row")
			continue
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Msg("error iterating due-time task rows")
		return
	}

	if len(tasks) == 0 {
		logger.Debug().Msg("no due-time tasks in current window")
		return
	}

	for _, t := range tasks {
		payload := map[string]string{
			"title": "DoIt: Task due now",
			"body":  t.Title,
			"url":   "/today",
		}
		payloadJSON, _ := json.Marshal(payload)

		sent := sendToUser(ctx, pool, opts, t.UserID, payloadJSON, logger)

		if _, err := pool.Exec(ctx, execLogTaskReminder, t.ID, t.DueDate); err != nil {
			logger.Error().Err(err).Str("task_id", t.ID.String()).Msg("failed to log task reminder")
		}

		logger.Info().
			Str("task_id", t.ID.String()).
			Str("user_id", t.UserID.String()).
			Str("title", t.Title).
			Int("notifications_sent", sent).
			Msg("due-time alert sent")
	}
}

// sendToUser sends a push notification to all subscriptions for a user.
// Returns the number of successful sends.
func sendToUser(ctx context.Context, pool *pgxpool.Pool, opts *webpush.Options, userID uuid.UUID, payload []byte, logger zerolog.Logger) int {
	subRows, err := pool.Query(ctx, queryUserSubscriptions, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID.String()).Msg("failed to load subscriptions")
		return 0
	}
	defer subRows.Close()

	var sent int
	for subRows.Next() {
		var sub pushSubscription
		if err := subRows.Scan(&sub.ID, &sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			logger.Error().Err(err).Msg("failed to scan subscription")
			continue
		}

		resp, err := webpush.SendNotification(payload, &webpush.Subscription{
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
			if _, err := pool.Exec(ctx, execDeleteSubscription, sub.ID); err != nil {
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

	return sent
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
