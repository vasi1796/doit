package recurring

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/broker"
	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
)

type stubLoader struct {
	events []eventstore.Event
	err    error
}

func (s *stubLoader) LoadByAggregate(_ context.Context, _ uuid.UUID) ([]eventstore.Event, error) {
	return s.events, s.err
}

type stubCreator struct {
	err   error
	calls []domain.CreateTask
}

func (s *stubCreator) CreateTask(_ context.Context, cmd domain.CreateTask) error {
	s.calls = append(s.calls, cmd)
	return s.err
}

func taskEvent(aggID, userID uuid.UUID, et eventstore.EventType, version int, payload any) eventstore.Event {
	data, _ := json.Marshal(payload) //nolint:errcheck
	return eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   aggID,
		AggregateType: eventstore.AggregateTypeTask,
		EventType:     et,
		UserID:        userID,
		Data:          data,
		Timestamp:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Version:       version,
	}
}

func TestNextTaskID(t *testing.T) {
	eventA := uuid.New()
	eventB := uuid.New()

	tests := []struct {
		name  string
		a, b  uuid.UUID
		equal bool
	}{
		{name: "same event yields same task ID", a: eventA, b: eventA, equal: true},
		{name: "different events yield different task IDs", a: eventA, b: eventB, equal: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextTaskID(tc.a) == NextTaskID(tc.b)
			if got != tc.equal {
				t.Errorf("NextTaskID equality = %v, want %v", got, tc.equal)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	userID := uuid.New()
	taskID := uuid.New()
	labelID := uuid.New()
	dueDate := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)

	recurringHistory := []eventstore.Event{
		taskEvent(taskID, userID, eventstore.EventTaskCreated, 1, domain.TaskCreatedPayload{
			Title:    "Water plants",
			Priority: domain.PriorityLow,
			DueDate:  &dueDate,
			Position: "a",
		}),
		taskEvent(taskID, userID, eventstore.EventTaskRecurrenceUpdated, 2, domain.TaskRecurrenceUpdatedPayload{
			RecurrenceRule: domain.RecurrenceWeekly,
		}),
		taskEvent(taskID, userID, eventstore.EventLabelAdded, 3, domain.LabelAddedPayload{
			LabelID: labelID,
		}),
	}

	tests := []struct {
		name        string
		events      []eventstore.Event
		createErr   error
		wantErr     bool
		wantCreates int
	}{
		{
			name:        "creates next occurrence with rule and labels",
			events:      recurringHistory,
			wantCreates: 1,
		},
		{
			name: "non-recurring task is ignored",
			events: []eventstore.Event{
				taskEvent(taskID, userID, eventstore.EventTaskCreated, 1, domain.TaskCreatedPayload{
					Title: "One-off", Position: "a",
				}),
			},
			wantCreates: 0,
		},
		{
			name:        "empty history is ignored",
			events:      nil,
			wantCreates: 0,
		},
		{
			name:        "version conflict on redelivery is treated as already processed",
			events:      recurringHistory,
			createErr:   domain.ErrVersionConflict,
			wantCreates: 1,
		},
		{
			name:        "other create errors propagate for redelivery",
			events:      recurringHistory,
			createErr:   errors.New("db down"),
			wantErr:     true,
			wantCreates: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			loader := &stubLoader{events: tc.events}
			creator := &stubCreator{err: tc.createErr}
			em := broker.EventMessage{
				EventID:     uuid.New(),
				AggregateID: taskID,
				EventType:   string(eventstore.EventTaskCompleted),
				UserID:      userID,
			}

			err := Handle(context.Background(), loader, creator, em, zerolog.Nop())

			if tc.wantErr != (err != nil) {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if len(creator.calls) != tc.wantCreates {
				t.Fatalf("CreateTask called %d times, want %d", len(creator.calls), tc.wantCreates)
			}
			if tc.wantCreates == 1 && tc.createErr == nil {
				cmd := creator.calls[0]
				if cmd.TaskID != NextTaskID(em.EventID) {
					t.Errorf("TaskID = %v, want deterministic %v", cmd.TaskID, NextTaskID(em.EventID))
				}
				if cmd.RecurrenceRule != domain.RecurrenceWeekly {
					t.Errorf("RecurrenceRule = %q, want weekly", cmd.RecurrenceRule)
				}
				if len(cmd.Labels) != 1 || cmd.Labels[0] != labelID {
					t.Errorf("Labels = %v, want [%v]", cmd.Labels, labelID)
				}
				wantDue := dueDate.AddDate(0, 0, 7)
				if cmd.DueDate == nil || !cmd.DueDate.Equal(wantDue) {
					t.Errorf("DueDate = %v, want %v", cmd.DueDate, wantDue)
				}
			}
		})
	}
}
