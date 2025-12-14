package pubsub

import (
	"sync"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
)

// Event represents a pubsub event
type Event struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// Upstream is an interface for upstream publishers (e.g., NATS)
type Upstream interface {
	Publish(Event)
	Subscribe() chan Event
	Unsubscribe(chan Event)
}

// PubSub implements a simple publish-subscribe system
type PubSub struct {
	mu          sync.RWMutex
	subscribers []chan Event
	upstream    Upstream // Optional upstream publisher (e.g., NATS)
}

// New creates a new PubSub instance
func New() *PubSub {
	return &PubSub{
		subscribers: []chan Event{},
	}
}

// NewWithUpstream creates a PubSub that bridges to an upstream publisher (e.g., NATS)
// When Publish is called, events are sent to the upstream, which broadcasts to all instances.
// Events from the upstream are forwarded to local subscribers.
func NewWithUpstream(upstream Upstream) *PubSub {
	ps := &PubSub{
		subscribers: []chan Event{},
		upstream:    upstream,
	}

	// Subscribe to upstream and forward events to local subscribers
	go func() {
		ch := upstream.Subscribe()
		logger.Debug("PubSub: Subscribed to upstream, waiting for events")
		for event := range ch {
			logger.Debug("PubSub: Received event from upstream, forwarding to local", "type", event.Type)
			ps.publishLocal(event)
		}
		logger.Debug("PubSub: Upstream channel closed")
	}()

	return ps
}

// Subscribe adds a new subscriber and returns a channel for receiving events
func (ps *PubSub) Subscribe() chan Event {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch := make(chan Event, 10)
	ps.subscribers = append(ps.subscribers, ch)
	logger.Debug("PubSub: New subscriber added", "totalSubscribers", len(ps.subscribers))
	return ch
}

// Unsubscribe removes a subscriber
func (ps *PubSub) Unsubscribe(ch chan Event) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for i, sub := range ps.subscribers {
		if sub == ch {
			close(ch)
			ps.subscribers = append(ps.subscribers[:i], ps.subscribers[i+1:]...)
			break
		}
	}
}

// Publish sends an event to all subscribers
// If an upstream is configured, the event is published to the upstream,
// which will broadcast it back to all instances (including this one)
func (ps *PubSub) Publish(event Event) {
	logger.Debug("PubSub: Publish called", "type", event.Type, "hasUpstream", ps.upstream != nil)
	if ps.upstream != nil {
		// Send to upstream; it will broadcast back to us via the subscription
		logger.Debug("PubSub: Forwarding to upstream", "type", event.Type)
		ps.upstream.Publish(event)
	} else {
		// No upstream, publish locally
		logger.Debug("PubSub: Publishing locally (no upstream)", "type", event.Type)
		ps.publishLocal(event)
	}
}

// publishLocal sends an event to local subscribers only
func (ps *PubSub) publishLocal(event Event) {
	ps.mu.RLock()
	subs := make([]chan Event, len(ps.subscribers))
	copy(subs, ps.subscribers)
	ps.mu.RUnlock()

	logger.Debug("PubSub: publishLocal", "type", event.Type, "subscriberCount", len(subs))

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Skip if channel is full
		}
	}
}
