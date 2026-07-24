package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

func TestListHandleCreate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     CreateList
		wantErr error
	}{
		{
			name: "valid list",
			cmd:  CreateList{ListID: uuid.New(), UserID: testUserID, Name: "Work", Colour: "#ff0000", Position: "a"},
		},
		{
			name:    "empty name",
			cmd:     CreateList{ListID: uuid.New(), UserID: testUserID, Name: "", Colour: "#ff0000", Position: "a"},
			wantErr: ErrEmptyName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agg := NewListAggregate()
			events, err := agg.HandleCreate(tc.cmd, testHLC)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}
			if events[0].Version != 1 {
				t.Errorf("Version = %d, want 1", events[0].Version)
			}
		})
	}
}

func TestListHandleCreateDuplicate(t *testing.T) {
	agg := NewListAggregate()
	cmd := CreateList{ListID: uuid.New(), UserID: testUserID, Name: "Work", Colour: "#ff0000", Position: "a"}

	// Create the list
	events, err := agg.HandleCreate(cmd, testHLC)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Apply the event so state reflects creation
	agg.Apply(events[0])

	// Try creating again
	_, err = agg.HandleCreate(cmd, testHLC)
	if !errors.Is(err, ErrListAlreadyCreated) {
		t.Fatalf("got error %v, want %v", err, ErrListAlreadyCreated)
	}
}

func TestListHandleUpdate(t *testing.T) {
	type setup int
	const (
		notCreated setup = iota
		active
		deleted
	)

	newAgg := func(s setup) *ListAggregate {
		agg := NewListAggregate()
		if s == notCreated {
			return agg
		}
		events, err := agg.HandleCreate(CreateList{ListID: uuid.New(), UserID: testUserID, Name: "Work", Colour: "#ff0000", Position: "a"}, testHLC)
		if err != nil {
			t.Fatalf("setup create: %v", err)
		}
		agg.Apply(events[0])
		if s == deleted {
			events, err = agg.HandleDelete(DeleteList{}, testHLC)
			if err != nil {
				t.Fatalf("setup delete: %v", err)
			}
			agg.Apply(events[0])
		}
		return agg
	}

	tests := []struct {
		name      string
		setup     setup
		update    func(agg *ListAggregate) ([]eventstore.Event, error)
		wantType  eventstore.EventType
		wantErr   error
	}{
		{
			name:     "rename active list",
			setup:    active,
			update:   func(a *ListAggregate) ([]eventstore.Event, error) { return a.HandleUpdateName(UpdateListName{Name: "Home"}, testHLC) },
			wantType: eventstore.EventListNameUpdated,
		},
		{
			name:    "rename to empty name",
			setup:   active,
			update:  func(a *ListAggregate) ([]eventstore.Event, error) { return a.HandleUpdateName(UpdateListName{Name: ""}, testHLC) },
			wantErr: ErrEmptyName,
		},
		{
			name:    "rename list that was never created",
			setup:   notCreated,
			update:  func(a *ListAggregate) ([]eventstore.Event, error) { return a.HandleUpdateName(UpdateListName{Name: "Home"}, testHLC) },
			wantErr: ErrListNotFound,
		},
		{
			name:    "rename deleted list",
			setup:   deleted,
			update:  func(a *ListAggregate) ([]eventstore.Event, error) { return a.HandleUpdateName(UpdateListName{Name: "Home"}, testHLC) },
			wantErr: ErrListAlreadyDeleted,
		},
		{
			name:     "recolour active list",
			setup:    active,
			update:   func(a *ListAggregate) ([]eventstore.Event, error) { return a.HandleUpdateColour(UpdateListColour{Colour: "#00ff00"}, testHLC) },
			wantType: eventstore.EventListColourUpdated,
		},
		{
			name:    "recolour deleted list",
			setup:   deleted,
			update:  func(a *ListAggregate) ([]eventstore.Event, error) { return a.HandleUpdateColour(UpdateListColour{Colour: "#00ff00"}, testHLC) },
			wantErr: ErrListAlreadyDeleted,
		},
		{
			name:    "recolour to empty colour",
			setup:   active,
			update:  func(a *ListAggregate) ([]eventstore.Event, error) { return a.HandleUpdateColour(UpdateListColour{Colour: ""}, testHLC) },
			wantErr: ErrEmptyColour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agg := newAgg(tc.setup)
			events, err := tc.update(agg)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}
			if events[0].EventType != tc.wantType {
				t.Errorf("EventType = %s, want %s", events[0].EventType, tc.wantType)
			}
		})
	}
}
