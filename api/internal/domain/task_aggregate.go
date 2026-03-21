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
	id             uuid.UUID
	userID         uuid.UUID
	version        int
	created        bool
	completed      bool
	deleted        bool
	labels         map[uuid.UUID]bool
	subtasks       map[uuid.UUID]*subtaskState
	recurrenceRule RecurrenceRule
	dueDate        *time.Time
	dueTime        *string
	title          string
	description    string
	priority       Priority
	listID         *uuid.UUID
	position       string
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
		var p TaskCreatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.title = p.Title
		a.description = p.Description
		a.priority = p.Priority
		a.dueDate = p.DueDate
		a.dueTime = p.DueTime
		a.listID = p.ListID
		a.position = p.Position
	case eventstore.EventTaskRecurrenceUpdated:
		var p TaskRecurrenceUpdatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.recurrenceRule = p.RecurrenceRule
	case eventstore.EventTaskCompleted:
		a.completed = true
	case eventstore.EventTaskUncompleted:
		a.completed = false
	case eventstore.EventTaskDeleted:
		a.deleted = true
	case eventstore.EventTaskRestored:
		a.deleted = false
	case eventstore.EventTaskTitleUpdated:
		var p TaskTitleUpdatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.title = p.Title
	case eventstore.EventTaskDescriptionUpdated:
		var p TaskDescriptionUpdatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.description = p.Description
	case eventstore.EventTaskPriorityUpdated:
		var p TaskPriorityUpdatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.priority = p.Priority
	case eventstore.EventTaskDueDateUpdated:
		var p TaskDueDateUpdatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.dueDate = p.DueDate
	case eventstore.EventTaskDueTimeUpdated:
		var p TaskDueTimeUpdatedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		a.dueTime = p.DueTime
	case eventstore.EventTaskMoved:
		var mp TaskMovedPayload
		_ = json.Unmarshal(e.Data, &mp) //nolint:errcheck
		listID := mp.ListID
		a.listID = &listID
		a.position = mp.Position
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
	case eventstore.EventSubtaskUncompleted:
		var p SubtaskUncompletedPayload
		_ = json.Unmarshal(e.Data, &p) //nolint:errcheck
		if st, ok := a.subtasks[p.SubtaskID]; ok {
			st.completed = false
		}
	case eventstore.EventSubtaskTitleUpdated:
		// Title is tracked only in the read model; no aggregate state to update.
	}
}

