# API Aggregator Makefile

.PHONY: build test lint integration-test clean help

# Default target
help:
	@echo "Available targets:"
	@echo "  build               - Build the API aggregator binary"
	@echo "  test                - Run unit tests"
	@echo "  lint                - Run golangci-lint"
	@echo "  integration-test    - Run integration tests with Docker Compose"
	@echo "  integration-test-local - Run integration tests locally"
	@echo "  clean               - Clean build artifacts"
	@echo "  help                - Show this help message"

# Build the binary
build:
	go build -o bin/api-aggregator ./cmd/api-aggregator

# Run unit tests
test:
	go test ./... -v

# Run linter
lint:
	golangci-lint run

# Run integration tests with Docker Compose
integration-test:
	@echo "Running integration tests with Docker Compose..."
	@docker compose -f docker-compose.integration.yaml up --build --abort-on-container-exit
	@docker compose -f docker-compose.integration.yaml down

# Run integration tests locally (without Docker)
integration-test-local:
	@echo "Starting API aggregator with integration test config..."
	@go build -o api-aggregator ./cmd/api-aggregator
	@API_AGGREGATOR_CONFIG_PATH=test/integration/config.yaml ./api-aggregator &
	@echo "Waiting for service to start..."
	@sleep 3
	@echo "Running integration tests..."
	@cd test/integration/tavern && API_BASE_URL=http://localhost:8080 pytest *.tavern.yaml -v
	@echo "Stopping service..."
	@pkill -f api-aggregator || true

# Clean build artifacts
clean:
	rm -f bin/api-aggregator
	rm -f api-aggregator

# Run all checks
check: lint test integration-test

# Development workflow
dev: build test lint
