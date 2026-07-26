package domain

import (
	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
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

func (a *ListAggregate) HandleCreate(cmd CreateList, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *ListAggregate) HandleDelete(cmd DeleteList, now hlc.Timestamp) ([]eventstore.Event, error) {
	if !a.created {
		return nil, ErrListNotFound
	}
	if a.deleted {
		return nil, ErrListAlreadyDeleted
	}

	e, err := a.newEvent(eventstore.EventListDeleted, ListDeletedPayload{
		DeletedAt: cmd.DeletedAt,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *ListAggregate) HandleUpdateName(cmd UpdateListName, now hlc.Timestamp) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if cmd.Name == "" {
		return nil, ErrEmptyName
	}

	e, err := a.newEvent(eventstore.EventListNameUpdated, ListNameUpdatedPayload{
		Name: cmd.Name,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *ListAggregate) HandleUpdateColour(cmd UpdateListColour, now hlc.Timestamp) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if cmd.Colour == "" {
		return nil, ErrEmptyColour
	}

	e, err := a.newEvent(eventstore.EventListColourUpdated, ListColourUpdatedPayload{
		Colour: cmd.Colour,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *ListAggregate) HandleReorder(cmd ReorderList, now hlc.Timestamp) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if cmd.Position == "" {
		return nil, ErrEmptyPosition
	}

	e, err := a.newEvent(eventstore.EventListReordered, ListReorderedPayload{
		Position: cmd.Position,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *ListAggregate) requireActive() error {
	if !a.created {
		return ErrListNotFound
	}
	if a.deleted {
		return ErrListAlreadyDeleted
	}
	return nil
}

func (a *ListAggregate) newEvent(eventType eventstore.EventType, payload any, now hlc.Timestamp) (eventstore.Event, error) {
	return buildEvent(a.id, eventstore.AggregateTypeList, a.userID, &a.version, eventType, payload, now)
}
