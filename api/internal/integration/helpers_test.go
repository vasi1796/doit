//go:build integration

// Package integration_test contains full-stack integration tests.
// Requires Postgres + RabbitMQ running, with NO worker processes
// consuming from the queues (tests consume messages directly).
package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/broker"
	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/hlc"
	"github.com/vasi1796/doit/internal/outbox"
	"github.com/vasi1796/doit/internal/projection"
)

// testHarness bundles all components needed for full-stack integration tests.
type testHarness struct {
	pool       *pgxpool.Pool
	store      *eventstore.Store
	cmdHandler *domain.CommandHandler
	projector  *projection.Projector
	broker     *broker.Broker
	poller     *outbox.Poller
	clock      *hlc.Clock
	logger     zerolog.Logger
	userID     uuid.UUID
}

func setupHarness(t *testing.T) *testHarness {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://doit:changeme@localhost:5432/doit?sslmode=disable"
	}
	rabbitURL := os.Getenv("TEST_RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://doit:changeme@localhost:5672/"
	}

	logger := zerolog.Nop()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	// Truncate all tables for test isolation
	tables := []string{
		"subtasks", "task_labels", "tasks", "labels", "lists",
		"aggregate_snapshots", "user_config", "outbox", "events", "users",
	}
	for _, table := range tables {
		if _, err := pool.Exec(ctx, "TRUNCATE "+table+" CASCADE"); err != nil {
			t.Fatalf("truncating %s: %v", table, err)
		}
	}

	// Insert a test user for FK constraints
	userID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, google_id, email, name, allowed) VALUES ($1, $2, $3, $4, true)`,
		userID, "google-"+userID.String(), userID.String()+"@test.com", "Test User",
	)
	if err != nil {
		t.Fatalf("inserting test user: %v", err)
	}

	store := eventstore.New(pool, logger)
	clock := hlc.New()
	cmdHandler := domain.NewCommandHandler(store, pool, clock)
	projector := projection.New(pool, logger)

	b, err := broker.New(rabbitURL, logger)
	if err != nil {
		t.Fatalf("connecting to RabbitMQ: %v", err)
	}
	t.Cleanup(func() { b.Close() })

	if err := b.Setup(); err != nil {
		t.Fatalf("setting up RabbitMQ: %v", err)
	}

	// Purge queues to isolate tests
	if err := b.PurgeQueue(broker.QueueProjections); err != nil {
		t.Fatalf("purging projections queue: %v", err)
	}
	if err := b.PurgeQueue(broker.QueueRecurring); err != nil {
		t.Fatalf("purging recurring queue: %v", err)
	}

	poller := outbox.NewPoller(pool, store, b, logger)

	return &testHarness{
		pool:       pool,
		store:      store,
		cmdHandler: cmdHandler,
		projector:  projector,
		broker:     b,
		poller:     poller,
		clock:      clock,
		logger:     logger,
		userID:     userID,
	}
}

// flushOutbox runs one poll cycle to publish outbox entries to RabbitMQ.
func (h *testHarness) flushOutbox(t *testing.T) {
	t.Helper()
	if err := h.poller.Poll(context.Background()); err != nil {
		t.Fatalf("flushing outbox: %v", err)
	}
}

// drainProjections consumes all pending messages from the projections queue
// and runs them through the projector. Returns the number of events processed.
func (h *testHarness) drainProjections(t *testing.T) int {
	t.Helper()
	return h.drainQueue(t, broker.QueueProjections, func(em broker.EventMessage) error {
		event := eventToStore(em)
		return h.projector.Project(context.Background(), []eventstore.Event{event})
	})
}

// drainRecurring consumes all pending messages from the recurring queue
// and runs the recurring task handler. Returns the number of events processed.
func (h *testHarness) drainRecurring(t *testing.T) int {
	t.Helper()
	return h.drainQueue(t, broker.QueueRecurring, func(em broker.EventMessage) error {
		if em.EventType != string(eventstore.EventTaskCompleted) {
			return nil
		}
		return handleRecurring(context.Background(), h.store, h.cmdHandler, em, h.logger)
	})
}

// drainQueue synchronously pulls messages from a queue using basic.get,
// calling handler for each. Stops when the queue is empty. Returns the count.
func (h *testHarness) drainQueue(t *testing.T, queue string, handler func(broker.EventMessage) error) int {
	t.Helper()

	count := 0
	for {
		msg, ok, err := h.broker.Get(queue)
		if err != nil {
			t.Fatalf("getting from %s: %v", queue, err)
		}
		if !ok {
			return count
		}

		var em broker.EventMessage
		if err := json.Unmarshal(msg.Body, &em); err != nil {
			t.Fatalf("unmarshal event message: %v", err)
		}
		if err := handler(em); err != nil {
			t.Fatalf("handling event %s: %v", em.EventType, err)
		}
		if err := msg.Ack(false); err != nil {
			t.Fatalf("ack failed: %v", err)
		}
		count++
	}
}

func eventToStore(em broker.EventMessage) eventstore.Event {
	return eventstore.Event{
		ID:            em.EventID,
		AggregateID:   em.AggregateID,
		AggregateType: eventstore.AggregateType(em.AggregateType),
		EventType:     eventstore.EventType(em.EventType),
		UserID:        em.UserID,
		Data:          em.Data,
		Timestamp:     em.Timestamp,
		Counter:       em.Counter,
		Version:       em.Version,
	}
}

// handleRecurring replicates the logic from cmd/worker-recurring/main.go.
func handleRecurring(ctx context.Context, store *eventstore.Store, cmdHandler *domain.CommandHandler, em broker.EventMessage, logger zerolog.Logger) error {
	events, err := store.LoadByAggregate(ctx, em.AggregateID)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	agg := domain.NewTaskAggregate()
	for _, e := range events {
		agg.Apply(e)
	}

	if agg.RecurrenceRule() == "" || agg.DueDate() == nil {
		return nil
	}

	nextDue := domain.NextDueDate(*agg.DueDate(), agg.RecurrenceRule())

	cmd := domain.CreateTask{
		TaskID:      domain.NewID(),
		UserID:      em.UserID,
		Title:       agg.Title(),
		Description: agg.Description(),
		Priority:    agg.Priority(),
		DueDate:     &nextDue,
		DueTime:     agg.DueTime(),
		ListID:      agg.ListID(),
		Position:    agg.Position(),
	}

	if err := cmdHandler.CreateTask(ctx, cmd); err != nil {
		return err
	}

	if err := cmdHandler.UpdateTaskRecurrence(ctx, cmd.TaskID, em.UserID, domain.UpdateTaskRecurrence{
		RecurrenceRule: agg.RecurrenceRule(),
	}); err != nil {
		logger.Warn().Err(err).Msg("failed to set recurrence on new task")
	}

	for _, labelID := range agg.Labels() {
		if err := cmdHandler.AddLabel(ctx, cmd.TaskID, em.UserID, domain.AddLabel{LabelID: labelID}); err != nil {
			logger.Warn().Err(err).Str("label_id", labelID.String()).Msg("failed to copy label to new task")
		}
	}

	return nil
}
