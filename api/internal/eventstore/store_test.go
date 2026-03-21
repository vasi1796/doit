//go:build integration

package eventstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/eventstore"
)

func setupTest(t *testing.T) *eventstore.Store {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://doit:changeme@localhost:5432/doit?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	// Truncate for test isolation
	_, err = pool.Exec(context.Background(), "TRUNCATE events")
	if err != nil {
		t.Fatalf("truncating events table: %v", err)
	}

	return eventstore.New(pool, zerolog.Nop())
}

func makeEvent(aggregateID, userID uuid.UUID, eventType eventstore.EventType, version int, ts time.Time) eventstore.Event {
	return eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   aggregateID,
		AggregateType: eventstore.AggregateTypeTask,
		EventType:     eventType,
		UserID:        userID,
		Data:          json.RawMessage(`{"title":"test task"}`),
		Timestamp:     ts,
		Version:       version,
	}
}

func TestAppend(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	aggID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		events     []eventstore.Event
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:   "append single event",
			events: []eventstore.Event{makeEvent(aggID, userID, eventstore.EventTaskCreated, 1, now)},
		},
		{
			name: "append multiple events for same aggregate",
			events: []eventstore.Event{
				makeEvent(aggID, userID, eventstore.EventTaskCompleted, 2, now.Add(time.Second)),
				makeEvent(aggID, userID, eventstore.EventTaskUncompleted, 3, now.Add(2*time.Second)),
			},
		},
		{
			name:    "version conflict returns ErrVersionConflict",
			events:  []eventstore.Event{makeEvent(aggID, userID, eventstore.EventTaskCreated, 1, now)},
			wantErr: eventstore.ErrVersionConflict,
		},
		{
			name:    "empty events returns ErrNoEvents",
			events:  []eventstore.Event{},
			wantErr: eventstore.ErrNoEvents,
		},
		{
			name: "mismatched aggregate IDs returns error",
			events: []eventstore.Event{
				makeEvent(uuid.New(), userID, eventstore.EventTaskCreated, 1, now),
				makeEvent(uuid.New(), userID, eventstore.EventTaskCreated, 1, now),
			},
			wantAnyErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := store.Append(ctx, tc.events)

			if tc.wantAnyErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadByAggregate(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()

	t.Run("returns events in version order", func(t *testing.T) {
		aggID := uuid.New()
		for _, e := range []eventstore.Event{
			makeEvent(aggID, userID, eventstore.EventTaskCreated, 1, now),
			makeEvent(aggID, userID, eventstore.EventTaskMoved, 2, now.Add(time.Second)),
			makeEvent(aggID, userID, eventstore.EventTaskCompleted, 3, now.Add(2*time.Second)),
		} {
			if err := store.Append(ctx, []eventstore.Event{e}); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}

		got, err := store.LoadByAggregate(ctx, aggID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d events, want 3", len(got))
		}
		for i, want := range []int{1, 2, 3} {
			if got[i].Version != want {
				t.Errorf("event[%d].Version = %d, want %d", i, got[i].Version, want)
			}
		}
	})

	t.Run("returns empty for unknown aggregate", func(t *testing.T) {
		got, err := store.LoadByAggregate(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %d events, want 0", len(got))
		}
	})

	t.Run("does not return events from other aggregates", func(t *testing.T) {
		agg1 := uuid.New()
		agg2 := uuid.New()

		if err := store.Append(ctx, []eventstore.Event{
			makeEvent(agg1, userID, eventstore.EventTaskCreated, 1, now),
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := store.Append(ctx, []eventstore.Event{
			makeEvent(agg2, userID, eventstore.EventListCreated, 1, now),
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := store.LoadByAggregate(ctx, agg1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d events, want 1", len(got))
		}
		if got[0].AggregateID != agg1 {
			t.Errorf("got aggregate %v, want %v", got[0].AggregateID, agg1)
		}
	})
}

func TestLoadByAggregateFromVersion(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()
	aggID := uuid.New()

	// Insert 5 events
	for i := 1; i <= 5; i++ {
		if err := store.Append(ctx, []eventstore.Event{
			makeEvent(aggID, userID, eventstore.EventTaskCreated, i, now.Add(time.Duration(i)*time.Second)),
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	tests := []struct {
		name         string
		fromVersion  int
		wantVersions []int
	}{
		{
			name:         "returns events from specified version",
			fromVersion:  3,
			wantVersions: []int{3, 4, 5},
		},
		{
			name:         "returns empty when fromVersion exceeds max",
			fromVersion:  10,
			wantVersions: []int{},
		},
		{
			name:         "returns all from version 1",
			fromVersion:  1,
			wantVersions: []int{1, 2, 3, 4, 5},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := store.LoadByAggregateFromVersion(ctx, aggID, tc.fromVersion)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.wantVersions) {
				t.Fatalf("got %d events, want %d", len(got), len(tc.wantVersions))
			}
			for i, want := range tc.wantVersions {
				if got[i].Version != want {
					t.Errorf("event[%d].Version = %d, want %d", i, got[i].Version, want)
				}
			}
		})
	}
}

func TestLoadByUserSince(t *testing.T) {
	store := setupTest(t)
	ctx := context.Background()
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	user1 := uuid.New()
	user2 := uuid.New()

	// User1: events at baseTime+1s, +2s, +3s
	for i := 1; i <= 3; i++ {
		agg := uuid.New()
		if err := store.Append(ctx, []eventstore.Event{
			makeEvent(agg, user1, eventstore.EventTaskCreated, 1, baseTime.Add(time.Duration(i)*time.Second)),
		}); err != nil {
			t.Fatalf("setup user1: %v", err)
		}
	}

	// User2: one event at baseTime+2s
	agg2 := uuid.New()
	if err := store.Append(ctx, []eventstore.Event{
		makeEvent(agg2, user2, eventstore.EventTaskCreated, 1, baseTime.Add(2*time.Second)),
	}); err != nil {
		t.Fatalf("setup user2: %v", err)
	}

	tests := []struct {
		name         string
		userID       uuid.UUID
		since        time.Time
		sinceCounter int
		wantCount    int
	}{
		{
			name:         "returns events after timestamp",
			userID:       user1,
			since:        baseTime.Add(1 * time.Second),
			sinceCounter: 0,
			wantCount:    2, // +2s and +3s (strictly after +1s)
		},
		{
			name:         "does not return events from other users",
			userID:       user2,
			since:        baseTime,
			sinceCounter: 0,
			wantCount:    1,
		},
		{
			name:         "returns empty for future timestamp",
			userID:       user1,
			since:        baseTime.Add(1 * time.Hour),
			sinceCounter: 0,
			wantCount:    0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := store.LoadByUserSince(ctx, tc.userID, tc.since, tc.sinceCounter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Fatalf("got %d events, want %d", len(got), tc.wantCount)
			}
		})
	}
}