func (a *TaskAggregate) HandleCreate(cmd CreateTask, now time.Time) ([]eventstore.Event, error) {
	if a.created {
		return nil, ErrTaskAlreadyCreated
	}
	if cmd.Title == "" {
		return nil, ErrEmptyTitle
	}
	if !cmd.Priority.Valid() {
		return nil, ErrInvalidPriority
	}
	if invalidOptionalDueTime(cmd.DueTime) {
		return nil, ErrInvalidDueTime
	}

	a.id = cmd.TaskID
	a.userID = cmd.UserID

	e, err := a.newEvent(eventstore.EventTaskCreated, TaskCreatedPayload{
		Title:       cmd.Title,
		Description: cmd.Description,
		Priority:    cmd.Priority,
		DueDate:     cmd.DueDate,
		DueTime:     cmd.DueTime,
		ListID:      cmd.ListID,
		Position:    cmd.Position,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

// RecurringTaskEvents holds the events for a new recurring task occurrence.
// These must be appended separately since they belong to a different aggregate.
type RecurringTaskEvents struct {
	Events []eventstore.Event
}

func (a *TaskAggregate) HandleComplete(cmd CompleteTask) ([]eventstore.Event, *RecurringTaskEvents, error) {
	if err := a.requireActive(); err != nil {
		return nil, nil, err
	}
	if a.completed {
		return nil, nil, ErrTaskAlreadyCompleted
	}

	e, err := a.newEvent(eventstore.EventTaskCompleted, TaskCompletedPayload{
		CompletedAt: cmd.CompletedAt,
	}, cmd.CompletedAt)
	if err != nil {
		return nil, nil, err
	}

	events := []eventstore.Event{e}
	var recurring *RecurringTaskEvents

	// If the task has a recurrence rule and a due date, create the next occurrence.
	if a.recurrenceRule != "" && a.dueDate != nil {
		nextDue := nextDueDate(*a.dueDate, a.recurrenceRule)
		newAgg := NewTaskAggregate()
		newAgg.id = uuid.New()
		newAgg.userID = a.userID

		createEvents, err := newAgg.HandleCreate(CreateTask{
			TaskID:      newAgg.id,
			UserID:      a.userID,
			Title:       a.title,
			Description: a.description,
			Priority:    a.priority,
			DueDate:     &nextDue,
			DueTime:     a.dueTime,
			ListID:      a.listID,
			Position:    a.position,
		}, cmd.CompletedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("creating recurring task: %w", err)
		}

		// Apply the create event so the aggregate state is "created" for the next command
		for _, ev := range createEvents {
			newAgg.Apply(ev)
		}

		recEvents, err := newAgg.HandleUpdateRecurrence(UpdateTaskRecurrence{
			RecurrenceRule: a.recurrenceRule,
		}, cmd.CompletedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("setting recurrence on recurring task: %w", err)
		}

		recurring = &RecurringTaskEvents{
			Events: append(createEvents, recEvents...),
		}
	}

	return events, recurring, nil
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

func (a *TaskAggregate) HandleRestore(cmd RestoreTask, now time.Time) ([]eventstore.Event, error) {
	if !a.created {
		return nil, ErrTaskNotFound
	}
	if !a.deleted {
		return nil, ErrTaskNotDeleted
	}

	e, err := a.newEvent(eventstore.EventTaskRestored, TaskRestoredPayload{}, now)
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

func (a *TaskAggregate) HandleUpdateTitle(cmd UpdateTaskTitle, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if cmd.Title == "" {
		return nil, ErrEmptyTitle
	}

	e, err := a.newEvent(eventstore.EventTaskTitleUpdated, TaskTitleUpdatedPayload{
		Title: cmd.Title,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUpdatePriority(cmd UpdateTaskPriority, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if !cmd.Priority.Valid() {
		return nil, ErrInvalidPriority
	}

	e, err := a.newEvent(eventstore.EventTaskPriorityUpdated, TaskPriorityUpdatedPayload{
		Priority: cmd.Priority,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUpdateDueDate(cmd UpdateTaskDueDate, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}

	e, err := a.newEvent(eventstore.EventTaskDueDateUpdated, TaskDueDateUpdatedPayload{
		DueDate: cmd.DueDate,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUpdateDueTime(cmd UpdateTaskDueTime, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if invalidOptionalDueTime(cmd.DueTime) {
		return nil, ErrInvalidDueTime
	}

	e, err := a.newEvent(eventstore.EventTaskDueTimeUpdated, TaskDueTimeUpdatedPayload{
		DueTime: cmd.DueTime,
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

func (a *TaskAggregate) HandleUncompleteSubtask(cmd UncompleteSubtask, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	st, ok := a.subtasks[cmd.SubtaskID]
	if !ok {
		return nil, ErrSubtaskNotFound
	}
	if !st.completed {
		return nil, ErrSubtaskNotCompleted
	}

	e, err := a.newEvent(eventstore.EventSubtaskUncompleted, SubtaskUncompletedPayload{
		SubtaskID: cmd.SubtaskID,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUpdateSubtaskTitle(cmd UpdateSubtaskTitle, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if _, ok := a.subtasks[cmd.SubtaskID]; !ok {
		return nil, ErrSubtaskNotFound
	}
	if cmd.Title == "" {
		return nil, ErrEmptyTitle
	}

	e, err := a.newEvent(eventstore.EventSubtaskTitleUpdated, SubtaskTitleUpdatedPayload{
		SubtaskID: cmd.SubtaskID,
		Title:     cmd.Title,
	}, now)
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

func (a *TaskAggregate) HandleUpdateRecurrence(cmd UpdateTaskRecurrence, now time.Time) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if !cmd.RecurrenceRule.Valid() {
		return nil, ErrInvalidRecurrenceRule
	}

	e, err := a.newEvent(eventstore.EventTaskRecurrenceUpdated, TaskRecurrenceUpdatedPayload{
		RecurrenceRule: cmd.RecurrenceRule,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

// nextDueDate calculates the next due date by advancing current according to the rule.
func nextDueDate(current time.Time, rule RecurrenceRule) time.Time {
	switch rule {
	case RecurrenceDaily:
		return current.AddDate(0, 0, 1)
	case RecurrenceWeekly:
		return current.AddDate(0, 0, 7)
	case RecurrenceMonthly:
		return current.AddDate(0, 1, 0)
	case RecurrenceYearly:
		return current.AddDate(1, 0, 0)
	default:
		return current
	}
}

func (a *TaskAggregate) newEvent(eventType eventstore.EventType, payload any, now time.Time) (eventstore.Event, error) {
	return buildEvent(a.id, eventstore.AggregateTypeTask, a.userID, &a.version, eventType, payload, now)
}
