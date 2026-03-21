package domain

import (
	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
)

// LabelAggregate enforces business rules for labels and produces events.
type LabelAggregate struct {
	id      uuid.UUID
	userID  uuid.UUID
	version int
	created bool
	deleted bool
}

func NewLabelAggregate() *LabelAggregate {
	return &LabelAggregate{}
}

func (a *LabelAggregate) ID() uuid.UUID      { return a.id }
func (a *LabelAggregate) Version() int        { return a.version }
func (a *LabelAggregate) UserID() uuid.UUID   { return a.userID }
func (a *LabelAggregate) IsDeleted() bool     { return a.deleted }

func (a *LabelAggregate) Apply(e eventstore.Event) {
	a.version = e.Version
	a.id = e.AggregateID
	a.userID = e.UserID

	switch e.EventType {
	case eventstore.EventLabelCreated:
		a.created = true
	case eventstore.EventLabelDeleted:
		a.deleted = true
	}
}

func (a *LabelAggregate) HandleCreate(cmd CreateLabel, now hlc.Timestamp) ([]eventstore.Event, error) {
	if a.created {
		return nil, ErrLabelAlreadyCreated
	}
	if cmd.Name == "" {
		return nil, ErrEmptyName
	}

	a.id = cmd.LabelID
	a.userID = cmd.UserID

	e, err := a.newEvent(eventstore.EventLabelCreated, LabelCreatedPayload{
		Name:   cmd.Name,
		Colour: cmd.Colour,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *LabelAggregate) HandleDelete(cmd DeleteLabel, now hlc.Timestamp) ([]eventstore.Event, error) {
	if !a.created {
		return nil, ErrLabelNotFound
	}
	if a.deleted {
		return nil, ErrLabelAlreadyDeleted
	}

	e, err := a.newEvent(eventstore.EventLabelDeleted, LabelDeletedPayload{
		DeletedAt: cmd.DeletedAt,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *LabelAggregate) newEvent(eventType eventstore.EventType, payload any, now hlc.Timestamp) (eventstore.Event, error) {
	return buildEvent(a.id, eventstore.AggregateTypeLabel, a.userID, &a.version, eventType, payload, now)
}
