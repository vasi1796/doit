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

func TestCommandHandlerCreateTask(t *testing.T) {
	store := &mockEventStore{}
	handler := NewCommandHandler(store, &mockPool{}, hlc.New())

	cmd := CreateTask{
		TaskID:   uuid.New(),
		UserID:   uuid.New(),
		Title:    "Test task",
		Priority: 1,
		Position: "a",
	}

	err := handler.CreateTask(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.appended) != 1 {
		t.Fatalf("appended %d events, want 1", len(store.appended))
	}
	if store.appended[0].EventType != eventstore.EventTaskCreated {
		t.Errorf("EventType = %q, want %q", store.appended[0].EventType, eventstore.EventTaskCreated)
	}
}

func TestCommandHandlerCompleteTask(t *testing.T) {
	aggID := uuid.New()
	store := &mockEventStore{
		events: []eventstore.Event{
			taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
		},
	}
	handler := NewCommandHandler(store, &mockPool{}, hlc.New())

	err := handler.CompleteTask(context.Background(), aggID, testUserID, CompleteTask{CompletedAt: testNow})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.appended) != 1 {
		t.Fatalf("appended %d events, want 1", len(store.appended))
	}
	if store.appended[0].Version != 2 {
		t.Errorf("Version = %d, want 2", store.appended[0].Version)
	}
}

func TestCommandHandlerTaskNotFound(t *testing.T) {
	store := &mockEventStore{events: []eventstore.Event{}}
	handler := NewCommandHandler(store, &mockPool{}, hlc.New())

	err := handler.CompleteTask(context.Background(), uuid.New(), testUserID, CompleteTask{CompletedAt: testNow})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got error %v, want %v", err, ErrTaskNotFound)
	}
}

func TestCommandHandlerAppendError(t *testing.T) {
	store := &mockEventStore{
		appendErr: eventstore.ErrVersionConflict,
	}
	handler := NewCommandHandler(store, &mockPool{}, hlc.New())

	cmd := CreateTask{
		TaskID:   uuid.New(),
		UserID:   uuid.New(),
		Title:    "Test",
		Priority: 0,
		Position: "a",
	}

	err := handler.CreateTask(context.Background(), cmd)
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("got error %v, want %v", err, ErrVersionConflict)
	}
}

func TestCommandHandlerLoadError(t *testing.T) {
	loadErr := errors.New("db connection failed")
	store := &mockEventStore{loadErr: loadErr}
	handler := NewCommandHandler(store, &mockPool{}, hlc.New())

	err := handler.CompleteTask(context.Background(), uuid.New(), testUserID, CompleteTask{CompletedAt: testNow})
	if !errors.Is(err, loadErr) {
		t.Fatalf("got error %v, want %v", err, loadErr)
	}
}
