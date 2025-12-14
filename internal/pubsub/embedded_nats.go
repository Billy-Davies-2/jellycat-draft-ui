package pubsub

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// EmbeddedNATSPubSub implements pub/sub using an embedded NATS server
// This is ideal for development as it provides a real NATS server in-process
// without requiring external infrastructure
type EmbeddedNATSPubSub struct {
	server      *server.Server
	nc          *nats.Conn
	js          nats.JetStreamContext
	subject     string
	subscribers []chan Event
	mu          sync.RWMutex
}

// EmbeddedNATSOptions configures the embedded NATS server
type EmbeddedNATSOptions struct {
	Port       int    // Port to listen on (0 = random available port)
	Subject    string // Subject to publish/subscribe to
	StreamName string // JetStream stream name
	StoreDir   string // Directory for JetStream storage (empty = in-memory)
}

// DefaultEmbeddedNATSOptions returns sensible defaults for development
func DefaultEmbeddedNATSOptions() EmbeddedNATSOptions {
	return EmbeddedNATSOptions{
		Port:       -1, // Random available port
		Subject:    "draft.events",
		StreamName: "DRAFT_EVENTS",
		StoreDir:   "", // In-memory
	}
}

// NewEmbeddedNATSPubSub creates a new embedded NATS server and pub/sub
func NewEmbeddedNATSPubSub(opts EmbeddedNATSOptions) (*EmbeddedNATSPubSub, error) {
	// Configure the embedded server
	// Use port -1 to select a random available port
	port := opts.Port
	if port == 0 {
		port = -1 // 0 means default (4222), -1 means random
	}

	serverOpts := &server.Options{
		Port:      port,
		JetStream: true,
		NoLog:     false,
		NoSigs:    true, // Don't register signal handlers
	}

	if opts.StoreDir != "" {
		serverOpts.StoreDir = opts.StoreDir
	}

	// Start the embedded server
	ns, err := server.NewServer(serverOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedded NATS server: %w", err)
	}

	// Configure logging to use our logger
	ns.SetLogger(&natsLogger{}, false, false)

	go ns.Start()

	// Wait for server to be ready
	if !ns.ReadyForConnections(10 * time.Second) {
		ns.Shutdown()
		return nil, fmt.Errorf("embedded NATS server failed to start within timeout")
	}

	clientURL := ns.ClientURL()
	logger.Info("Embedded NATS server started", "url", clientURL)

	// Connect to the embedded server
	nc, err := nats.Connect(clientURL)
	if err != nil {
		ns.Shutdown()
		return nil, fmt.Errorf("failed to connect to embedded NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create stream
	streamName := opts.StreamName
	if streamName == "" {
		streamName = "DRAFT_EVENTS"
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{opts.Subject},
		Storage:  nats.MemoryStorage, // Use memory storage for dev
		MaxAge:   time.Hour,          // Keep events for 1 hour
	})
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, fmt.Errorf("failed to create JetStream stream: %w", err)
	}

	logger.Info("JetStream stream created", "stream", streamName, "subject", opts.Subject)

	ps := &EmbeddedNATSPubSub{
		server:      ns,
		nc:          nc,
		js:          js,
		subject:     opts.Subject,
		subscribers: make([]chan Event, 0),
	}

	// Start a goroutine to receive messages from JetStream and broadcast to local subscribers
	go ps.startSubscription()

	return ps, nil
}

// startSubscription subscribes to the JetStream subject and broadcasts to local subscribers
func (p *EmbeddedNATSPubSub) startSubscription() {
	_, err := p.js.Subscribe(p.subject, func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("Failed to unmarshal event from JetStream", "error", err)
			msg.Nak()
			return
		}

		// Broadcast to all local subscribers
		p.mu.RLock()
		subs := make([]chan Event, len(p.subscribers))
		copy(subs, p.subscribers)
		p.mu.RUnlock()

		for _, sub := range subs {
			select {
			case sub <- event:
			default:
				logger.Warn("Embedded NATS: Skipping slow subscriber", "event_type", event.Type)
			}
		}

		msg.Ack()
	}, nats.ManualAck(), nats.DeliverNew())

	if err != nil {
		logger.Error("Failed to subscribe to JetStream", "error", err, "subject", p.subject)
		return
	}

	logger.Debug("Subscribed to JetStream", "subject", p.subject)
}

// Publish publishes an event to the embedded NATS JetStream
func (p *EmbeddedNATSPubSub) Publish(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		logger.Error("Failed to marshal event", "error", err, "event_type", event.Type)
		return
	}

	// Publish to JetStream
	_, err = p.js.Publish(p.subject, data)
	if err != nil {
		logger.Error("Failed to publish to embedded NATS", "error", err, "subject", p.subject, "event_type", event.Type)
		return
	}

	logger.Debug("Published event to embedded NATS", "event_type", event.Type, "subject", p.subject)
}

// Subscribe creates a subscription channel for events
func (p *EmbeddedNATSPubSub) Subscribe() chan Event {
	ch := make(chan Event, 100)

	p.mu.Lock()
	p.subscribers = append(p.subscribers, ch)
	subCount := len(p.subscribers)
	p.mu.Unlock()

	logger.Debug("Embedded NATS: New subscriber added", "total_subscribers", subCount)
	return ch
}

// Unsubscribe removes a subscription channel
func (p *EmbeddedNATSPubSub) Unsubscribe(ch chan Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, sub := range p.subscribers {
		if sub == ch {
			p.subscribers = append(p.subscribers[:i], p.subscribers[i+1:]...)
			close(ch)
			logger.Debug("Embedded NATS: Subscriber removed", "remaining_subscribers", len(p.subscribers))
			break
		}
	}
}

// Close shuts down the embedded NATS server
func (p *EmbeddedNATSPubSub) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	logger.Info("Shutting down embedded NATS server")

	for _, sub := range p.subscribers {
		close(sub)
	}
	p.subscribers = nil

	if p.nc != nil {
		p.nc.Close()
	}

	if p.server != nil {
		p.server.Shutdown()
		p.server.WaitForShutdown()
	}

	logger.Info("Embedded NATS server shut down")
}

// GetServerURL returns the URL of the embedded NATS server
// This can be useful for debugging or connecting additional clients
func (p *EmbeddedNATSPubSub) GetServerURL() string {
	return p.server.ClientURL()
}

// GetSubscriberCount returns the number of active local subscribers
func (p *EmbeddedNATSPubSub) GetSubscriberCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subscribers)
}

// natsLogger adapts our logger to the NATS server logger interface
type natsLogger struct{}

func (l *natsLogger) Noticef(format string, v ...interface{}) {
	logger.Info(fmt.Sprintf("[NATS] "+format, v...))
}

func (l *natsLogger) Warnf(format string, v ...interface{}) {
	logger.Warn(fmt.Sprintf("[NATS] "+format, v...))
}

func (l *natsLogger) Fatalf(format string, v ...interface{}) {
	logger.Error(fmt.Sprintf("[NATS] "+format, v...))
}

func (l *natsLogger) Errorf(format string, v ...interface{}) {
	logger.Error(fmt.Sprintf("[NATS] "+format, v...))
}

func (l *natsLogger) Debugf(format string, v ...interface{}) {
	logger.Debug(fmt.Sprintf("[NATS] "+format, v...))
}

func (l *natsLogger) Tracef(format string, v ...interface{}) {
	logger.Debug(fmt.Sprintf("[NATS TRACE] "+format, v...))
}
