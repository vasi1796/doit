//go:build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// testPool creates a connection pool and truncates reminder-related tables.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://doit:changeme@localhost:5432/doit?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	// Clean up test data
	if _, err := pool.Exec(ctx, `TRUNCATE task_reminder_log, reminder_log, push_subscriptions, tasks, users CASCADE`); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}
	return pool
}

// insertTestUser creates a user and returns the user ID.
func insertTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, google_id, email, name, allowed) VALUES ($1, $2, $3, $4, true)`,
		userID, "google-"+userID.String(), userID.String()+"@test.com", "Test User",
	)
	if err != nil {
		t.Fatalf("inserting test user: %v", err)
	}
	return userID
}

// insertTask inserts a task directly into the read model for testing.
func insertTask(t *testing.T, pool *pgxpool.Pool, id, userID uuid.UUID, title string, dueDate string, dueTime *string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO tasks (id, user_id, title, due_date, due_time, position, is_completed, is_deleted)
		 VALUES ($1, $2, $3, $4, $5::time, 'a', false, false)`,
		id, userID, title, dueDate, dueTime,
	)
	if err != nil {
		t.Fatalf("inserting test task: %v", err)
	}
}

// insertSubscription inserts a push subscription for a user.
func insertSubscription(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO push_subscriptions (user_id, endpoint, key_p256dh, key_auth)
		 VALUES ($1, $2, $3, $4)`,
		userID, "https://fake-push-service.example/push/test-sub-"+uuid.NewString(),
		"fake-p256dh-key", "fake-auth-key",
	)
	if err != nil {
		t.Fatalf("inserting test subscription: %v", err)
	}
}

// dummyVAPIDOpts returns VAPID options that won't actually send (push will fail
// but that's OK — we're testing the query/idempotency logic, not webpush delivery).
func dummyVAPIDOpts() *webpush.Options {
	return &webpush.Options{
		VAPIDPublicKey:  "BDummy_Public_Key_For_Testing_Only_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		VAPIDPrivateKey: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx=",
		Subscriber:      "mailto:test@example.com",
		TTL:             60,
	}
}

func TestSendMorningDigest(t *testing.T) {
	pool := testPool(t)
	logger := zerolog.Nop()
	opts := dummyVAPIDOpts()
	ctx := context.Background()

	tests := []struct {
		name             string
		setup            func(t *testing.T) uuid.UUID
		wantLogCount     int // expected total rows in reminder_log for this user
		wantTaskCount    int // expected task_count value in the log (only checked if wantLogCount > 0)
	}{
		{
			name: "sends digest for date-only tasks",
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				insertTask(t, pool, uuid.New(), userID, "Task 1", "2026-03-24", nil)
				insertTask(t, pool, uuid.New(), userID, "Task 2", "2026-03-24", nil)
				insertSubscription(t, pool, userID)
				return userID
			},
			wantLogCount:  1,
			wantTaskCount: 2,
		},
		{
			name: "skips tasks with due_time set",
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				dueTime := "14:00:00"
				insertTask(t, pool, uuid.New(), userID, "Timed task", "2026-03-24", &dueTime)
				insertSubscription(t, pool, userID)
				return userID
			},
			wantLogCount: 0,
		},
		{
			name: "skips completed tasks",
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				taskID := uuid.New()
				insertTask(t, pool, taskID, userID, "Done task", "2026-03-24", nil)
				_, _ = pool.Exec(ctx, `UPDATE tasks SET is_completed = true WHERE id = $1`, taskID)
				insertSubscription(t, pool, userID)
				return userID
			},
			wantLogCount: 0,
		},
		{
			name: "idempotent — does not send twice",
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				insertTask(t, pool, uuid.New(), userID, "Task A", "2026-03-24", nil)
				insertSubscription(t, pool, userID)
				// Pre-insert a reminder_log entry
				_, _ = pool.Exec(ctx,
					`INSERT INTO reminder_log (user_id, sent_date, task_count) VALUES ($1, '2026-03-24', 1)`,
					userID,
				)
				return userID
			},
			wantLogCount:  1, // pre-existing record, function should not add another
			wantTaskCount: 1, // original task_count preserved (not overwritten)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _ = pool.Exec(ctx, `TRUNCATE task_reminder_log, reminder_log, push_subscriptions, tasks CASCADE`)

			userID := tc.setup(t)
			now := time.Date(2026, 3, 24, 8, 0, 0, 0, time.UTC)

			sendMorningDigest(ctx, pool, opts, now, logger)

			var count int
			_ = pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM reminder_log WHERE user_id = $1 AND sent_date = '2026-03-24'`,
				userID,
			).Scan(&count)

			if count != tc.wantLogCount {
				t.Errorf("reminder_log count = %d, want %d", count, tc.wantLogCount)
			}

			if tc.wantLogCount > 0 && tc.wantTaskCount > 0 {
				var taskCount int
				_ = pool.QueryRow(ctx,
					`SELECT task_count FROM reminder_log WHERE user_id = $1 AND sent_date = '2026-03-24'`,
					userID,
				).Scan(&taskCount)
				if taskCount != tc.wantTaskCount {
					t.Errorf("digest task_count = %d, want %d", taskCount, tc.wantTaskCount)
				}
			}
		})
	}
}

