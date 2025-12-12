package pubsub

import (
	"encoding/json"
	"sync"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
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
	logger.Info("Using mock NATS pub/sub for local development", "subject", subject)
	logger.Debug("Note: Mock NATS does not require a real NATS server connection")

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
			logger.Warn("Mock NATS: Skipping slow subscriber", "event_type", event.Type)
		}
	}

	// Log in development mode
	data, _ := json.Marshal(event)
	logger.Debug("Mock NATS: Published event", "event_type", event.Type, "data", string(data))
}

// Subscribe creates a subscription channel for events
func (p *MockNATSPubSub) Subscribe() chan Event {
	ch := make(chan Event, 100)

	p.mu.Lock()
	p.subscribers = append(p.subscribers, ch)
	subCount := len(p.subscribers)
	p.mu.Unlock()

	logger.Debug("Mock NATS: New subscriber added", "total_subscribers", subCount)
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
			logger.Debug("Mock NATS: Subscriber removed", "remaining_subscribers", len(p.subscribers))
			break
		}
	}
}

// SubscribeJetStream simulates a durable JetStream subscription
// In the mock, this is the same as Subscribe but with a consumer name for logging
func (p *MockNATSPubSub) SubscribeJetStream(consumerName string, handler func(Event)) error {
	logger.Debug("Mock NATS: Creating durable subscription (simulated)", "consumer_name", consumerName)

	ch := p.Subscribe()

	// Start a goroutine to handle events
	go func() {
		for event := range ch {
			handler(event)
		}
		logger.Debug("Mock NATS: Durable subscription closed", "consumer_name", consumerName)
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

	logger.Debug("Mock NATS: Replaying messages", "count", len(p.messages[start:]))

	for _, event := range p.messages[start:] {
		select {
		case ch <- event:
		default:
			logger.Warn("Mock NATS: Channel full during replay, skipping event")
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

	logger.Info("Mock NATS: Closing all subscriptions", "active_subscriptions", len(p.subscribers))

	for _, sub := range p.subscribers {
		close(sub)
	}
	p.subscribers = nil
}
