package domain

import (
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
	deleted bool
}

func NewListAggregate() *ListAggregate {
	return &ListAggregate{}
}

func (a *ListAggregate) ID() uuid.UUID      { return a.id }
func (a *ListAggregate) Version() int        { return a.version }
func (a *ListAggregate) UserID() uuid.UUID   { return a.userID }
func (a *ListAggregate) IsDeleted() bool     { return a.deleted }

func (a *ListAggregate) Apply(e eventstore.Event) {
	a.version = e.Version
	a.id = e.AggregateID
	a.userID = e.UserID

	switch e.EventType {
	case eventstore.EventListCreated:
		a.created = true
	case eventstore.EventListDeleted:
		a.deleted = true
	}
}

func (a *ListAggregate) HandleCreate(cmd CreateList, now time.Time) ([]eventstore.Event, error) {
	if a.created {
		return nil, ErrListAlreadyCreated
	}
	if cmd.Name == "" {
		return nil, ErrEmptyName
	}

	a.id = cmd.ListID
	a.userID = cmd.UserID

	e, err := a.newEvent(eventstore.EventListCreated, ListCreatedPayload{
		Name:     cmd.Name,
		Colour:   cmd.Colour,
		Icon:     cmd.Icon,
		Position: cmd.Position,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *ListAggregate) HandleDelete(cmd DeleteList) ([]eventstore.Event, error) {
	if !a.created {
		return nil, ErrListNotFound
	}
	if a.deleted {
		return nil, ErrListAlreadyDeleted
	}

	e, err := a.newEvent(eventstore.EventListDeleted, ListDeletedPayload{
		DeletedAt: cmd.DeletedAt,
	}, cmd.DeletedAt)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *ListAggregate) newEvent(eventType eventstore.EventType, payload any, now time.Time) (eventstore.Event, error) {
	return buildEvent(a.id, eventstore.AggregateTypeList, a.userID, &a.version, eventType, payload, now)
}
