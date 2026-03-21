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

func TestLabelApply(t *testing.T) {
	agg := NewLabelAggregate()
	labelID := uuid.New()

	agg.Apply(eventstore.Event{
		AggregateID:   labelID,
		AggregateType: eventstore.AggregateTypeLabel,
		EventType:     eventstore.EventLabelCreated,
		UserID:        testUserID,
		Version:       1,
	})

	if agg.ID() != labelID {
		t.Errorf("ID = %v, want %v", agg.ID(), labelID)
	}
	if agg.Version() != 1 {
		t.Errorf("Version = %d, want 1", agg.Version())
	}
}
