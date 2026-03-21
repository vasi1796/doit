package domain

import "errors"

var (
	ErrEmptyTitle              = errors.New("domain: title cannot be empty")
	ErrEmptyName               = errors.New("domain: name cannot be empty")
	ErrInvalidPriority         = errors.New("domain: priority must be 0-3")
	ErrTaskAlreadyCreated      = errors.New("domain: task already exists")
	ErrTaskAlreadyCompleted    = errors.New("domain: task is already completed")
	ErrTaskNotCompleted        = errors.New("domain: task is not completed")
	ErrTaskAlreadyDeleted      = errors.New("domain: task is already deleted")
	ErrTaskNotFound            = errors.New("domain: task not found")
	ErrListNotFound            = errors.New("domain: list not found")
	ErrListAlreadyCreated      = errors.New("domain: list already exists")
	ErrListAlreadyDeleted      = errors.New("domain: list is already deleted")
	ErrLabelNotFound           = errors.New("domain: label not found")
	ErrLabelAlreadyCreated     = errors.New("domain: label already exists")
	ErrLabelAlreadyDeleted     = errors.New("domain: label is already deleted")
	ErrLabelAlreadyAttached    = errors.New("domain: label is already on this task")
	ErrLabelNotAttached        = errors.New("domain: label is not on this task")
	ErrSubtaskNotFound         = errors.New("domain: subtask not found")
	ErrSubtaskAlreadyCompleted = errors.New("domain: subtask is already completed")
	ErrSubtaskNotCompleted     = errors.New("domain: subtask is not completed")
	ErrTaskNotDeleted          = errors.New("domain: task is not deleted")
	ErrInvalidRecurrenceRule   = errors.New("domain: recurrence rule must be one of daily, weekly, monthly, yearly, or empty")
	ErrInvalidDueTime          = errors.New("domain: due_time must be in HH:MM format")
	ErrVersionConflict         = errors.New("domain: version conflict")
)
