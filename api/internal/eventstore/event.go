package eventstore

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AggregateType represents the type of aggregate an event belongs to.
type AggregateType string

const (
	AggregateTypeTask  AggregateType = "task"
	AggregateTypeList  AggregateType = "list"
	AggregateTypeLabel AggregateType = "label"
)

// EventType represents the type of domain event.
type EventType string

const (
	EventTaskCreated            EventType = "TaskCreated"
	EventTaskCompleted          EventType = "TaskCompleted"
	EventTaskUncompleted        EventType = "TaskUncompleted"
	EventTaskDeleted            EventType = "TaskDeleted"
	EventTaskMoved              EventType = "TaskMoved"
	EventTaskDescriptionUpdated EventType = "TaskDescriptionUpdated"
	EventLabelAdded             EventType = "LabelAdded"
	EventLabelRemoved           EventType = "LabelRemoved"
	EventListCreated            EventType = "ListCreated"
	EventLabelCreated           EventType = "LabelCreated"
	EventSubtaskCreated         EventType = "SubtaskCreated"
	EventSubtaskCompleted       EventType = "SubtaskCompleted"
)

// Event represents a single domain event stored in the event store.
// The Data field is a raw JSON message, keeping the store agnostic
// to event payload structure.
type Event struct {
	ID            uuid.UUID       `json:"id"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	AggregateType AggregateType   `json:"aggregate_type"`
	EventType     EventType       `json:"event_type"`
	UserID        uuid.UUID       `json:"user_id"`
	Data          json.RawMessage `json:"data"`
	Timestamp     time.Time       `json:"timestamp"`
	Version       int             `json:"version"`
}
