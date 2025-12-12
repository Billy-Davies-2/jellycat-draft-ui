# gRPC and NATS Architecture

This document explains how gRPC and NATS work together in the Jellycat Draft application for real-time messaging and event streaming.

## Overview

The application provides **dual interfaces** for real-time updates:

1. **HTTP Server-Sent Events (SSE)**: For web browsers and HTTP clients
2. **gRPC Streaming**: For programmatic API access and microservices

Both interfaces use **NATS JetStream** as the underlying message broker for pub/sub functionality.

## Architecture

```
┌─────────────────┐
│   Web Browser   │
│   (htmx/SSE)    │
└────────┬────────┘
         │ HTTP SSE
         ▼
┌─────────────────────────────────────┐
│      Go Application (main.go)       │
│  ┌──────────────┐  ┌─────────────┐ │
│  │ HTTP Handler │  │ gRPC Server │ │
│  └──────┬───────┘  └──────┬──────┘ │
│         │                 │         │
│         └────────┬────────┘         │
│                  ▼                  │
│         ┌──────────────────┐        │
│         │  Local PubSub    │        │
│         │   (in-memory)    │        │
│         └────────┬─────────┘        │
│                  │                  │
└──────────────────┼──────────────────┘
                   │
                   ▼
         ┌──────────────────┐
         │  NATS JetStream  │
         │  (Message Broker)│
         └──────────────────┘
                   ▲
                   │
         ┌─────────┴──────────┐
         │                    │
    ┌────▼─────┐      ┌──────▼────┐
    │ gRPC     │      │ Other     │
    │ Client   │      │ Services  │
    └──────────┘      └───────────┘
```

## Why Keep gRPC?

### Use Cases

1. **Programmatic API Access**: External services can consume draft events via gRPC
2. **Microservice Integration**: Other microservices can integrate using type-safe gRPC
3. **Real-time Event Streaming**: gRPC `StreamEvents` provides real-time updates without polling
4. **Chat Message Streaming**: Chat messages are published via NATS and streamed to gRPC clients

### gRPC StreamEvents + NATS Flow

1. Client calls `StreamEvents()` gRPC method
2. Server subscribes to NATS pub/sub channel
3. When events are published to NATS (e.g., draft picks, chat messages):
   - NATS broadcasts to all subscribers
   - gRPC server receives event from NATS
   - Event is streamed to gRPC client
4. Client receives real-time updates

Example:
```go
// gRPC client code
stream, err := client.StreamEvents(ctx, &pb.Empty{})
for {
    event, err := stream.Recv()
    if err != nil {
        break
    }
    // Handle event: draft:pick, chat:add, etc.
    fmt.Printf("Event: %s, Payload: %v\n", event.Type, event.Payload)
}
```

## Event Types

Events published via NATS and available on both HTTP SSE and gRPC streams:

- `draft:pick` - Player drafted by a team
- `draft:reset` - Draft reset to initial state
- `teams:add` - New team added
- `teams:reorder` - Teams reordered
- `players:add` - New player added
- `players:updatePoints` - Player points updated
- `chat:add` - New chat message
- `chat:react` - Reaction added to message

## NATS Configuration

### Development Mode

```bash
ENVIRONMENT=development
# Uses MockNATSPubSub - in-memory simulation, no NATS server needed
```

### Production Mode

```bash
ENVIRONMENT=production
NATS_URL=nats://nats.default.svc.cluster.local:4222
NATS_SUBJECT=draft.events

# Optional: Authentication
NATS_USERNAME=user
NATS_PASSWORD=password
NATS_CREDS=/path/to/creds.file
```

## gRPC Endpoints for Chat and Streaming

### Chat Operations (Use NATS pub/sub)

```protobuf
// Send a chat message (published to NATS)
rpc SendChatMessage(SendChatRequest) returns (ChatMessage);

// Add reaction (published to NATS)
rpc AddReaction(AddReactionRequest) returns (ChatMessage);
```

When these are called:
1. Message is saved to database
2. Event is published to NATS
3. All subscribers (HTTP SSE, gRPC StreamEvents) receive the event
4. UI updates in real-time

### Event Streaming

```protobuf
// Stream all events in real-time
rpc StreamEvents(Empty) returns (stream Event);
```

This provides server-to-client streaming of all draft and chat events via NATS.

## Should You Remove gRPC?

**No**, keep gRPC because:

1. ✅ **It's used for NATS streaming**: `StreamEvents` provides real-time updates via NATS
2. ✅ **Chat uses NATS**: Chat messages are published to NATS and consumed via gRPC
3. ✅ **Programmatic API**: gRPC provides type-safe API for external services
4. ✅ **Already implemented**: Removing it provides no benefit
5. ✅ **Dual interface**: HTTP for browsers, gRPC for services

**When to use HTTP SSE**:
- Web browsers with htmx
- Simple integrations
- Testing with curl

**When to use gRPC**:
- Microservices communication
- Programmatic clients (Go, Python, Node.js)
- Type-safe API contracts
- Efficient binary protocol

## Deployment Considerations

### Kubernetes Deployment

Both HTTP and gRPC are exposed:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: jellycat-draft
spec:
  ports:
  - name: http
    port: 80
    targetPort: 3000
  - name: grpc
    port: 50051
    targetPort: 50051
```

### NATS Deployment

For production, deploy NATS with JetStream:

```bash
# Using Helm
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm install nats nats/nats --set nats.jetstream.enabled=true
```

## Monitoring

### Health Checks

The `/api/health` endpoint checks NATS connectivity in production:

```json
{
  "status": "ok",
  "checks": {
    "database": {"status": "healthy"},
    "nats": {"status": "healthy"},
    "clickhouse": {"status": "healthy"}
  }
}
```

### Metrics

- **HTTP SSE connections**: Number of active SSE connections
- **gRPC streams**: Number of active gRPC StreamEvents connections
- **NATS messages**: Messages published/received per second

## Summary

- **gRPC is essential** for NATS-based event streaming and chat functionality
- Both HTTP SSE and gRPC use NATS as the message broker
- Removing gRPC would remove programmatic API access and type-safe contracts
- The dual interface (HTTP + gRPC) provides flexibility for different clients

## References

- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [Main README](../README.md)
