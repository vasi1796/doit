// Package broker provides a RabbitMQ client for event publishing and consuming.
package broker

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	ExchangeName     = "doit.events"
	QueueProjections = "doit.projections"
	QueueRecurring   = "doit.recurring"
	QueueDeadLetter  = "doit.dead-letter"
	DLXName          = "doit.dlx"

	reconnectBaseDelay = 1 * time.Second
	reconnectMaxDelay  = 30 * time.Second
)

// Broker wraps an AMQP connection and channel with automatic reconnection.
type Broker struct {
	url     string
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  zerolog.Logger
	mu      sync.RWMutex

	// closed is closed when Close() is called to stop the reconnect loop.
	closed chan struct{}
	// reconnected is closed and replaced each time a reconnection succeeds.
	// Consumers can watch this to know when to re-subscribe.
	reconnected chan struct{}
}

// New connects to RabbitMQ and opens a channel. It starts a background
// goroutine that watches for connection loss and reconnects automatically.
func New(url string, logger zerolog.Logger) (*Broker, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("broker: dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("broker: open channel: %w", err)
	}
	b := &Broker{
		url:         url,
		conn:        conn,
		channel:     ch,
		logger:      logger,
		closed:      make(chan struct{}),
		reconnected: make(chan struct{}),
	}
	go b.watchConnection()
	return b, nil
}

// watchConnection waits for the AMQP connection to close, then reconnects
// with exponential backoff. After reconnecting it re-runs Setup() and signals
// consumers via the reconnected channel.
func (b *Broker) watchConnection() {
	for {
		b.mu.RLock()
		connCloseCh := b.conn.NotifyClose(make(chan *amqp.Error, 1))
		b.mu.RUnlock()

		select {
		case <-b.closed:
			return
		case amqpErr, ok := <-connCloseCh:
			if !ok {
				// Channel closed without error — connection was shut down cleanly.
				// Check if we're shutting down.
				select {
				case <-b.closed:
					return
				default:
				}
			}
			if amqpErr != nil {
				b.logger.Error().Int("code", amqpErr.Code).Str("reason", amqpErr.Reason).Msg("AMQP connection lost")
			} else {
				b.logger.Warn().Msg("AMQP connection closed")
			}
		}

		b.reconnect()
	}
}

// reconnect attempts to re-establish the AMQP connection and channel with
// exponential backoff and jitter. It blocks until successful or Close() is called.
func (b *Broker) reconnect() {
	attempt := 0
	for {
		select {
		case <-b.closed:
			return
		default:
		}

		delay := backoffDelay(attempt)
		b.logger.Info().Dur("delay", delay).Int("attempt", attempt+1).Msg("reconnecting to RabbitMQ")

		select {
		case <-b.closed:
			return
		case <-time.After(delay):
		}

		conn, err := amqp.Dial(b.url)
		if err != nil {
			b.logger.Error().Err(err).Msg("reconnect dial failed")
			attempt++
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			b.logger.Error().Err(err).Msg("reconnect open channel failed")
			conn.Close()
			attempt++
			continue
		}

		b.mu.Lock()
		b.conn = conn
		b.channel = ch
		// Signal consumers that reconnection happened. Close the old channel
		// and create a fresh one for the next reconnection cycle.
		oldReconnected := b.reconnected
		b.reconnected = make(chan struct{})
		b.mu.Unlock()

		// Re-declare exchanges, queues, and bindings on the new channel.
		if err := b.Setup(); err != nil {
			b.logger.Error().Err(err).Msg("reconnect setup failed, retrying")
			attempt++
			continue
		}

		b.logger.Info().Msg("reconnected to RabbitMQ successfully")
		close(oldReconnected)
		return
	}
}

// backoffDelay calculates exponential backoff with jitter.
func backoffDelay(attempt int) time.Duration {
	delay := float64(reconnectBaseDelay) * math.Pow(2, float64(attempt))
	if delay > float64(reconnectMaxDelay) {
		delay = float64(reconnectMaxDelay)
	}
	// Add jitter: 75%-125% of the calculated delay.
	jitter := 0.75 + rand.Float64()*0.5
	return time.Duration(delay * jitter)
}

