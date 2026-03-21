package broker

import (
	"encoding/json"
	"time"

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
	Timestamp     time.Time       `json:"timestamp"`
	Counter       int             `json:"counter"`
	Version       int             `json:"version"`
}
