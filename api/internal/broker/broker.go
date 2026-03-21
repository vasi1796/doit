// Package broker provides a RabbitMQ client for event publishing and consuming.
package broker

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	ExchangeName = "doit.events"
	QueueProjections = "doit.projections"
	QueueRecurring   = "doit.recurring"
	QueueDeadLetter  = "doit.dead-letter"
	DLXName          = "doit.dlx"
)

// Broker wraps an AMQP connection and channel.
type Broker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  zerolog.Logger
}

// New connects to RabbitMQ and opens a channel.
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
	return &Broker{conn: conn, channel: ch, logger: logger}, nil
}

// Setup declares the exchange, queues, and bindings.
func (b *Broker) Setup() error {
	// Dead-letter exchange + queue
	if err := b.channel.ExchangeDeclare(DLXName, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("broker: declare DLX: %w", err)
	}
	if _, err := b.channel.QueueDeclare(QueueDeadLetter, true, false, false, false, nil); err != nil {
		return fmt.Errorf("broker: declare DLQ: %w", err)
	}
	if err := b.channel.QueueBind(QueueDeadLetter, "#", DLXName, false, nil); err != nil {
		return fmt.Errorf("broker: bind DLQ: %w", err)
	}

	// Main topic exchange
	if err := b.channel.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("broker: declare exchange: %w", err)
	}

	dlArgs := amqp.Table{
		"x-dead-letter-exchange": DLXName,
	}

	// Projections queue — receives all events
	if _, err := b.channel.QueueDeclare(QueueProjections, true, false, false, false, dlArgs); err != nil {
		return fmt.Errorf("broker: declare projections queue: %w", err)
	}
	if err := b.channel.QueueBind(QueueProjections, "#", ExchangeName, false, nil); err != nil {
		return fmt.Errorf("broker: bind projections queue: %w", err)
	}

	// Recurring tasks queue — only task.completed events
	if _, err := b.channel.QueueDeclare(QueueRecurring, true, false, false, false, dlArgs); err != nil {
		return fmt.Errorf("broker: declare recurring queue: %w", err)
	}
	if err := b.channel.QueueBind(QueueRecurring, "TaskCompleted", ExchangeName, false, nil); err != nil {
		return fmt.Errorf("broker: bind recurring queue: %w", err)
	}

	return nil
}

// Publish sends a message to the exchange with the given routing key.
func (b *Broker) Publish(routingKey string, body []byte) error {
	return b.channel.Publish(ExchangeName, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

// Consume returns a delivery channel for the given queue.
func (b *Broker) Consume(queue string) (<-chan amqp.Delivery, error) {
	return b.channel.Consume(queue, "", false, false, false, false, nil)
}

// Close shuts down the channel and connection.
func (b *Broker) Close() error {
	if err := b.channel.Close(); err != nil {
		return err
	}
	return b.conn.Close()
}
