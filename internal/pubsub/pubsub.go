package pubsub

import (
	"sync"
)

// Event represents a pubsub event
type Event struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// PubSub implements a simple publish-subscribe system
type PubSub struct {
	mu          sync.RWMutex
	subscribers []chan Event
}

// New creates a new PubSub instance
func New() *PubSub {
	return &PubSub{
		subscribers: []chan Event{},
	}
}

// Subscribe adds a new subscriber and returns a channel for receiving events
func (ps *PubSub) Subscribe() chan Event {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch := make(chan Event, 10)
	ps.subscribers = append(ps.subscribers, ch)
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
func (ps *PubSub) Publish(event Event) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for _, ch := range ps.subscribers {
		select {
		case ch <- event:
		default:
			// Skip if channel is full
		}
	}
}
