package pubsub

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

// NATSPubSub implements pub/sub using NATS JetStream
type NATSPubSub struct {
	nc          *nats.Conn
	js          nats.JetStreamContext
	subject     string
	subscribers []chan Event
	mu          sync.RWMutex
}

// NewNATSPubSub creates a new NATS JetStream pub/sub
func NewNATSPubSub(natsURL, subject string) (*NATSPubSub, error) {
	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or get stream
	streamName := "DRAFT_EVENTS"
	_, err = js.StreamInfo(streamName)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{subject},
			Storage:  nats.FileStorage,
			MaxAge:   0, // Keep events indefinitely for replay
		})
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("failed to create stream: %w", err)
		}
	}

	ps := &NATSPubSub{
		nc:          nc,
		js:          js,
		subject:     subject,
		subscribers: make([]chan Event, 0),
	}

	return ps, nil
}

// Publish publishes an event to NATS JetStream
func (p *NATSPubSub) Publish(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	// Publish to JetStream
	_, err = p.js.Publish(p.subject, data)
	if err != nil {
		log.Printf("Failed to publish to NATS: %v", err)
		return
	}

	// Also send to local subscribers for in-process delivery
	p.mu.RLock()
	subs := make([]chan Event, len(p.subscribers))
	copy(subs, p.subscribers)
	p.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub <- event:
		default:
			// Subscriber is slow or blocked, skip
		}
	}
}

// Subscribe creates a subscription channel for events
func (p *NATSPubSub) Subscribe() chan Event {
	ch := make(chan Event, 100)

	p.mu.Lock()
	p.subscribers = append(p.subscribers, ch)
	p.mu.Unlock()

	return ch
}

// Unsubscribe removes a subscription channel
func (p *NATSPubSub) Unsubscribe(ch chan Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, sub := range p.subscribers {
		if sub == ch {
			// Remove from slice
			p.subscribers = append(p.subscribers[:i], p.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// SubscribeJetStream creates a durable JetStream subscription
// This allows multiple instances to process events
func (p *NATSPubSub) SubscribeJetStream(consumerName string, handler func(Event)) error {
	_, err := p.js.Subscribe(p.subject, func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			msg.Nak()
			return
		}

		handler(event)
		msg.Ack()
	}, nats.Durable(consumerName), nats.ManualAck())

	return err
}

// Close closes the NATS connection
func (p *NATSPubSub) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, sub := range p.subscribers {
		close(sub)
	}
	p.subscribers = nil

	if p.nc != nil {
		p.nc.Close()
	}
}
