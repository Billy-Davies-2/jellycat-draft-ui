package pubsub

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ps := New()
	if ps == nil {
		t.Fatal("New() returned nil")
	}
	if ps.subscribers == nil {
		t.Error("subscribers slice should be initialized")
	}
	if ps.upstream != nil {
		t.Error("upstream should be nil for basic PubSub")
	}
}

func TestSubscribe(t *testing.T) {
	ps := New()

	ch := ps.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe() returned nil channel")
	}

	// Verify subscriber was added
	ps.mu.RLock()
	if len(ps.subscribers) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(ps.subscribers))
	}
	ps.mu.RUnlock()
}

func TestSubscribeMultiple(t *testing.T) {
	ps := New()

	ch1 := ps.Subscribe()
	ch2 := ps.Subscribe()
	ch3 := ps.Subscribe()

	if ch1 == nil || ch2 == nil || ch3 == nil {
		t.Fatal("Subscribe() returned nil channel")
	}

	ps.mu.RLock()
	if len(ps.subscribers) != 3 {
		t.Errorf("expected 3 subscribers, got %d", len(ps.subscribers))
	}
	ps.mu.RUnlock()
}

func TestUnsubscribe(t *testing.T) {
	ps := New()

	ch := ps.Subscribe()
	ps.Unsubscribe(ch)

	ps.mu.RLock()
	if len(ps.subscribers) != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", len(ps.subscribers))
	}
	ps.mu.RUnlock()

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

func TestUnsubscribeMiddle(t *testing.T) {
	ps := New()

	ch1 := ps.Subscribe()
	ch2 := ps.Subscribe()
	ch3 := ps.Subscribe()

	// Unsubscribe the middle one
	ps.Unsubscribe(ch2)

	ps.mu.RLock()
	if len(ps.subscribers) != 2 {
		t.Errorf("expected 2 subscribers, got %d", len(ps.subscribers))
	}
	ps.mu.RUnlock()

	// ch1 and ch3 should still work
	ps.Publish(Event{Type: "test"})

	select {
	case <-ch1:
		// ok
	case <-time.After(100 * time.Millisecond):
		t.Error("ch1 should have received event")
	}

	select {
	case <-ch3:
		// ok
	case <-time.After(100 * time.Millisecond):
		t.Error("ch3 should have received event")
	}
}

func TestPublishNoSubscribers(t *testing.T) {
	ps := New()

	// Should not panic
	ps.Publish(Event{Type: "test"})
}

func TestPublishSingleSubscriber(t *testing.T) {
	ps := New()
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
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestPublishMultipleSubscribers(t *testing.T) {
	ps := New()
	ch1 := ps.Subscribe()
	ch2 := ps.Subscribe()
	ch3 := ps.Subscribe()

	event := Event{Type: "broadcast"}
	ps.Publish(event)

	for i, ch := range []chan Event{ch1, ch2, ch3} {
		select {
		case received := <-ch:
			if received.Type != "broadcast" {
				t.Errorf("subscriber %d: expected type broadcast, got %s", i, received.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestPublishDropsWhenChannelFull(t *testing.T) {
	ps := New()
	ch := ps.Subscribe()

	// Fill up the channel (buffer size is 10)
	for i := 0; i < 15; i++ {
		ps.Publish(Event{Type: "fill"})
	}

	// Should have received 10 events (buffer size)
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 10 {
		t.Errorf("expected 10 events (buffer size), got %d", count)
	}
}

func TestConcurrentPublish(t *testing.T) {
	ps := New()
	ch := ps.Subscribe()

	var wg sync.WaitGroup
	numPublishers := 10
	eventsPerPublisher := 100

	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerPublisher; j++ {
				ps.Publish(Event{Type: "concurrent"})
			}
		}(i)
	}

	// Collect events in another goroutine
	received := 0
	done := make(chan struct{})
	go func() {
		for range ch {
			received++
			if received >= numPublishers*eventsPerPublisher {
				break
			}
		}
		close(done)
	}()

	wg.Wait()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		// Some events may have been dropped due to buffer full, that's ok
	}

	// We should have received some events (may not be all due to buffer overflow)
	if received == 0 {
		t.Error("expected to receive some events")
	}
}

func TestConcurrentSubscribeUnsubscribe(t *testing.T) {
	ps := New()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := ps.Subscribe()
			// Small delay
			time.Sleep(time.Millisecond)
			ps.Unsubscribe(ch)
		}()
	}

	// Also publish concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ps.Publish(Event{Type: "concurrent"})
		}()
	}

	wg.Wait()

	// Should not deadlock or panic
	ps.mu.RLock()
	subCount := len(ps.subscribers)
	ps.mu.RUnlock()

	if subCount != 0 {
		t.Errorf("expected 0 subscribers after all unsubscribe, got %d", subCount)
	}
}

