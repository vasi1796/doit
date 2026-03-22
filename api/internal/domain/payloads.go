package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskCreatedPayload struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    Priority   `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	DueTime     *string    `json:"due_time,omitempty"`
	ListID      *uuid.UUID `json:"list_id,omitempty"`
	Position    string     `json:"position"`
}

type TaskCompletedPayload struct {
	CompletedAt time.Time `json:"completed_at"`
}

type TaskUncompletedPayload struct{}

type TaskDeletedPayload struct {
	DeletedAt time.Time `json:"deleted_at"`
}

type TaskMovedPayload struct {
	ListID   uuid.UUID `json:"list_id"`
	Position string    `json:"position"`
}

type TaskReorderedPayload struct {
	Position string `json:"position"`
}

type TaskDescriptionUpdatedPayload struct {
	Description string `json:"description"`
}

type TaskTitleUpdatedPayload struct {
	Title string `json:"title"`
}

type TaskPriorityUpdatedPayload struct {
	Priority Priority `json:"priority"`
}

type TaskDueDateUpdatedPayload struct {
	DueDate *time.Time `json:"due_date,omitempty"`
}

type LabelAddedPayload struct {
	LabelID uuid.UUID `json:"label_id"`
}

type LabelRemovedPayload struct {
	LabelID uuid.UUID `json:"label_id"`
}

type ListCreatedPayload struct {
	Name     string `json:"name"`
	Colour   string `json:"colour"`
	Icon     string `json:"icon,omitempty"`
	Position string `json:"position"`
}

type LabelCreatedPayload struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`
}

type SubtaskCreatedPayload struct {
	SubtaskID uuid.UUID `json:"subtask_id"`
	Title     string    `json:"title"`
	Position  string    `json:"position"`
}

type SubtaskCompletedPayload struct {
	SubtaskID   uuid.UUID `json:"subtask_id"`
	CompletedAt time.Time `json:"completed_at"`
}

type SubtaskTitleUpdatedPayload struct {
	SubtaskID uuid.UUID `json:"subtask_id"`
	Title     string    `json:"title"`
}

type SubtaskUncompletedPayload struct {
	SubtaskID uuid.UUID `json:"subtask_id"`
}

type TaskRestoredPayload struct{}

type TaskRecurrenceUpdatedPayload struct {
	RecurrenceRule RecurrenceRule `json:"recurrence_rule"`
}

type TaskDueTimeUpdatedPayload struct {
	DueTime *string `json:"due_time,omitempty"`
}

type ListDeletedPayload struct {
	DeletedAt time.Time `json:"deleted_at"`
}

type LabelDeletedPayload struct {
	DeletedAt time.Time `json:"deleted_at"`
}
