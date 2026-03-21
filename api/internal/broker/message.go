package broker

import (
	"encoding/json"

	"github.com/google/uuid"
)

// EventMessage is the JSON payload published to RabbitMQ.
type EventMessage struct {
	EventID       uuid.UUID       `json:"event_id"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	EventType     string          `json:"event_type"`
	UserID        uuid.UUID       `json:"user_id"`
	Data          json.RawMessage `json:"data"`
}
