package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/broker"
)

const (
	relayRetryBase = 1 * time.Second
	relayRetryMax  = 10 * time.Second
)

// BroadcastSource is the subset of the message broker the relay consumes.
// Implemented by *broker.Broker.
type BroadcastSource interface {
	ConsumeBroadcast() (<-chan amqp.Delivery, error)
	Reconnected() <-chan struct{}
}

// Notifier receives a per-user signal that new events exist.
// Implemented by *Hub.
type Notifier interface {
	Notify(userID uuid.UUID)
}

// Relay consumes all events from the broker and pings each event's user over
// WebSocket. It carries no event payloads — clients pull state through the
// sync endpoint. Delivery is best-effort; the client's periodic poll is the
// convergence guarantee.
type Relay struct {
	source BroadcastSource
	hub    Notifier
	logger zerolog.Logger
}

func NewRelay(source BroadcastSource, hub Notifier, logger zerolog.Logger) *Relay {
	return &Relay{source: source, hub: hub, logger: logger}
}

// Run consumes broadcast deliveries until the context is cancelled,
// re-subscribing whenever the broker reconnects or the delivery channel
// closes. Malformed messages are logged and skipped — they must never stop
// the relay.
func (r *Relay) Run(ctx context.Context) {
	r.logger.Info().Msg("ws relay started")
	retryDelay := relayRetryBase

	for {
		deliveries, err := r.source.ConsumeBroadcast()
		if err != nil {
			r.logger.Warn().Err(err).Dur("retry_in", retryDelay).Msg("ws relay: subscribe failed")
			select {
			case <-ctx.Done():
				r.logger.Info().Msg("ws relay stopped")
				return
			case <-time.After(retryDelay):
			}
			retryDelay = min(retryDelay*2, relayRetryMax)
			continue
		}
		retryDelay = relayRetryBase

		if !r.consume(ctx, deliveries) {
			r.logger.Info().Msg("ws relay stopped")
			return
		}
		r.logger.Info().Msg("ws relay: re-subscribing after broker reconnect")
	}
}

// consume processes deliveries until the context is cancelled (returns false)
// or the subscription is invalidated by a reconnect or channel close
// (returns true, meaning: re-subscribe).
func (r *Relay) consume(ctx context.Context, deliveries <-chan amqp.Delivery) bool {
	reconnected := r.source.Reconnected()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-reconnected:
			return true
		case d, ok := <-deliveries:
			if !ok {
				return true
			}
			var msg broker.EventMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				r.logger.Warn().Err(err).Str("routing_key", d.RoutingKey).Msg("ws relay: malformed event message, skipping")
				continue
			}
			if msg.UserID == uuid.Nil {
				r.logger.Warn().Str("event_type", msg.EventType).Msg("ws relay: event without user ID, skipping")
				continue
			}
			r.hub.Notify(msg.UserID)
		}
	}
}
