# API Aggregator - Claude Context Guide

## Project Overview

**API Aggregator** is a high-performance Go service that aggregates responses from multiple backend APIs and provides a unified interface. It was originally an internal service that has been prepared for open-source release.

## Architecture

```
cmd/api-aggregator/          # Main application entry point
├── main.go                  # Application bootstrap and configuration
├── logging.go               # Structured logging setup
├── reloadable_server.go     # HTTP server with graceful shutdown
└── telemetry.go             # OpenTelemetry tracing and metrics

internal/                    # Internal packages (not public API)
├── build/                   # Build-time information
├── client/                  # HTTP client for backend requests
├── config/                  # Configuration parsing and validation
├── merger/                  # Response aggregation and merging logic
├── server/                  # HTTP server implementation
├── telemetry/               # OpenTelemetry integration
├── transformer/             # Response transformation engine
└── types/                   # Shared type definitions

test/integration/            # Integration test suite
└── tavern/                  # Tavern-based API tests
```

## Key Features

- **Multi-backend aggregation**: Combines responses from multiple APIs
- **Response transformation**: Filtering, grouping, mapping, targeting
- **Flexible configuration**: YAML-based endpoint and backend configuration
- **Observability**: OpenTelemetry tracing and metrics
- **Production-ready**: Docker support, health checks, graceful shutdown
- **Comprehensive testing**: Unit and integration tests

## Configuration

The service uses YAML configuration files. Key configuration concepts:

- **Endpoints**: Define ingress API endpoints
- **Backends**: Upstream services to aggregate from
- **Transformations**: Data manipulation (filter, group, map, target)
- **Timeouts**: Global and per-endpoint timeout settings
- **Encoding**: Support for JSON, XML, and YAML

Example configuration structure:
```yaml
timeout: 10s
port: "8080"
endpoints:
  - endpoint: "/users/{id}"
    method: GET
    backends:
      - url_pattern: "/users/{id}"
        host: "https://api.example.com"
        group: "user_data"
```

## Development Commands

```bash
# Development workflow
make dev                     # Build, test, and lint
make build                   # Build binary
make test                    # Run unit tests
make lint                    # Run golangci-lint
make integration-test        # Run integration tests with Docker
make clean                   # Clean build artifacts

# Testing
go test ./...                # Unit tests
make integration-test        # Full integration test suite
```

## Docker & Deployment

- **Dockerfile**: Multi-stage build for production
- **Dockerfile.goreleaser**: Optimized for pre-built binaries
- **docker-compose.yml**: Development environment
- **docker-compose.integration.yaml**: Integration testing

## CI/CD Pipeline

### GitHub Actions Workflows

1. **test.yml**: Runs on push/PR
   - Unit tests with coverage
   - Integration tests with Tavern
   - Linting with golangci-lint
   - Multi-platform build verification

2. **release.yml**: Runs on git tags
   - GoReleaser for multi-arch binaries
   - Docker image builds
   - GitHub Container Registry publishing
   - GPG signing of release artifacts

### Release Process

1. Create and push a git tag: `git tag v1.0.0 && git push origin v1.0.0`
2. GitHub Actions automatically:
   - Builds binaries for Linux, macOS, Windows (amd64, arm64)
   - Creates Docker images for linux/amd64
   - Signs checksums with GPG
   - Publishes to GitHub Releases and GHCR

## Code Quality & Standards

- **Pre-commit hooks**: Automated formatting, linting, and testing
- **MIT License**: All Go files include copyright headers
- **Code coverage**: Tracked with codecov
- **Linting**: golangci-lint, yamllint, hadolint
- **Go Report Card**: Maintains high code quality score

## Dependencies

- **Go 1.23+**: Required for build
- **Chi router**: HTTP routing
- **Zerolog**: Structured logging
- **OpenTelemetry**: Observability
- **Testify**: Testing framework
- **Docker**: Container deployment
- **Tavern**: Integration testing

## Build Configuration

### GoReleaser
- Multi-architecture binary builds
- Docker image creation
- GPG signing of releases
- GitHub Releases automation

### Build-time Information
The service injects build information via ldflags:
- `ServiceVersion`: Git tag or "dev"
- `GitCommit`: Git commit hash
- `BuildDate`: Build timestamp

## Security

- **GPG Signing**: All release binaries are GPG signed
- **Container Security**: Runs as non-root user
- **Dependency Scanning**: Trivy security scanning
- **No secrets in code**: All sensitive config via environment variables

## Environment Variables

All configuration can be overridden with environment variables prefixed with `API_AGGREGATOR_`:

- `API_AGGREGATOR_CONFIG_PATH`: Configuration file path
- `API_AGGREGATOR_PORT`: HTTP server port
- `API_AGGREGATOR_LOG_LEVEL`: Logging level
- `API_AGGREGATOR_TRACING_ENABLED`: Enable OpenTelemetry tracing
- `API_AGGREGATOR_METRICS_ENABLED`: Enable metrics collection

## Common Issues & Solutions

1. **Docker containers not starting**: Check that all required packages are installed (ca-certificates, curl, libc6-compat)
2. **Integration tests failing**: Ensure Docker Compose is available and ports are not in use
3. **Build failures**: Check Go version (1.23+ required) and module dependencies
4. **Pre-commit hook failures**: Install required tools (golangci-lint, yamllint, hadolint)

## Open Source Preparation

This project was prepared for open-source release by:
- Removing all internal company references
- Updating module path to GitHub repository
- Adding comprehensive documentation and badges
- Implementing professional CI/CD pipeline
- Adding proper licensing and security measures

## Testing Strategy

- **Unit Tests**: Cover all internal packages with high coverage
- **Integration Tests**: End-to-end testing with Tavern framework
- **Docker Tests**: Validate container builds and deployments
- **CI Testing**: Automated testing on multiple platforms

## Monitoring & Observability

- **Health Endpoint**: `/health` for liveness/readiness checks
- **OpenTelemetry**: Distributed tracing support
- **Metrics**: Custom metrics for aggregation performance
- **Structured Logging**: JSON-formatted logs with correlation IDs

## Contributing

- Use pre-commit hooks for code quality
- Follow existing code patterns and architecture
- Add tests for new functionality
- Update documentation for configuration changes
- Ensure all CI checks pass before merging

### Commit Message Guidelines

**Atomic commits**. Create small, focused commits that represent a
single logical change. Each commit should be self-contained and make
sense on its own.

Right before creating a commit, read the
`.claude/custom/commit_formatting.md` file for instructions on how you
must format the commit message.
