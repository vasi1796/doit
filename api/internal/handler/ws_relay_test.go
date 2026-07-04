package handler

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/broker"
)

type fakeNotifier struct {
	ch chan uuid.UUID
}

func (f *fakeNotifier) Notify(userID uuid.UUID) {
	f.ch <- userID
}

type fakeSource struct {
	consume     func() (<-chan amqp.Delivery, error)
	reconnected func() <-chan struct{}
}

func (f *fakeSource) ConsumeBroadcast() (<-chan amqp.Delivery, error) { return f.consume() }
func (f *fakeSource) Reconnected() <-chan struct{}                    { return f.reconnected() }

func eventDelivery(t *testing.T, userID uuid.UUID) amqp.Delivery {
	t.Helper()
	body, err := json.Marshal(broker.EventMessage{
		EventID:       uuid.New(),
		AggregateID:   uuid.New(),
		AggregateType: "task",
		EventType:     "TaskCreated",
		UserID:        userID,
		Data:          json.RawMessage(`{}`),
		Timestamp:     time.Now(),
		Version:       1,
	})
	if err != nil {
		t.Fatalf("marshal event message: %v", err)
	}
	return amqp.Delivery{Body: body, RoutingKey: "TaskCreated"}
}

// collectNotifications reads exactly n user IDs from the notifier or fails.
func collectNotifications(t *testing.T, ch <-chan uuid.UUID, n int) []uuid.UUID {
	t.Helper()
	got := make([]uuid.UUID, 0, n)
	for len(got) < n {
		select {
		case id := <-ch:
			got = append(got, id)
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for notification %d of %d", len(got)+1, n)
		}
	}
	return got
}

func TestRelayNotify(t *testing.T) {
	userA := uuid.New()
	userB := uuid.New()

	tests := []struct {
		name       string
		deliveries func(t *testing.T) []amqp.Delivery
		want       []uuid.UUID
	}{
		{
			name: "valid event notifies its user",
			deliveries: func(t *testing.T) []amqp.Delivery {
				return []amqp.Delivery{eventDelivery(t, userA)}
			},
			want: []uuid.UUID{userA},
		},
		{
			name: "malformed message is skipped and loop continues",
			deliveries: func(t *testing.T) []amqp.Delivery {
				return []amqp.Delivery{
					{Body: []byte("not json"), RoutingKey: "TaskCreated"},
					eventDelivery(t, userA),
				}
			},
			want: []uuid.UUID{userA},
		},
		{
			name: "event without user ID is skipped and loop continues",
			deliveries: func(t *testing.T) []amqp.Delivery {
				return []amqp.Delivery{
					eventDelivery(t, uuid.Nil),
					eventDelivery(t, userA),
				}
			},
			want: []uuid.UUID{userA},
		},
		{
			name: "multiple users are routed independently in order",
			deliveries: func(t *testing.T) []amqp.Delivery {
				return []amqp.Delivery{
					eventDelivery(t, userA),
					eventDelivery(t, userB),
					eventDelivery(t, userA),
				}
			},
			want: []uuid.UUID{userA, userB, userA},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs := tc.deliveries(t)
			deliveries := make(chan amqp.Delivery, len(msgs))
			for _, d := range msgs {
				deliveries <- d
			}

			reconnected := make(chan struct{})
			source := &fakeSource{
				consume:     func() (<-chan amqp.Delivery, error) { return deliveries, nil },
				reconnected: func() <-chan struct{} { return reconnected },
			}
			notifier := &fakeNotifier{ch: make(chan uuid.UUID, len(msgs))}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go NewRelay(source, notifier, zerolog.Nop()).Run(ctx)

			got := collectNotifications(t, notifier.ch, len(tc.want))
			for i, want := range tc.want {
				if got[i] != want {
					t.Errorf("notification %d: got user %s, want %s", i, got[i], want)
				}
			}
		})
	}
}

func TestRelayResubscribe(t *testing.T) {
	user := uuid.New()

	tests := []struct {
		name string
		// trigger invalidates the first subscription (its deliveries channel
		// and its reconnected channel are passed in).
		trigger func(deliveries chan amqp.Delivery, reconnected chan struct{})
	}{
		{
			name: "re-subscribes on broker reconnect signal",
			trigger: func(_ chan amqp.Delivery, reconnected chan struct{}) {
				close(reconnected)
			},
		},
		{
			name: "re-subscribes when delivery channel closes",
			trigger: func(deliveries chan amqp.Delivery, _ chan struct{}) {
				close(deliveries)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			first := make(chan amqp.Delivery)
			second := make(chan amqp.Delivery, 1)
			second <- eventDelivery(t, user)
			firstReconnected := make(chan struct{})

			var mu sync.Mutex
			calls := 0
			source := &fakeSource{
				consume: func() (<-chan amqp.Delivery, error) {
					mu.Lock()
					defer mu.Unlock()
					calls++
					if calls == 1 {
						return first, nil
					}
					return second, nil
				},
				reconnected: func() <-chan struct{} {
					mu.Lock()
					defer mu.Unlock()
					if calls == 1 {
						return firstReconnected
					}
					// Fresh, open channel after re-subscribe — mirrors the
					// broker replacing its reconnected channel.
					return make(chan struct{})
				},
			}
			notifier := &fakeNotifier{ch: make(chan uuid.UUID, 1)}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go NewRelay(source, notifier, zerolog.Nop()).Run(ctx)

			tc.trigger(first, firstReconnected)

			got := collectNotifications(t, notifier.ch, 1)
			if got[0] != user {
				t.Errorf("got user %s, want %s", got[0], user)
			}
			mu.Lock()
			if calls != 2 {
				t.Errorf("ConsumeBroadcast called %d times, want 2", calls)
			}
			mu.Unlock()
		})
	}
}

func TestRelaySubscribeErrorRetries(t *testing.T) {
	user := uuid.New()

	deliveries := make(chan amqp.Delivery, 1)
	deliveries <- eventDelivery(t, user)

	var mu sync.Mutex
	calls := 0
	source := &fakeSource{
		consume: func() (<-chan amqp.Delivery, error) {
			mu.Lock()
			defer mu.Unlock()
			calls++
			if calls == 1 {
				return nil, context.DeadlineExceeded // any error
			}
			return deliveries, nil
		},
		reconnected: func() <-chan struct{} { return make(chan struct{}) },
	}
	notifier := &fakeNotifier{ch: make(chan uuid.UUID, 1)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go NewRelay(source, notifier, zerolog.Nop()).Run(ctx)

	got := collectNotifications(t, notifier.ch, 1)
	if got[0] != user {
		t.Errorf("got user %s, want %s", got[0], user)
	}
	mu.Lock()
	if calls != 2 {
		t.Errorf("ConsumeBroadcast called %d times, want 2", calls)
	}
	mu.Unlock()
}
