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
	Priority    Priority
	DueDate     *time.Time
	DueTime     *string
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

type ReorderTask struct {
	Position string
}

type UpdateTaskDescription struct {
	Description string
}

type UpdateTaskTitle struct {
	Title string
}

type UpdateTaskPriority struct {
	Priority Priority
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

type UncompleteSubtask struct {
	SubtaskID uuid.UUID
}

type UpdateTaskRecurrence struct {
	RecurrenceRule RecurrenceRule
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

type DeleteList struct {
	DeletedAt time.Time
}

// Label commands

type CreateLabel struct {
	LabelID uuid.UUID
	UserID  uuid.UUID
	Name    string
	Colour  string
}

type DeleteLabel struct {
	DeletedAt time.Time
}