// Reconnected returns a channel that is closed when a reconnection completes.
// After receiving the signal, callers should call Reconnected() again to get
// a fresh channel for the next reconnection cycle.
func (b *Broker) Reconnected() <-chan struct{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.reconnected
}

// Setup declares the exchange, queues, and bindings.
func (b *Broker) Setup() error {
	b.mu.RLock()
	ch := b.channel
	b.mu.RUnlock()

	// Dead-letter exchange + queue
	if err := ch.ExchangeDeclare(DLXName, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("broker: declare DLX: %w", err)
	}
	if _, err := ch.QueueDeclare(QueueDeadLetter, true, false, false, false, nil); err != nil {
		return fmt.Errorf("broker: declare DLQ: %w", err)
	}
	if err := ch.QueueBind(QueueDeadLetter, "#", DLXName, false, nil); err != nil {
		return fmt.Errorf("broker: bind DLQ: %w", err)
	}

	// Main topic exchange
	if err := ch.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("broker: declare exchange: %w", err)
	}

	dlArgs := amqp.Table{
		"x-dead-letter-exchange": DLXName,
	}

	// Projections queue — receives all events
	if _, err := ch.QueueDeclare(QueueProjections, true, false, false, false, dlArgs); err != nil {
		return fmt.Errorf("broker: declare projections queue: %w", err)
	}
	if err := ch.QueueBind(QueueProjections, "#", ExchangeName, false, nil); err != nil {
		return fmt.Errorf("broker: bind projections queue: %w", err)
	}

	// Recurring tasks queue — only task.completed events
	if _, err := ch.QueueDeclare(QueueRecurring, true, false, false, false, dlArgs); err != nil {
		return fmt.Errorf("broker: declare recurring queue: %w", err)
	}
	if err := ch.QueueBind(QueueRecurring, "TaskCompleted", ExchangeName, false, nil); err != nil {
		return fmt.Errorf("broker: bind recurring queue: %w", err)
	}

	return nil
}

// Publish sends a message to the exchange with the given routing key.
// It acquires a read lock to safely access the channel during reconnection.
func (b *Broker) Publish(routingKey string, body []byte) error {
	b.mu.RLock()
	ch := b.channel
	b.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("broker: channel not available (reconnecting)")
	}

	return ch.Publish(ExchangeName, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

// Consume returns a delivery channel for the given queue.
// It acquires a read lock to safely access the channel during reconnection.
func (b *Broker) Consume(queue string) (<-chan amqp.Delivery, error) {
	b.mu.RLock()
	ch := b.channel
	b.mu.RUnlock()

	if ch == nil {
		return nil, fmt.Errorf("broker: channel not available (reconnecting)")
	}

	return ch.Consume(queue, "", false, false, false, false, nil)
}

// PurgeQueue removes all messages from a queue. Useful for test isolation.
func (b *Broker) PurgeQueue(queue string) error {
	b.mu.RLock()
	ch := b.channel
	b.mu.RUnlock()

	_, err := ch.QueuePurge(queue, false)
	if err != nil {
		return fmt.Errorf("broker: purge %s: %w", queue, err)
	}
	return nil
}

// Get retrieves a single message from a queue (basic.get). Returns the message,
// whether a message was available, and any error. Useful for synchronous test
// consumption where the streaming Consume API is overkill.
func (b *Broker) Get(queue string) (amqp.Delivery, bool, error) {
	b.mu.RLock()
	ch := b.channel
	b.mu.RUnlock()

	msg, ok, err := ch.Get(queue, false)
	if err != nil {
		return amqp.Delivery{}, false, fmt.Errorf("broker: get from %s: %w", queue, err)
	}
	return msg, ok, nil
}

// Close shuts down the reconnect watcher, then closes the channel and connection.
func (b *Broker) Close() error {
	close(b.closed)

	b.mu.RLock()
	ch := b.channel
	conn := b.conn
	b.mu.RUnlock()

	var firstErr error
	if ch != nil {
		if err := ch.Close(); err != nil {
			firstErr = err
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
