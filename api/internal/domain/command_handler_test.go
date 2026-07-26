package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
)

type mockEventStore struct {
	events    []eventstore.Event
	loadErr   error
	appendErr error
	appended  []eventstore.Event
}

func (m *mockEventStore) LoadByAggregate(_ context.Context, _ uuid.UUID) ([]eventstore.Event, error) {
	return m.events, m.loadErr
}

func (m *mockEventStore) Append(_ context.Context, events []eventstore.Event) error {
	if m.appendErr != nil {
		return m.appendErr
	}
	m.appended = append(m.appended, events...)
	return nil
}

func (m *mockEventStore) AppendTx(_ context.Context, _ pgx.Tx, events []eventstore.Event) error {
	if m.appendErr != nil {
		return m.appendErr
	}
	m.appended = append(m.appended, events...)
	return nil
}

func (m *mockEventStore) InsertOutbox(_ context.Context, _ pgx.Tx, _ []eventstore.Event) error {
	return nil
}

// mockTx implements pgx.Tx for testing (no-op transaction).
type mockTx struct{ pgx.Tx }

func (m mockTx) Commit(_ context.Context) error   { return nil }
func (m mockTx) Rollback(_ context.Context) error { return nil }

type mockPool struct{}

func (m *mockPool) Begin(_ context.Context) (pgx.Tx, error) {
	return mockTx{}, nil
}

func TestCommandHandler(t *testing.T) {
	loadErr := errors.New("db connection failed")

	tests := []struct {
		name           string
		store          *mockEventStore
		action         func(h *CommandHandler) error
		wantErr        error
		wantAppended   int
		checkEventType eventstore.EventType
		checkVersion   int
	}{
		{
			name:  "create task appends TaskCreated event",
			store: &mockEventStore{},
			action: func(h *CommandHandler) error {
				return h.CreateTask(context.Background(), CreateTask{
					TaskID:   uuid.New(),
					UserID:   uuid.New(),
					Title:    "Test task",
					Priority: 1,
					Position: "a",
				})
			},
			wantAppended:   1,
			checkEventType: eventstore.EventTaskCreated,
		},
		{
			name: "complete task appends event with version 2",
			store: &mockEventStore{
				events: []eventstore.Event{
					taskEvent(uuid.MustParse("00000000-0000-0000-0000-000000000001"), eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
				},
			},
			action: func(h *CommandHandler) error {
				return h.CompleteTask(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000001"), testUserID, CompleteTask{CompletedAt: testNow})
			},
			wantAppended: 1,
			checkVersion: 2,
		},
		{
			name:  "complete task returns ErrTaskNotFound for empty event history",
			store: &mockEventStore{events: []eventstore.Event{}},
			action: func(h *CommandHandler) error {
				return h.CompleteTask(context.Background(), uuid.New(), testUserID, CompleteTask{CompletedAt: testNow})
			},
			wantErr: ErrTaskNotFound,
		},
		{
			name:  "create task returns ErrVersionConflict on append error",
			store: &mockEventStore{appendErr: eventstore.ErrVersionConflict},
			action: func(h *CommandHandler) error {
				return h.CreateTask(context.Background(), CreateTask{
					TaskID:   uuid.New(),
					UserID:   uuid.New(),
					Title:    "Test",
					Priority: 0,
					Position: "a",
				})
			},
			wantErr: ErrVersionConflict,
		},
		{
			name:  "complete task returns load error",
			store: &mockEventStore{loadErr: loadErr},
			action: func(h *CommandHandler) error {
				return h.CompleteTask(context.Background(), uuid.New(), testUserID, CompleteTask{CompletedAt: testNow})
			},
			wantErr: loadErr,
		},
		{
			name: "update list with rename and reorder appends both events",
			store: &mockEventStore{
				events: []eventstore.Event{
					taskEvent(uuid.MustParse("00000000-0000-0000-0000-000000000002"), eventstore.EventListCreated, 1, ListCreatedPayload{Name: "Work", Position: "a"}),
				},
			},
			action: func(h *CommandHandler) error {
				name, position := "Renamed", "aO"
				return h.UpdateList(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000002"), testUserID, UpdateList{Name: &name, Position: &position})
			},
			wantAppended:   2,
			checkEventType: eventstore.EventListNameUpdated,
		},
		{
			name: "update label with position only appends LabelReordered",
			store: &mockEventStore{
				events: []eventstore.Event{
					taskEvent(uuid.MustParse("00000000-0000-0000-0000-000000000003"), eventstore.EventLabelCreated, 1, LabelCreatedPayload{Name: "Urgent"}),
				},
			},
			action: func(h *CommandHandler) error {
				position := "aO"
				return h.UpdateLabel(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000003"), testUserID, UpdateLabel{Position: &position})
			},
			wantAppended:   1,
			checkEventType: eventstore.EventLabelReordered,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewCommandHandler(tc.store, &mockPool{}, hlc.New())

			err := tc.action(handler)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantAppended > 0 && len(tc.store.appended) != tc.wantAppended {
				t.Fatalf("appended %d events, want %d", len(tc.store.appended), tc.wantAppended)
			}

			if tc.checkEventType != "" && tc.store.appended[0].EventType != tc.checkEventType {
				t.Errorf("EventType = %q, want %q", tc.store.appended[0].EventType, tc.checkEventType)
			}

			if tc.checkVersion > 0 && tc.store.appended[0].Version != tc.checkVersion {
				t.Errorf("Version = %d, want %d", tc.store.appended[0].Version, tc.checkVersion)
			}
		})
	}
}

func TestUpdateListInvalidFieldAppendsNothing(t *testing.T) {
	listID := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	store := &mockEventStore{
		events: []eventstore.Event{
			taskEvent(listID, eventstore.EventListCreated, 1, ListCreatedPayload{Name: "Work", Position: "a"}),
		},
	}
	handler := NewCommandHandler(store, &mockPool{}, hlc.New())

	name, position := "Renamed", ""
	err := handler.UpdateList(context.Background(), listID, testUserID, UpdateList{Name: &name, Position: &position})

	if !errors.Is(err, ErrEmptyPosition) {
		t.Fatalf("got error %v, want %v", err, ErrEmptyPosition)
	}
	if len(store.appended) != 0 {
		t.Errorf("appended %d events, want 0 — a partially-valid edit must append nothing", len(store.appended))
	}
}
