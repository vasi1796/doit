package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

// LabelAggregate enforces business rules for labels and produces events.
type LabelAggregate struct {
	id      uuid.UUID
	userID  uuid.UUID
	version int
	created bool
}

func NewLabelAggregate() *LabelAggregate {
	return &LabelAggregate{}
}

func (a *LabelAggregate) ID() uuid.UUID { return a.id }
func (a *LabelAggregate) Version() int  { return a.version }

func (a *LabelAggregate) Apply(e eventstore.Event) {
	a.version = e.Version
	a.id = e.AggregateID
	a.userID = e.UserID

	switch e.EventType {
	case eventstore.EventLabelCreated:
		a.created = true
	}
}

func (a *LabelAggregate) HandleCreate(cmd CreateLabel, now time.Time) ([]eventstore.Event, error) {
	if a.created {
		return nil, ErrLabelAlreadyCreated
	}
	if cmd.Name == "" {
		return nil, ErrEmptyTitle
	}

	a.id = cmd.LabelID
	a.userID = cmd.UserID

	data, err := json.Marshal(LabelCreatedPayload{
		Name:   cmd.Name,
		Colour: cmd.Colour,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling event payload: %w", err)
	}

	a.version++
	e := eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   a.id,
		AggregateType: eventstore.AggregateTypeLabel,
		EventType:     eventstore.EventLabelCreated,
		UserID:        a.userID,
		Data:          data,
		Timestamp:     now,
		Version:       a.version,
	}
	return []eventstore.Event{e}, nil
}
