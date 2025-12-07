# syntax=docker/dockerfile:1

# Multi-stage Dockerfile for Go application

FROM golang:1.24-alpine AS builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o jellycat-draft main.go

FROM alpine:latest
WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

# Copy binary and static files from builder
COPY --from=builder /app/jellycat-draft .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Environment variables
ENV PORT=3000 \
    GRPC_PORT=50051 \
    DB_DRIVER=sqlite \
    SQLITE_FILE=/data/draft.sqlite

# Create data directory for SQLite
RUN mkdir -p /data

EXPOSE 3000 50051

# Run the application
CMD ["./jellycat-draft"]
