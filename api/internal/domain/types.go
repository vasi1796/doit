package domain

import "regexp"

// RecurrenceRule represents a valid task recurrence frequency.
type RecurrenceRule string

const (
	RecurrenceNone    RecurrenceRule = ""
	RecurrenceDaily   RecurrenceRule = "daily"
	RecurrenceWeekly  RecurrenceRule = "weekly"
	RecurrenceMonthly RecurrenceRule = "monthly"
	RecurrenceYearly  RecurrenceRule = "yearly"
)

func (r RecurrenceRule) Valid() bool {
	switch r {
	case RecurrenceNone, RecurrenceDaily, RecurrenceWeekly, RecurrenceMonthly, RecurrenceYearly:
		return true
	}
	return false
}

// Priority represents a task priority level.
type Priority int

const (
	PriorityNone   Priority = 0
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

func (p Priority) Valid() bool {
	return p >= PriorityNone && p <= PriorityHigh
}

// dueTimeRegex matches HH:MM format (24-hour).
var dueTimeRegex = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// ValidDueTime checks that a due time string is in HH:MM format.
func ValidDueTime(t string) bool {
	return dueTimeRegex.MatchString(t)
}