func TestSendDueTimeAlerts(t *testing.T) {
	pool := testPool(t)
	logger := zerolog.Nop()
	opts := dummyVAPIDOpts()
	ctx := context.Background()

	tests := []struct {
		name         string
		setup        func(t *testing.T) uuid.UUID
		now          time.Time
		wantLogCount int // expected total rows in task_reminder_log for this task
	}{
		{
			name: "sends alert for task due before now",
			now:  time.Date(2026, 3, 24, 14, 5, 0, 0, time.UTC),
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				taskID := uuid.New()
				dueTime := "14:00:00"
				insertTask(t, pool, taskID, userID, "Meeting prep", "2026-03-24", &dueTime)
				insertSubscription(t, pool, userID)
				return taskID
			},
			wantLogCount: 1,
		},
		{
			name: "catches up overdue task from earlier today",
			now:  time.Date(2026, 3, 24, 18, 0, 0, 0, time.UTC),
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				taskID := uuid.New()
				dueTime := "09:00:00"
				insertTask(t, pool, taskID, userID, "Missed morning task", "2026-03-24", &dueTime)
				insertSubscription(t, pool, userID)
				return taskID
			},
			wantLogCount: 1,
		},
		{
			name: "skips future task",
			now:  time.Date(2026, 3, 24, 14, 5, 0, 0, time.UTC),
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				taskID := uuid.New()
				dueTime := "16:00:00"
				insertTask(t, pool, taskID, userID, "Later task", "2026-03-24", &dueTime)
				insertSubscription(t, pool, userID)
				return taskID
			},
			wantLogCount: 0,
		},
		{
			name: "skips task with no due_time",
			now:  time.Date(2026, 3, 24, 14, 5, 0, 0, time.UTC),
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				taskID := uuid.New()
				insertTask(t, pool, taskID, userID, "Date only", "2026-03-24", nil)
				insertSubscription(t, pool, userID)
				return taskID
			},
			wantLogCount: 0,
		},
		{
			name: "idempotent — does not alert twice",
			now:  time.Date(2026, 3, 24, 14, 5, 0, 0, time.UTC),
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				taskID := uuid.New()
				dueTime := "14:00:00"
				insertTask(t, pool, taskID, userID, "Already alerted", "2026-03-24", &dueTime)
				insertSubscription(t, pool, userID)
				// Pre-insert task_reminder_log entry
				_, _ = pool.Exec(ctx,
					`INSERT INTO task_reminder_log (task_id, due_date) VALUES ($1, '2026-03-24')`,
					taskID,
				)
				return taskID
			},
			wantLogCount: 1, // pre-existing record, function should not add another
		},
		{
			name: "skips deleted task",
			now:  time.Date(2026, 3, 24, 14, 5, 0, 0, time.UTC),
			setup: func(t *testing.T) uuid.UUID {
				userID := insertTestUser(t, pool)
				taskID := uuid.New()
				dueTime := "14:00:00"
				insertTask(t, pool, taskID, userID, "Deleted task", "2026-03-24", &dueTime)
				_, _ = pool.Exec(ctx, `UPDATE tasks SET is_deleted = true WHERE id = $1`, taskID)
				insertSubscription(t, pool, userID)
				return taskID
			},
			wantLogCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _ = pool.Exec(ctx, `TRUNCATE task_reminder_log, reminder_log, push_subscriptions, tasks CASCADE`)

			taskID := tc.setup(t)

			sendDueTimeAlerts(ctx, pool, opts, tc.now, logger)

			var count int
			_ = pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM task_reminder_log WHERE task_id = $1`,
				taskID,
			).Scan(&count)

			if count != tc.wantLogCount {
				t.Errorf("task_reminder_log count = %d, want %d", count, tc.wantLogCount)
			}
		})
	}
}

func TestTick_HourGating(t *testing.T) {
	// This test verifies that tick only calls sendMorningDigest at the reminder hour.
	// We can't easily mock sendMorningDigest, but we can verify through the DB:
	// if tick is called outside reminder hour, no reminder_log entries should be created.
	pool := testPool(t)
	logger := zerolog.Nop()
	opts := dummyVAPIDOpts()
	ctx := context.Background()

	userID := insertTestUser(t, pool)
	insertTask(t, pool, uuid.New(), userID, "Test task", "2026-03-24", nil)
	insertSubscription(t, pool, userID)

	tests := []struct {
		name         string
		hour         int
		reminderHour int
		wantDigest   bool
	}{
		{name: "sends at reminder hour", hour: 8, reminderHour: 8, wantDigest: true},
		{name: "skips before reminder hour", hour: 7, reminderHour: 8, wantDigest: false},
		{name: "skips after reminder hour", hour: 9, reminderHour: 8, wantDigest: false},
		{name: "custom reminder hour", hour: 10, reminderHour: 10, wantDigest: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _ = pool.Exec(ctx, `TRUNCATE task_reminder_log, reminder_log CASCADE`)

			now := time.Date(2026, 3, 24, tc.hour, 5, 0, 0, time.UTC)
			tick(ctx, pool, opts, tc.reminderHour, now, logger)

			var exists bool
			_ = pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM reminder_log WHERE user_id = $1 AND sent_date = '2026-03-24')`,
				userID,
			).Scan(&exists)

			if tc.wantDigest && !exists {
				t.Error("expected morning digest at reminder hour, but none was logged")
			}
			if !tc.wantDigest && exists {
				t.Error("expected no morning digest outside reminder hour, but one was logged")
			}
		})
	}
}

