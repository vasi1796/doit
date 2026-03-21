package domain

import (
	"time"

	"github.com/google/uuid"
)

// Task commands

type CreateTask struct {
	TaskID      uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description string
	Priority    int
	DueDate     *time.Time
	ListID      *uuid.UUID
	Position    string
}

type CompleteTask struct {
	CompletedAt time.Time
}

type UncompleteTask struct{}

type DeleteTask struct {
	DeletedAt time.Time
}

type RestoreTask struct{}

type MoveTask struct {
	ListID   uuid.UUID
	Position string
}

type UpdateTaskDescription struct {
	Description string
}

type UpdateTaskTitle struct {
	Title string
}

type UpdateTaskPriority struct {
	Priority int
}

type UpdateTaskDueDate struct {
	DueDate *time.Time
}

type AddLabel struct {
	LabelID uuid.UUID
}

type RemoveLabel struct {
	LabelID uuid.UUID
}

type CreateSubtask struct {
	SubtaskID uuid.UUID
	Title     string
	Position  string
}

type CompleteSubtask struct {
	SubtaskID   uuid.UUID
	CompletedAt time.Time
}

type UpdateSubtaskTitle struct {
	SubtaskID uuid.UUID
	Title     string
}

type UpdateTaskRecurrence struct {
	RecurrenceRule string // "daily", "weekly", "monthly", "yearly", or "" to clear
}

type UpdateTaskDueTime struct {
	DueTime *string
}

// List commands

type CreateList struct {
	ListID   uuid.UUID
	UserID   uuid.UUID
	Name     string
	Colour   string
	Icon     string
	Position string
}

// Label commands

type CreateLabel struct {
	LabelID uuid.UUID
	UserID  uuid.UUID
	Name    string
	Colour  string
}
