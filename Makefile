.PHONY: all build test fuzz-test fuzz-http fuzz-grpc clean proto

# Build the application
all: build

# Build the Go binary
build:
	go build -o jellycat-draft main.go

# Run all tests
test:
	go test ./...

# Run fuzz tests for HTTP endpoints (specify time with FUZZTIME, default 30s)
fuzz-http:
	@echo "Fuzzing HTTP endpoints..."
	go test -fuzz=FuzzHTTPDraftPick -fuzztime=${FUZZTIME:-30s} ./internal/fuzz
	go test -fuzz=FuzzHTTPAddTeam -fuzztime=${FUZZTIME:-30s} ./internal/fuzz
	go test -fuzz=FuzzHTTPSendChat -fuzztime=${FUZZTIME:-30s} ./internal/fuzz
	go test -fuzz=FuzzHTTPSetPlayerPoints -fuzztime=${FUZZTIME:-30s} ./internal/fuzz

# Run fuzz tests for gRPC endpoints (specify time with FUZZTIME, default 30s)
fuzz-grpc:
	@echo "Fuzzing gRPC endpoints..."
	go test -fuzz=FuzzGRPCDraftPlayer -fuzztime=${FUZZTIME:-30s} ./internal/fuzz
	go test -fuzz=FuzzGRPCAddTeam -fuzztime=${FUZZTIME:-30s} ./internal/fuzz
	go test -fuzz=FuzzGRPCSendChatMessage -fuzztime=${FUZZTIME:-30s} ./internal/fuzz

# Run all fuzz tests
fuzz-test: fuzz-http fuzz-grpc

# Regenerate protobuf files
proto:
	@echo "Generating protobuf files..."
	@if [ ! -f /tmp/protoc/bin/protoc ]; then \
		curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v25.1/protoc-25.1-linux-x86_64.zip && \
		unzip -o protoc-25.1-linux-x86_64.zip -d /tmp/protoc && \
		rm protoc-25.1-linux-x86_64.zip; \
	fi
	PATH="$$PATH:/tmp/protoc/bin:$$HOME/go/bin" /tmp/protoc/bin/protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/draft.proto

# Clean build artifacts
clean:
	rm -f jellycat-draft
	rm -f protoc-*.zip

# Run the application with memory storage
run-memory:
	DB_DRIVER=memory ./jellycat-draft

# Run the application with SQLite storage
run-sqlite:
	DB_DRIVER=sqlite SQLITE_FILE=draft.db ./jellycat-draft

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	go vet ./...