func TestSendToUser_StaleSubscription(t *testing.T) {
	// Test that sendToUser handles push delivery errors gracefully.
	// With dummy VAPID keys, webpush.SendNotification will fail,
	// but sendToUser should not panic and should return 0.
	pool := testPool(t)
	logger := zerolog.Nop()
	opts := dummyVAPIDOpts()
	ctx := context.Background()

	userID := insertTestUser(t, pool)
	insertSubscription(t, pool, userID)

	payload, _ := json.Marshal(map[string]string{"title": "Test", "body": "test"})
	sent := sendToUser(ctx, pool, opts, userID, payload, logger)

	if sent != 0 {
		t.Errorf("sendToUser with dummy keys should return 0, got %d", sent)
	}
}

func TestSendToUser_NoSubscriptions(t *testing.T) {
	// A user with no subscriptions should return 0 sends without errors.
	pool := testPool(t)
	logger := zerolog.Nop()
	opts := dummyVAPIDOpts()
	ctx := context.Background()

	userID := insertTestUser(t, pool)
	// No subscriptions inserted

	payload, _ := json.Marshal(map[string]string{"title": "Test", "body": "test"})
	sent := sendToUser(ctx, pool, opts, userID, payload, logger)

	if sent != 0 {
		t.Errorf("sendToUser with no subscriptions should return 0, got %d", sent)
	}
}
