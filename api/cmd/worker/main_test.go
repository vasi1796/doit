package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/eventstore"
)

type stubProjector struct {
	failures int
	calls    int
}

func (s *stubProjector) Project(_ context.Context, _ []eventstore.Event) error {
	s.calls++
	if s.calls <= s.failures {
		return errors.New("transient failure")
	}
	return nil
}

func TestProjectWithRetry(t *testing.T) {
	tests := []struct {
		name      string
		failures  int
		attempts  int
		wantErr   bool
		wantCalls int
	}{
		{
			name:      "succeeds first attempt",
			failures:  0,
			attempts:  5,
			wantCalls: 1,
		},
		{
			name:      "transient failure recovers within budget",
			failures:  2,
			attempts:  5,
			wantCalls: 3,
		},
		{
			name:      "exhausted retries return the error",
			failures:  10,
			attempts:  3,
			wantErr:   true,
			wantCalls: 3,
		},
		{
			name:      "attempts below one clamps to a single try",
			failures:  10,
			attempts:  0,
			wantErr:   true,
			wantCalls: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubProjector{failures: tc.failures}
			event := eventstore.Event{ID: uuid.New()}

			err := projectWithRetry(context.Background(), stub, event, tc.attempts, time.Millisecond, zerolog.Nop())

			if tc.wantErr != (err != nil) {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if stub.calls != tc.wantCalls {
				t.Errorf("Project called %d times, want %d", stub.calls, tc.wantCalls)
			}
		})
	}
}

func TestProjectWithRetryStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stub := &stubProjector{failures: 10}

	err := projectWithRetry(ctx, stub, eventstore.Event{ID: uuid.New()}, 5, time.Hour, zerolog.Nop())

	if err == nil {
		t.Fatal("expected error after cancellation")
	}
	if stub.calls != 1 {
		t.Errorf("Project called %d times, want 1 (no retries after cancel)", stub.calls)
	}
}
