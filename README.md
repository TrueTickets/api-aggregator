# API Aggregator Service

[![Test](https://github.com/TrueTickets/api-aggregator/actions/workflows/test.yml/badge.svg)](https://github.com/TrueTickets/api-aggregator/actions/workflows/test.yml)
[![Release](https://github.com/TrueTickets/api-aggregator/actions/workflows/release.yml/badge.svg)](https://github.com/TrueTickets/api-aggregator/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/TrueTickets/api-aggregator)](https://goreportcard.com/report/github.com/TrueTickets/api-aggregator)
[![codecov](https://codecov.io/gh/TrueTickets/api-aggregator/branch/main/graph/badge.svg)](https://codecov.io/gh/TrueTickets/api-aggregator)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=flat&logo=docker&logoColor=white)](https://github.com/TrueTickets/api-aggregator/pkgs/container/api-aggregator)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/TrueTickets/api-aggregator)](https://github.com/TrueTickets/api-aggregator/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/TrueTickets/api-aggregator)](https://github.com/TrueTickets/api-aggregator/blob/main/go.mod)

A high-performance API aggregation and composition service built in Go,
designed to aggregate responses from multiple backend services and
provide a unified API interface.

## Features

- **Multiple Endpoint Support**: Configure arbitrary number of ingress
  endpoints
- **Backend Aggregation**: Aggregate responses from multiple upstream
  backends per endpoint
- **Response Transformation**: Comprehensive data transformation
  capabilities
- **Multiple Encodings**: Support for JSON, XML, and YAML
- **Compression Support**: Automatic handling of gzip and deflate
  compressed responses from backends
- **Timeout Management**: Configurable timeouts at global and endpoint
  levels
- **Observability**: OpenTelemetry tracing and metrics support
- **Health Monitoring**: Built-in health check endpoint
- **Production Ready**: Docker support, graceful shutdown, and
  comprehensive testing
- **Integration Testing**: Full test suite using Tavern with DummyJSON
  backends

## Response Transformations

### Filtering

- **Allow Lists**: Include only specific fields
- **Deny Lists**: Exclude specific fields
- **Nested Field Support**: Use dot notation for nested field access

### Grouping

Wrap backend responses in named groups to avoid field collisions:

```yaml
backend:
    - url_pattern: "/users/{id}"
      group: "user_data"
      host: ["http://user-service"]
```

### Mapping

Rename fields in the response:

```yaml
backend:
    - url_pattern: "/users/{id}"
      mapping:
          "fullName": "name"
          "emailAddress": "email"
      host: ["http://user-service"]
```

### Targeting (Capturing)

Extract nested data from generic containers:

```yaml
backend:
    - url_pattern: "/api/data"
      target: "response.data" # Extracts content from nested structure
      host: ["http://data-service"]
```

### Header Management

Control which headers are forwarded to backend services:

```yaml
backend:
    - url_pattern: "/secure-api"
      remove_headers:
          - "Authorization" # Remove auth header for this backend
          - "X-Internal-Token"
      host: ["http://public-service"]
```

### Response Concatenation

Append backend responses to arrays under specified keys:

```yaml
backend:
    - url_pattern: "/posts/{user}"
      concat: "user_posts" # Appends response to array under "user_posts" key
      host: ["http://posts-service"]
    - url_pattern: "/comments/{user}"
      concat: "user_posts" # Appends to the same array
      host: ["http://comments-service"]
    - url_pattern: "/likes/{user}"
      concat: "user_likes" # Creates separate array under "user_likes" key
      host: ["http://likes-service"]
```

Results in:

```json
{
    "user_posts": [
        { "id": 1, "title": "First Post", "content": "..." },
        { "id": 5, "post_id": 1, "text": "Great post!" }
    ],
    "user_likes": [{ "post_id": 1, "liked": true }]
}
```

### Compression Support

The API Aggregator automatically handles compressed responses from
backends using Go's standard library:

- **Standard Library Integration**: Uses Go's built-in HTTP client
  compression handling
- **Automatic Decompression**: Supports gzip, deflate, and other
  standard compression formats
- **Transparent Processing**: `Accept-Encoding` headers from clients are
  not forwarded to backends, allowing Go's HTTP client to handle
  compression automatically
- **Zero Configuration**: No setup required - compression handling works
  out of the box
- **Robust Error Handling**: Standard library provides reliable
  compression error handling

This approach leverages Go's proven HTTP client compression
capabilities, ensuring reliable handling of compressed backend responses
without custom implementation complexity.

The service automatically forwards all incoming request headers to
backends, except for system headers like `Host`, `Content-Length`,
`Transfer-Encoding`, `Connection`, `Upgrade`, and `Accept-Encoding`. Use
`remove_headers` to exclude specific headers per backend.

## Configuration

Configure endpoints via YAML file:

```yaml
timeout: 10s
port: "8080"
log_level: "info"

endpoints:
    - endpoint: "/users/{user}"
      method: GET
      timeout: 800ms
      backend:
          - url_pattern: "/users/{user}"
            host: ["https://jsonplaceholder.typicode.com"]
          - url_pattern: "/posts"
            host: ["https://jsonplaceholder.typicode.com"]
            allow: ["userId", "id", "title", "body"]
```

### Configuration Options

#### Global Settings

- `timeout`: Default timeout for all endpoints
- `port`: HTTP server port
- `log_level`: Logging level (debug, info, warn, error)
- `tracing_enabled`: Enable OpenTelemetry tracing
- `metrics_enabled`: Enable metrics collection

#### Endpoint Configuration

- `endpoint`: URL pattern with path parameters
- `method`: HTTP method (GET, POST, PUT, DELETE)
- `timeout`: Endpoint-specific timeout (overrides global)
- `encoding`: Default encoding for backends (json, xml, yaml)

#### Backend Configuration

- `url_pattern`: Backend URL pattern with parameter substitution
- `host`: List of backend hosts (supports load balancing)
- `encoding`: Backend-specific encoding (overrides endpoint)
- `remove_headers`: List of headers to remove before forwarding to this
  backend
- `group`: Group name for response wrapping
- `target`: Path to extract data from nested response
- `allow`: Fields to include (whitelist)
- `deny`: Fields to exclude (blacklist)
- `mapping`: Field name mapping (old_name: new_name)
- `concat`: Key name for appending response to an array

## Running the Service

### Local Development

```bash
# Build the service
go build -o api-aggregator ./cmd/api-aggregator/

# Run with default config
./api-aggregator

# Run with custom config
API_AGGREGATOR_CONFIG_PATH=custom-config.yaml ./api-aggregator
```

### Docker

```bash
# Build and run with docker-compose
docker-compose up --build

# Or build and run manually
docker build -t api-aggregator .
docker run -p 10400:8080 -v $(pwd)/config.yaml:/app/config.yaml api-aggregator
```

### Binary Installation

Download pre-built binaries from the [releases page](https://github.com/TrueTickets/api-aggregator/releases).

**Security Note**: All release binaries are signed with GPG. You can verify the signature using:

```bash
# Download the binary and signature
wget https://github.com/TrueTickets/api-aggregator/releases/download/v1.0.0/checksums.txt
wget https://github.com/TrueTickets/api-aggregator/releases/download/v1.0.0/checksums.txt.sig

# Verify the signature
gpg --verify checksums.txt.sig checksums.txt

# Verify the binary checksum
sha256sum -c checksums.txt --ignore-missing
```

## Testing

### Unit Tests

```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/server -v
```

### Integration Tests

The service includes a comprehensive integration test suite using Tavern
and DummyJSON:

```bash
# Run all integration tests with Docker Compose (recommended)
make integration-test

# Or run locally (requires Tavern: pip install tavern[pytest])
make integration-test-local
```

The integration tests cover:

- Basic response merging from multiple backends
- Field filtering (allow/deny lists)
- Response grouping and field mapping
- Target extraction from nested responses
- Complex multi-transformation scenarios
- Timeout and partial response handling

See [test/integration/README.md](test/integration/README.md) for
detailed information.

### Linting

```bash
# Run golangci-lint
make lint

# Or directly
golangci-lint run
```

## API Response Headers

The service adds the following response headers:

- `X-API-Aggregation-Completed`: `true` if all backends succeeded,
  `false` if some failed
- `Content-Type`: `application/json`

## Health Check

Health check endpoint is available at `/health`:

```bash
curl http://localhost:8080/health
```

Response:

```json
{
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z",
    "version": "1.0.0"
}
```

## Example Usage

### Simple Aggregation

Request to `/users/1` with configuration:

```yaml
endpoints:
    - endpoint: "/users/{user}"
      backend:
          - url_pattern: "/users/{user}"
            host: ["https://jsonplaceholder.typicode.com"]
          - url_pattern: "/posts"
            host: ["https://jsonplaceholder.typicode.com"]
            allow: ["userId", "id", "title", "body"]
```

Aggregates user data and posts into a single response.

### Grouped Responses

```yaml
endpoints:
    - endpoint: "/dashboard/{user}"
      backend:
          - url_pattern: "/users/{user}"
            group: "user"
            host: ["https://api.example.com"]
          - url_pattern: "/stats/{user}"
            group: "statistics"
            host: ["https://analytics.example.com"]
```

Results in:

```json
{
    "user": { "id": 1, "name": "John" },
    "statistics": { "posts": 5, "views": 120 }
}
```

## Environment Variables

All environment variables are prefixed with `API_AGGREGATOR_`:

- `API_AGGREGATOR_CONFIG_PATH`: Path to configuration file (default:
  config.yaml)
- `API_AGGREGATOR_PORT`: HTTP server port (default: 8080)
- `API_AGGREGATOR_LOG_LEVEL`: Log level (default: info)
- `API_AGGREGATOR_TRACING_ENABLED`: Enable tracing (default: false)
- `API_AGGREGATOR_TRACING_ENDPOINT`: OpenTelemetry endpoint
- `API_AGGREGATOR_METRICS_ENABLED`: Enable metrics (default: false)
- `API_AGGREGATOR_SERVICE_NAME`: Service name for telemetry (default:
  api-aggregator)

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting

```bash
# Run golangci-lint (from repository root)
./golinter.py run svc/api-aggregator
```

## Architecture

The service follows a modular architecture:

- **cmd/api-aggregator**: Main application entry point
- **internal/config**: Configuration loading and validation
- **internal/server**: HTTP server and routing
- **internal/client**: Backend HTTP client
- **internal/merger**: Response aggregation and transformation
- **internal/telemetry**: OpenTelemetry integration
- **internal/types**: Shared type definitions

## Performance Considerations

- Backend requests are made concurrently
- Configurable timeouts prevent hanging requests
- Partial responses are returned if some backends timeout
- Connection pooling for backend requests
- Graceful shutdown handling

## Contributing

1. Follow the existing code style and patterns
2. Add comprehensive unit tests for new features
3. Update documentation for configuration changes
4. Ensure all tests pass before submitting
