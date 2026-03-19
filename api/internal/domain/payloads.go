package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskCreatedPayload struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    int        `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	ListID      uuid.UUID  `json:"list_id"`
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

type TaskDescriptionUpdatedPayload struct {
	Description string `json:"description"`
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
