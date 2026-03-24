package domain

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
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

func (a *TaskAggregate) ID() uuid.UUID      { return a.id }
func (a *TaskAggregate) Version() int        { return a.version }
func (a *TaskAggregate) UserID() uuid.UUID   { return a.userID }

// Getters for recurring task worker
func (a *TaskAggregate) Title() string              { return a.title }
func (a *TaskAggregate) Description() string        { return a.description }
func (a *TaskAggregate) Priority() Priority         { return a.priority }
func (a *TaskAggregate) RecurrenceRule() RecurrenceRule { return a.recurrenceRule }
func (a *TaskAggregate) DueDate() *time.Time        { return a.dueDate }
func (a *TaskAggregate) DueTime() *string           { return a.dueTime }
func (a *TaskAggregate) ListID() *uuid.UUID         { return a.listID }
func (a *TaskAggregate) Position() string           { return a.position }
func (a *TaskAggregate) Labels() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(a.labels))
	for id := range a.labels {
		ids = append(ids, id)
	}
	return ids
}

func NewID() uuid.UUID { return uuid.New() }

// NextDueDate calculates the next due date based on a recurrence rule.
func NextDueDate(current time.Time, rule RecurrenceRule) time.Time {
	return nextDueDate(current, rule)
}

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
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.title = p.Title
		a.description = p.Description
		a.priority = p.Priority
		a.dueDate = p.DueDate
		a.dueTime = p.DueTime
		a.listID = p.ListID
		a.position = p.Position
	case eventstore.EventTaskRecurrenceUpdated:
		var p TaskRecurrenceUpdatedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
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
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.title = p.Title
	case eventstore.EventTaskDescriptionUpdated:
		var p TaskDescriptionUpdatedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.description = p.Description
	case eventstore.EventTaskPriorityUpdated:
		var p TaskPriorityUpdatedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.priority = p.Priority
	case eventstore.EventTaskDueDateUpdated:
		var p TaskDueDateUpdatedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.dueDate = p.DueDate
	case eventstore.EventTaskDueTimeUpdated:
		var p TaskDueTimeUpdatedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.dueTime = p.DueTime
	case eventstore.EventTaskMoved:
		var mp TaskMovedPayload
		if err := json.Unmarshal(e.Data, &mp); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		listID := mp.ListID
		a.listID = &listID
		a.position = mp.Position
	case eventstore.EventTaskReordered:
		var p TaskReorderedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.position = p.Position
	case eventstore.EventLabelAdded:
		var p LabelAddedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.labels[p.LabelID] = true
	case eventstore.EventLabelRemoved:
		var p LabelRemovedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		delete(a.labels, p.LabelID)
	case eventstore.EventSubtaskCreated:
		var p SubtaskCreatedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		a.subtasks[p.SubtaskID] = &subtaskState{id: p.SubtaskID}
	case eventstore.EventSubtaskCompleted:
		var p SubtaskCompletedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		if st, ok := a.subtasks[p.SubtaskID]; ok {
			st.completed = true
		}
	case eventstore.EventSubtaskUncompleted:
		var p SubtaskUncompletedPayload
		if err := json.Unmarshal(e.Data, &p); err != nil {
			log.Printf("warn: failed to unmarshal %s event payload: %v", e.EventType, err)
			return
		}
		if st, ok := a.subtasks[p.SubtaskID]; ok {
			st.completed = false
		}
	case eventstore.EventSubtaskTitleUpdated:
		// Title is tracked only in the read model; no aggregate state to update.
	}
}

func (a *TaskAggregate) HandleCreate(cmd CreateTask, now hlc.Timestamp) ([]eventstore.Event, error) {
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

// HandleComplete marks the task as completed.
// Recurring task creation is handled asynchronously by the recurring tasks worker.
func (a *TaskAggregate) HandleComplete(cmd CompleteTask, now hlc.Timestamp) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}
	if a.completed {
		return nil, ErrTaskAlreadyCompleted
	}

	e, err := a.newEvent(eventstore.EventTaskCompleted, TaskCompletedPayload{
		CompletedAt: cmd.CompletedAt,
	}, now)
	if err != nil {
		return nil, err
	}

	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUncomplete(cmd UncompleteTask, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleDelete(cmd DeleteTask, now hlc.Timestamp) ([]eventstore.Event, error) {
	if !a.created {
		return nil, ErrTaskNotFound
	}
	if a.deleted {
		return nil, ErrTaskAlreadyDeleted
	}

	e, err := a.newEvent(eventstore.EventTaskDeleted, TaskDeletedPayload{
		DeletedAt: cmd.DeletedAt,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleRestore(cmd RestoreTask, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleMove(cmd MoveTask, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleReorder(cmd ReorderTask, now hlc.Timestamp) ([]eventstore.Event, error) {
	if err := a.requireActive(); err != nil {
		return nil, err
	}

	e, err := a.newEvent(eventstore.EventTaskReordered, TaskReorderedPayload{
		Position: cmd.Position,
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUpdateDescription(cmd UpdateTaskDescription, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleUpdateTitle(cmd UpdateTaskTitle, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleUpdatePriority(cmd UpdateTaskPriority, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleUpdateDueDate(cmd UpdateTaskDueDate, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleUpdateDueTime(cmd UpdateTaskDueTime, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleAddLabel(cmd AddLabel, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleRemoveLabel(cmd RemoveLabel, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleCreateSubtask(cmd CreateSubtask, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleCompleteSubtask(cmd CompleteSubtask, now hlc.Timestamp) ([]eventstore.Event, error) {
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
	}, now)
	if err != nil {
		return nil, err
	}
	return []eventstore.Event{e}, nil
}

func (a *TaskAggregate) HandleUncompleteSubtask(cmd UncompleteSubtask, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleUpdateSubtaskTitle(cmd UpdateSubtaskTitle, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) HandleUpdateRecurrence(cmd UpdateTaskRecurrence, now hlc.Timestamp) ([]eventstore.Event, error) {
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

func (a *TaskAggregate) newEvent(eventType eventstore.EventType, payload any, now hlc.Timestamp) (eventstore.Event, error) {
	return buildEvent(a.id, eventstore.AggregateTypeTask, a.userID, &a.version, eventType, payload, now)
}
