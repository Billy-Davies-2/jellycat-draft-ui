package pubsub

import (
	"encoding/json"
	"log"
	"sync"
)

// MockNATSPubSub implements a mock NATS JetStream pub/sub for local development
// This provides the same interface as NATSPubSub but doesn't require an actual NATS server
type MockNATSPubSub struct {
	subject     string
	subscribers []chan Event
	mu          sync.RWMutex
	messages    []Event // Store messages for replay (simulates JetStream storage)
	maxMessages int
}

// NewMockNATSPubSub creates a new mock NATS JetStream pub/sub for local development
func NewMockNATSPubSub(natsURL, subject string) (*MockNATSPubSub, error) {
	log.Printf("Using mock NATS pub/sub for local development (subject: %s)", subject)
	log.Printf("Note: Mock NATS does not require a real NATS server connection")
	
	return &MockNATSPubSub{
		subject:     subject,
		subscribers: make([]chan Event, 0),
		messages:    make([]Event, 0),
		maxMessages: 1000, // Keep last 1000 messages
	}, nil
}

// Publish publishes an event to the mock NATS (stores in memory)
func (p *MockNATSPubSub) Publish(event Event) {
	// Store message for potential replay
	p.mu.Lock()
	p.messages = append(p.messages, event)
	
	// Keep only the last maxMessages
	if len(p.messages) > p.maxMessages {
		p.messages = p.messages[len(p.messages)-p.maxMessages:]
	}
	
	// Make a copy of subscribers to avoid holding lock during delivery
	subs := make([]chan Event, len(p.subscribers))
	copy(subs, p.subscribers)
	p.mu.Unlock()

	// Deliver to all subscribers
	for _, sub := range subs {
		select {
		case sub <- event:
		default:
			// Subscriber is slow or blocked, skip
			log.Printf("Mock NATS: Skipping slow subscriber for event type: %s", event.Type)
		}
	}

	// Log in development mode
	data, _ := json.Marshal(event)
	log.Printf("Mock NATS: Published event [%s]: %s", event.Type, string(data))
}

// Subscribe creates a subscription channel for events
func (p *MockNATSPubSub) Subscribe() chan Event {
	ch := make(chan Event, 100)

	p.mu.Lock()
	p.subscribers = append(p.subscribers, ch)
	subCount := len(p.subscribers)
	p.mu.Unlock()

	log.Printf("Mock NATS: New subscriber added (total: %d)", subCount)
	return ch
}

// Unsubscribe removes a subscription channel
func (p *MockNATSPubSub) Unsubscribe(ch chan Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, sub := range p.subscribers {
		if sub == ch {
			// Remove from slice
			p.subscribers = append(p.subscribers[:i], p.subscribers[i+1:]...)
			close(ch)
			log.Printf("Mock NATS: Subscriber removed (remaining: %d)", len(p.subscribers))
			break
		}
	}
}

// SubscribeJetStream simulates a durable JetStream subscription
// In the mock, this is the same as Subscribe but with a consumer name for logging
func (p *MockNATSPubSub) SubscribeJetStream(consumerName string, handler func(Event)) error {
	log.Printf("Mock NATS: Creating durable subscription '%s' (simulated)", consumerName)
	
	ch := p.Subscribe()
	
	// Start a goroutine to handle events
	go func() {
		for event := range ch {
			handler(event)
		}
		log.Printf("Mock NATS: Durable subscription '%s' closed", consumerName)
	}()
	
	return nil
}

// ReplayMessages simulates JetStream message replay by sending stored messages
func (p *MockNATSPubSub) ReplayMessages(ch chan Event, count int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	start := len(p.messages) - count
	if start < 0 {
		start = 0
	}

	log.Printf("Mock NATS: Replaying %d messages", len(p.messages[start:]))
	
	for _, event := range p.messages[start:] {
		select {
		case ch <- event:
		default:
			log.Printf("Mock NATS: Channel full during replay, skipping event")
		}
	}
}

// GetMessageCount returns the number of stored messages
func (p *MockNATSPubSub) GetMessageCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.messages)
}

// GetSubscriberCount returns the number of active subscribers
func (p *MockNATSPubSub) GetSubscriberCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subscribers)
}

// Close closes all subscriptions
func (p *MockNATSPubSub) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("Mock NATS: Closing all subscriptions (%d active)", len(p.subscribers))
	
	for _, sub := range p.subscribers {
		close(sub)
	}
	p.subscribers = nil
}
