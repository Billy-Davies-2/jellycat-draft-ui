package pubsub

import (
	"sync"
	"testing"
	"time"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
)

func init() {
	// Initialize logger for tests
	logger.Init()
}

func TestNewEmbeddedNATSPubSub(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	if ps.server == nil {
		t.Error("server should not be nil")
	}
	if ps.nc == nil {
		t.Error("NATS connection should not be nil")
	}
	if ps.js == nil {
		t.Error("JetStream context should not be nil")
	}
}

func TestEmbeddedNATSGetServerURL(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	url := ps.GetServerURL()
	if url == "" {
		t.Error("server URL should not be empty")
	}
	t.Logf("Embedded NATS URL: %s", url)
}

func TestEmbeddedNATSSubscribe(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	ch := ps.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe() returned nil channel")
	}

	if ps.GetSubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", ps.GetSubscriberCount())
	}
}

func TestEmbeddedNATSUnsubscribe(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	ch := ps.Subscribe()
	ps.Unsubscribe(ch)

	if ps.GetSubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", ps.GetSubscriberCount())
	}

	// Verify channel is closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after unsubscribe")
		}
	default:
		t.Error("channel should be closed and readable")
	}
}

func TestEmbeddedNATSPublishAndReceive(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	// Give the subscription goroutine time to start
	time.Sleep(100 * time.Millisecond)

	ch := ps.Subscribe()

	event := Event{
		Type:    "test:event",
		Payload: map[string]interface{}{"key": "value"},
	}

	ps.Publish(event)

	select {
	case received := <-ch:
		if received.Type != event.Type {
			t.Errorf("expected type %s, got %s", event.Type, received.Type)
		}
		if received.Payload["key"] != "value" {
			t.Error("payload mismatch")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestEmbeddedNATSMultipleSubscribers(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	// Give the subscription goroutine time to start
	time.Sleep(100 * time.Millisecond)

	ch1 := ps.Subscribe()
	ch2 := ps.Subscribe()
	ch3 := ps.Subscribe()

	if ps.GetSubscriberCount() != 3 {
		t.Errorf("expected 3 subscribers, got %d", ps.GetSubscriberCount())
	}

	event := Event{Type: "broadcast:test"}
	ps.Publish(event)

	for i, ch := range []chan Event{ch1, ch2, ch3} {
		select {
		case received := <-ch:
			if received.Type != "broadcast:test" {
				t.Errorf("subscriber %d: expected type broadcast:test, got %s", i, received.Type)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestEmbeddedNATSConcurrentPublish(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	// Give the subscription goroutine time to start
	time.Sleep(100 * time.Millisecond)

	ch := ps.Subscribe()

	var wg sync.WaitGroup
	numPublishers := 5
	eventsPerPublisher := 10

	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerPublisher; j++ {
				ps.Publish(Event{
					Type:    "concurrent:test",
					Payload: map[string]interface{}{"publisher": id, "seq": j},
				})
			}
		}(i)
	}

	// Collect events
	received := 0
	expectedTotal := numPublishers * eventsPerPublisher
	timeout := time.After(5 * time.Second)

	for received < expectedTotal {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Logf("Received %d/%d events before timeout", received, expectedTotal)
			goto done
		}
	}
done:

	wg.Wait()

	// We should receive all events (JetStream guarantees delivery)
	if received != expectedTotal {
		t.Errorf("expected %d events, received %d", expectedTotal, received)
	}
}

func TestEmbeddedNATSClose(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}

	ch := ps.Subscribe()

	// Close should not panic and should close the channel
	ps.Close()

	// Verify channel is closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after Close()")
		}
	default:
		t.Error("channel should be closed and readable")
	}
}

func TestEmbeddedNATSCustomOptions(t *testing.T) {
	opts := EmbeddedNATSOptions{
		Port:       0, // Random port
		Subject:    "custom.events",
		StreamName: "CUSTOM_STREAM",
		StoreDir:   "", // In-memory
	}

	ps, err := NewEmbeddedNATSPubSub(opts)
	if err != nil {
		t.Fatalf("Failed to create embedded NATS with custom options: %v", err)
	}
	defer ps.Close()

	if ps.subject != "custom.events" {
		t.Errorf("expected subject custom.events, got %s", ps.subject)
	}
}

func TestDefaultEmbeddedNATSOptions(t *testing.T) {
	opts := DefaultEmbeddedNATSOptions()

	if opts.Port != -1 {
		t.Errorf("expected port -1 (random), got %d", opts.Port)
	}
	if opts.Subject != "draft.events" {
		t.Errorf("expected subject draft.events, got %s", opts.Subject)
	}
	if opts.StreamName != "DRAFT_EVENTS" {
		t.Errorf("expected stream name DRAFT_EVENTS, got %s", opts.StreamName)
	}
	if opts.StoreDir != "" {
		t.Errorf("expected empty store dir, got %s", opts.StoreDir)
	}
}

func TestEmbeddedNATSEventPayload(t *testing.T) {
	ps, err := NewEmbeddedNATSPubSub(DefaultEmbeddedNATSOptions())
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}
	defer ps.Close()

	// Give the subscription goroutine time to start
	time.Sleep(100 * time.Millisecond)

	ch := ps.Subscribe()

	payload := map[string]interface{}{
		"string":  "value",
		"number":  42.0,
		"boolean": true,
		"nested": map[string]interface{}{
			"key": "nested_value",
		},
	}

	ps.Publish(Event{Type: "payload:test", Payload: payload})

	select {
	case received := <-ch:
		if received.Payload["string"] != "value" {
			t.Error("string payload mismatch")
		}
		if received.Payload["number"] != 42.0 {
			t.Error("number payload mismatch")
		}
		if received.Payload["boolean"] != true {
			t.Error("boolean payload mismatch")
		}
		nested, ok := received.Payload["nested"].(map[string]interface{})
		if !ok {
			t.Error("nested payload should be a map")
		} else if nested["key"] != "nested_value" {
			t.Error("nested payload mismatch")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for event")
	}
}