// MockUpstream implements Upstream for testing
type MockUpstream struct {
	mu          sync.Mutex
	published   []Event
	subscribers []chan Event
}

func NewMockUpstream() *MockUpstream {
	return &MockUpstream{
		published:   []Event{},
		subscribers: []chan Event{},
	}
}

func (m *MockUpstream) Publish(event Event) {
	m.mu.Lock()
	m.published = append(m.published, event)
	subs := make([]chan Event, len(m.subscribers))
	copy(subs, m.subscribers)
	m.mu.Unlock()

	// Broadcast to all subscribers
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (m *MockUpstream) Subscribe() chan Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan Event, 100)
	m.subscribers = append(m.subscribers, ch)
	return ch
}

func (m *MockUpstream) Unsubscribe(ch chan Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, sub := range m.subscribers {
		if sub == ch {
			close(ch)
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			break
		}
	}
}

func (m *MockUpstream) PublishedEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Event, len(m.published))
	copy(result, m.published)
	return result
}

func TestNewWithUpstream(t *testing.T) {
	upstream := NewMockUpstream()
	ps := NewWithUpstream(upstream)

	if ps == nil {
		t.Fatal("NewWithUpstream() returned nil")
	}
	if ps.upstream != upstream {
		t.Error("upstream not set correctly")
	}
}

func TestPublishWithUpstream(t *testing.T) {
	upstream := NewMockUpstream()
	ps := NewWithUpstream(upstream)

	// Give the goroutine time to start
	time.Sleep(10 * time.Millisecond)

	ch := ps.Subscribe()

	event := Event{Type: "upstream:test", Payload: map[string]interface{}{"foo": "bar"}}
	ps.Publish(event)

	// Verify event was sent to upstream
	time.Sleep(10 * time.Millisecond)
	published := upstream.PublishedEvents()
	if len(published) != 1 {
		t.Errorf("expected 1 event published to upstream, got %d", len(published))
	}
	if len(published) > 0 && published[0].Type != "upstream:test" {
		t.Errorf("expected event type upstream:test, got %s", published[0].Type)
	}

	// Verify local subscriber received the event (via upstream broadcast back)
	select {
	case received := <-ch:
		if received.Type != "upstream:test" {
			t.Errorf("expected type upstream:test, got %s", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event from upstream")
	}
}

func TestUpstreamBroadcastToLocalSubscribers(t *testing.T) {
	upstream := NewMockUpstream()
	ps := NewWithUpstream(upstream)

	// Give the goroutine time to start
	time.Sleep(10 * time.Millisecond)

	ch1 := ps.Subscribe()
	ch2 := ps.Subscribe()

	// Publish directly to upstream (simulating another instance publishing)
	upstream.Publish(Event{Type: "external:event"})

	// Both local subscribers should receive the event
	for i, ch := range []chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.Type != "external:event" {
				t.Errorf("subscriber %d: expected type external:event, got %s", i, received.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestPublishLocalWhenNoUpstream(t *testing.T) {
	ps := New() // No upstream
	ch := ps.Subscribe()

	ps.Publish(Event{Type: "local:only"})

	select {
	case received := <-ch:
		if received.Type != "local:only" {
			t.Errorf("expected type local:only, got %s", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestEventPayload(t *testing.T) {
	ps := New()
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
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestUnsubscribeNonexistent(t *testing.T) {
	ps := New()

	// Create a channel that was never subscribed
	ch := make(chan Event, 10)

	// Should not panic
	ps.Unsubscribe(ch)

	// Channel should NOT be closed (since it wasn't managed by pubsub)
	select {
	case ch <- Event{Type: "test"}:
		// ok, channel is still open
	default:
		// This is also ok if buffer is full
	}
}
