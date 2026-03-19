package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

type mockEventLoader struct {
	events    []eventstore.Event
	loadErr   error
	appendErr error
	appended  []eventstore.Event
}

func (m *mockEventLoader) LoadByAggregate(_ context.Context, _ uuid.UUID) ([]eventstore.Event, error) {
	return m.events, m.loadErr
}

func (m *mockEventLoader) Append(_ context.Context, events []eventstore.Event) error {
	if m.appendErr != nil {
		return m.appendErr
	}
	m.appended = append(m.appended, events...)
	return nil
}

type mockProjector struct {
	projected  []eventstore.Event
	projectErr error
}

func (m *mockProjector) Project(_ context.Context, events []eventstore.Event) error {
	if m.projectErr != nil {
		return m.projectErr
	}
	m.projected = append(m.projected, events...)
	return nil
}

func TestCommandHandlerCreateTask(t *testing.T) {
	store := &mockEventLoader{}
	proj := &mockProjector{}
	handler := NewCommandHandler(store, proj)

	cmd := CreateTask{
		TaskID:   uuid.New(),
		UserID:   uuid.New(),
		Title:    "Test task",
		Priority: 1,
		ListID:   uuid.New(),
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
	if len(proj.projected) != 1 {
		t.Fatalf("projected %d events, want 1", len(proj.projected))
	}
}

func TestCommandHandlerCompleteTask(t *testing.T) {
	aggID := uuid.New()
	store := &mockEventLoader{
		events: []eventstore.Event{
			taskEvent(aggID, eventstore.EventTaskCreated, 1, TaskCreatedPayload{Title: "x"}),
		},
	}
	proj := &mockProjector{}
	handler := NewCommandHandler(store, proj)

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
	store := &mockEventLoader{events: []eventstore.Event{}}
	proj := &mockProjector{}
	handler := NewCommandHandler(store, proj)

	err := handler.CompleteTask(context.Background(), uuid.New(), testUserID, CompleteTask{CompletedAt: testNow})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("got error %v, want %v", err, ErrTaskNotFound)
	}
}

func TestCommandHandlerAppendError(t *testing.T) {
	store := &mockEventLoader{
		appendErr: eventstore.ErrVersionConflict,
	}
	proj := &mockProjector{}
	handler := NewCommandHandler(store, proj)

	cmd := CreateTask{
		TaskID:   uuid.New(),
		UserID:   uuid.New(),
		Title:    "Test",
		Priority: 0,
		Position: "a",
	}

	err := handler.CreateTask(context.Background(), cmd)
	if !errors.Is(err, eventstore.ErrVersionConflict) {
		t.Fatalf("got error %v, want %v", err, eventstore.ErrVersionConflict)
	}
	if len(proj.projected) != 0 {
		t.Errorf("projected %d events, want 0 (append failed)", len(proj.projected))
	}
}

func TestCommandHandlerLoadError(t *testing.T) {
	loadErr := errors.New("db connection failed")
	store := &mockEventLoader{loadErr: loadErr}
	proj := &mockProjector{}
	handler := NewCommandHandler(store, proj)

	err := handler.CompleteTask(context.Background(), uuid.New(), testUserID, CompleteTask{CompletedAt: testNow})
	if !errors.Is(err, loadErr) {
		t.Fatalf("got error %v, want %v", err, loadErr)
	}
}

func TestCommandHandlerProjectionError(t *testing.T) {
	store := &mockEventLoader{}
	projErr := errors.New("projection failed")
	proj := &mockProjector{projectErr: projErr}
	handler := NewCommandHandler(store, proj)

	cmd := CreateTask{
		TaskID:   uuid.New(),
		UserID:   uuid.New(),
		Title:    "Test",
		Priority: 0,
		Position: "a",
	}

	err := handler.CreateTask(context.Background(), cmd)
	if !errors.Is(err, projErr) {
		t.Fatalf("got error %v, want %v", err, projErr)
	}
	if len(store.appended) != 1 {
		t.Errorf("appended %d events, want 1 (append should succeed before projection)", len(store.appended))
	}
}
