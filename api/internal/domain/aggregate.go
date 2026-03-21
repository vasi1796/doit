package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vasi1796/doit/internal/eventstore"
)

// buildEvent constructs a domain event, marshaling the payload to JSON and
// incrementing the version counter. Shared by all aggregate types.
func buildEvent(
	aggregateID uuid.UUID,
	aggregateType eventstore.AggregateType,
	userID uuid.UUID,
	version *int,
	eventType eventstore.EventType,
	payload any,
	now time.Time,
) (eventstore.Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventstore.Event{}, fmt.Errorf("marshaling event payload: %w", err)
	}

	*version++
	return eventstore.Event{
		ID:            uuid.New(),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     eventType,
		UserID:        userID,
		Data:          data,
		Timestamp:     now,
		Version:       *version,
	}, nil
}

// invalidOptionalDueTime returns true if the due time pointer is non-nil,
// non-empty, and not in valid HH:MM format.
func invalidOptionalDueTime(t *string) bool {
	return t != nil && *t != "" && !ValidDueTime(*t)
}
