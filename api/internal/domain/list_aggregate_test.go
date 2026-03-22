package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
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
