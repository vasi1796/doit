package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

type subtaskState struct {
	id        uuid.UUID
	completed bool
}

// TaskAggregate enforces business rules for tasks and produces events.
// It is a pure domain object with no database dependency.
type TaskAggregate struct {
	id        uuid.UUID
	userID    uuid.UUID
	version   int
	created   bool
	completed bool
	deleted   bool
	labels    map[uuid.UUID]bool
	subtasks  map[uuid.UUID]*subtaskState
}

func NewTaskAggregate() *TaskAggregate {
	return &TaskAggregate{
		labels:   make(map[uuid.UUID]bool),
		subtasks: make(map[uuid.UUID]*subtaskState),
	}
}

// ID returns the aggregate's ID.
func (a *TaskAggregate) ID() uuid.UUID { return a.id }

// Version returns the current version (last applied event's version).
func (a *TaskAggregate) Version() int { return a.version }

// UserID returns the aggregate's owning user ID.
func (a *TaskAggregate) UserID() uuid.UUID { return a.userID }

// Apply replays a historical event to rebuild aggregate state.
// Events are trusted facts — no validation is performed.
func (a *TaskAggregate) Apply(e eventstore.Event) {
	a.version = e.Version
	a.id = e.AggregateID
	a.userID = e.UserID

	switch e.EventType {
	case eventstore.EventTaskCreated:
		a.created = true
	case eventstore.EventTaskCompleted:
		a.completed = true
	case eventstore.EventTaskUncompleted:
		a.completed = false
	case eventstore.EventTaskDeleted:
		a.deleted = true
	case eventstore.EventLabelAdded:
		var p LabelAddedPayload
		// Events from the store are trusted; unmarshal errors indicate a bug
		// in event production, not a recoverable runtime error.
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.labels[p.LabelID] = true
	case eventstore.EventLabelRemoved:
		var p LabelRemovedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		delete(a.labels, p.LabelID)
	case eventstore.EventSubtaskCreated:
		var p SubtaskCreatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.subtasks[p.SubtaskID] = &subtaskState{id: p.SubtaskID}
	case eventstore.EventSubtaskCompleted:
		var p SubtaskCompletedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		if st, ok := a.subtasks[p.SubtaskID]; ok {
			st.completed = true
		}
	}
}

func (a *TaskAggregate) HandleCreate(cmd CreateTask, now time.Time) ([]eventstore.Event, error) {
	if a.created {
		return nil, ErrTaskAlreadyCreated
	}
	if cmd.Title == "" {
		return nil, ErrEmptyTitle
	}
	if cmd.Priority < 0 || cmd.Priority > 3 {
		return nil, ErrInvalidPriority
	}

	a.id = cmd.TaskID
	a.userID = cmd.UserID

	e, err := a.newEvent(eventstore.EventTaskCreated, TaskCreatedPayload{
		Title:       cmd.Title,
		Description: cmd.Description,
		Priority:    cmd.Priority,
		DueDate:     cmd.DueDate,
		ListID:      cmd.ListID,
		Position:    cmd.Position,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleComplete(cmd CompleteTask) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if a.completed {
		return nil, ErrTaskAlreadyCompleted
	}

	e, err := a.newEvent(eventstore.EventTaskCompleted, TaskCompletedPayload{
		CompletedAt: cmd.CompletedAt,
	}, cmd.CompletedAt)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUncomplete(cmd UncompleteTask, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if !a.completed {
		return nil, ErrTaskNotCompleted
	}

	e, err := a.newEvent(eventstore.EventTaskUncompleted, TaskUncompletedPayload{}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleDelete(cmd DeleteTask) ([]eventstore.Event, error) {
	if !a.created {
		return nil, ErrTaskNotFound
	}
	if a.deleted {
		return nil, ErrTaskAlreadyDeleted
	}

	e, err := a.newEvent(eventstore.EventTaskDeleted, TaskDeletedPayload{
		DeletedAt: cmd.DeletedAt,
	}, cmd.DeletedAt)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleMove(cmd MoveTask, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}

	e, err := a.newEvent(eventstore.EventTaskMoved, TaskMovedPayload{
		ListID:   cmd.ListID,
		Position: cmd.Position,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUpdateDescription(cmd UpdateTaskDescription, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}

	e, err := a.newEvent(eventstore.EventTaskDescriptionUpdated, TaskDescriptionUpdatedPayload{
		Description: cmd.Description,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleAddLabel(cmd AddLabel, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if a.labels[cmd.LabelID] {
		return nil, ErrLabelAlreadyAttached
	}

	e, err := a.newEvent(eventstore.EventLabelAdded, LabelAddedPayload{
		LabelID: cmd.LabelID,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleRemoveLabel(cmd RemoveLabel, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if !a.labels[cmd.LabelID] {
		return nil, ErrLabelNotAttached
	}

	e, err := a.newEvent(eventstore.EventLabelRemoved, LabelRemovedPayload{
		LabelID: cmd.LabelID,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleCreateSubtask(cmd CreateSubtask, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if cmd.Title == "" {
		return nil, ErrEmptyTitle
	}

	e, err := a.newEvent(eventstore.EventSubtaskCreated, SubtaskCreatedPayload{
		SubtaskID: cmd.SubtaskID,
		Title:     cmd.Title,
		Position:  cmd.Position,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleCompleteSubtask(cmd CompleteSubtask) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	st, ok := a.subtasks[cmd.SubtaskID]
	if !ok {
		return nil, ErrSubtaskNotFound
	}
	if st.completed {
		return nil, ErrSubtaskAlreadyCompleted
	}

	e, err := a.newEvent(eventstore.EventSubtaskCompleted, SubtaskCompletedPayload{
		SubtaskID:   cmd.SubtaskID,
		CompletedAt: cmd.CompletedAt,
	}, cmd.CompletedAt)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

// requireActive checks that the task exists and is not deleted.
func (a *TaskAggregate) requireActive() error {
	if !a.created {
		return ErrTaskNotFound
	}
	if a.deleted {
		return ErrTaskAlreadyDeleted
	}
	return nil
}

// newEvent builds an event with the next version, marshaling the payload to JSON.
func (a *TaskAggregate) newEvent(eventType eventstore.EventType, payload any, now time.Time) (eventstore.Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventstore.Event{}, fmt.Errorf("marshaling event payload: %w", err)
	}

	a.version++
	return eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   a.id,
		AggregateType: eventstore.AggregateTypeTask,
		EventType:     eventType,
		UserID:        a.userID,
		Data:          data,
		Timestamp:     now,
		Version:       a.version,
	}, nil
}
