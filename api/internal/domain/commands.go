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

type MoveTask struct {
	ListID   uuid.UUID
	Position string
}

type UpdateTaskDescription struct {
	Description string
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
