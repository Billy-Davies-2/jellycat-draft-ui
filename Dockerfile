# syntax=docker/dockerfile:1

# Multi-stage Dockerfile for Go application

FROM golang:1.25-alpine AS builder
WORKDIR /app

# Update CA certificates and install build dependencies for static compilation
RUN apk update && apk add --no-cache gcc musl-dev sqlite-dev curl ca-certificates

# Download TailwindCSS standalone CLI
RUN curl -sL https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.18/tailwindcss-linux-x64-musl -o tailwindcss && \
  chmod +x tailwindcss
# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build TailwindCSS
RUN ./tailwindcss -i static/css/input.css -o static/css/styles.css --minify

# Build the application as a static binary
# Note: We link statically against musl and sqlite
RUN CGO_ENABLED=1 GOOS=linux go build \
  -a \
  -ldflags '-linkmode external -extldflags "-static"' \
  -tags netgo,osusergo \
  -o jellycat-draft main.go

# Use distroless for better SSL/TLS support in production
# distroless/static-debian12 contains CA certificates and timezone data
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

# Copy binary and static files from builder
COPY --from=builder /app/jellycat-draft .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Environment variables
# DB_DRIVER defaults to memory but should be set to postgres for production
# Set ENVIRONMENT=production for production deployments
ENV PORT=3000 \
  GRPC_PORT=50051 \
  DB_DRIVER=memory

EXPOSE 3000 50051

# Run the application
# Using ENTRYPOINT for better compatibility with distroless
ENTRYPOINT ["./jellycat-draft"]
