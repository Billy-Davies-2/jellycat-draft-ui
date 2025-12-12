package mocks

import (
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
)

// MockNATSPubSub provides a mock NATS/JetStream implementation for local development
type MockNATSPubSub struct {
	*pubsub.PubSub
}

// NewMockNATSPubSub creates a mock NATS pub/sub using the in-memory implementation
func NewMockNATSPubSub() *MockNATSPubSub {
	logger.Info("Using MOCK NATS/JetStream (in-memory pub/sub) for local development")

	return &MockNATSPubSub{
		PubSub: pubsub.New(),
	}
}

// Close is a no-op for mock
func (m *MockNATSPubSub) Close() {
	// No cleanup needed for in-memory
}
