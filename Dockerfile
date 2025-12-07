# syntax=docker/dockerfile:1

# Multi-stage Dockerfile for Go application

FROM golang:1.24-alpine AS builder
WORKDIR /app

# Update CA certificates and install build dependencies for static compilation
RUN apk update && apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application as a static binary
# Note: We link statically against musl and sqlite
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -ldflags '-linkmode external -extldflags "-static"' \
    -tags netgo,osusergo \
    -o jellycat-draft main.go

FROM scratch
WORKDIR /app

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary and static files from builder
COPY --from=builder /app/jellycat-draft .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Environment variables
# Note: Using memory driver by default since scratch has no writable filesystem
# For SQLite, mount a volume at runtime
ENV PORT=3000 \
    GRPC_PORT=50051 \
    DB_DRIVER=memory

EXPOSE 3000 50051

# Run the application
CMD ["./jellycat-draft"]
