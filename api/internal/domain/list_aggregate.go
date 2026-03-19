package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

// ListAggregate enforces business rules for lists and produces events.
type ListAggregate struct {
	id      uuid.UUID
	userID  uuid.UUID
	version int
	created bool
}

func NewListAggregate() *ListAggregate {
	return &ListAggregate{}
}

func (a *ListAggregate) ID() uuid.UUID { return a.id }
func (a *ListAggregate) Version() int  { return a.version }

func (a *ListAggregate) Apply(e eventstore.Event) {
	a.version = e.Version
	a.id = e.AggregateID
	a.userID = e.UserID

	switch e.EventType {
	case eventstore.EventListCreated:
		a.created = true
	}
}

func (a *ListAggregate) HandleCreate(cmd CreateList, now time.Time) ([]eventstore.Event, error) {
	if a.created {
		return nil, ErrListAlreadyCreated
	}
	if cmd.Name == "" {
		return nil, ErrEmptyTitle
	}

	a.id = cmd.ListID
	a.userID = cmd.UserID

	data, err := json.Marshal(ListCreatedPayload{
		Name:     cmd.Name,
		Colour:   cmd.Colour,
		Icon:     cmd.Icon,
		Position: cmd.Position,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling event payload: %w", err)
	}

	a.version++
	e := eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   a.id,
		AggregateType: eventstore.AggregateTypeList,
		EventType:     eventstore.EventListCreated,
		UserID:        a.userID,
		Data:          data,
		Timestamp:     now,
		Version:       a.version,
	}
	return []eventstore.Event{e}, nil
}
