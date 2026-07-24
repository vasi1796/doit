package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

func TestLabelHandleCreate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     CreateLabel
		wantErr error
	}{
		{
			name: "valid label",
			cmd:  CreateLabel{LabelID: uuid.New(), UserID: testUserID, Name: "urgent", Colour: "#ff0000"},
		},
		{
			name:    "empty name",
			cmd:     CreateLabel{LabelID: uuid.New(), UserID: testUserID, Name: "", Colour: "#ff0000"},
			wantErr: ErrEmptyName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agg := NewLabelAggregate()
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

func TestLabelHandleCreateDuplicate(t *testing.T) {
	agg := NewLabelAggregate()
	cmd := CreateLabel{LabelID: uuid.New(), UserID: testUserID, Name: "urgent", Colour: "#ff0000"}

	events, err := agg.HandleCreate(cmd, testHLC)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	agg.Apply(events[0])

	_, err = agg.HandleCreate(cmd, testHLC)
	if !errors.Is(err, ErrLabelAlreadyCreated) {
		t.Fatalf("got error %v, want %v", err, ErrLabelAlreadyCreated)
	}
}

func TestLabelHandleUpdate(t *testing.T) {
	newActive := func() *LabelAggregate {
		agg := NewLabelAggregate()
		events, err := agg.HandleCreate(CreateLabel{LabelID: uuid.New(), UserID: testUserID, Name: "Urgent", Colour: "#ff0000"}, testHLC)
		if err != nil {
			t.Fatalf("setup create: %v", err)
		}
		agg.Apply(events[0])
		return agg
	}

	tests := []struct {
		name     string
		agg      func() *LabelAggregate
		update   func(agg *LabelAggregate) ([]eventstore.Event, error)
		wantType eventstore.EventType
		wantErr  error
	}{
		{
			name:     "rename active label",
			agg:      newActive,
			update:   func(a *LabelAggregate) ([]eventstore.Event, error) { return a.HandleUpdateName(UpdateLabelName{Name: "Important"}, testHLC) },
			wantType: eventstore.EventLabelNameUpdated,
		},
		{
			name:    "rename to empty name",
			agg:     newActive,
			update:  func(a *LabelAggregate) ([]eventstore.Event, error) { return a.HandleUpdateName(UpdateLabelName{Name: ""}, testHLC) },
			wantErr: ErrEmptyName,
		},
		{
			name:    "rename label that was never created",
			agg:     func() *LabelAggregate { return NewLabelAggregate() },
			update:  func(a *LabelAggregate) ([]eventstore.Event, error) { return a.HandleUpdateName(UpdateLabelName{Name: "Important"}, testHLC) },
			wantErr: ErrLabelNotFound,
		},
		{
			name: "recolour deleted label",
			agg: func() *LabelAggregate {
				agg := newActive()
				events, err := agg.HandleDelete(DeleteLabel{}, testHLC)
				if err != nil {
					t.Fatalf("setup delete: %v", err)
				}
				agg.Apply(events[0])
				return agg
			},
			update:  func(a *LabelAggregate) ([]eventstore.Event, error) { return a.HandleUpdateColour(UpdateLabelColour{Colour: "#00ff00"}, testHLC) },
			wantErr: ErrLabelAlreadyDeleted,
		},
		{
			name:     "recolour active label",
			agg:      newActive,
			update:   func(a *LabelAggregate) ([]eventstore.Event, error) { return a.HandleUpdateColour(UpdateLabelColour{Colour: "#00ff00"}, testHLC) },
			wantType: eventstore.EventLabelColourUpdated,
		},
		{
			name:    "recolour to empty colour",
			agg:     newActive,
			update:  func(a *LabelAggregate) ([]eventstore.Event, error) { return a.HandleUpdateColour(UpdateLabelColour{Colour: ""}, testHLC) },
			wantErr: ErrEmptyColour,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			events, err := tc.update(tc.agg())

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
