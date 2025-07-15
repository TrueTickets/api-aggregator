# Integration Tests

This directory contains integration tests for the API Aggregator service
using [Tavern](https://tavern.readthedocs.io/) and
[DummyJSON](https://dummyjson.com/) as test backend data sources.

## Overview

The integration tests validate the full functionality of the API
Aggregator service, including:

- **Basic Merge**: Aggregating responses from multiple backends
- **Filtering**: Allow and deny lists for field filtering
- **Grouping**: Organizing backend responses into named groups
- **Field Mapping**: Renaming fields in responses
- **Target Extraction**: Capturing data from nested response structures
- **Complex Scenarios**: Multiple transformations applied together
- **Timeout Handling**: Partial responses when backends are slow
- **Response Concatenation**: Appending backend responses to arrays

## Prerequisites

### Docker Compose (Recommended)

The easiest way to run integration tests is using Docker Compose, which
requires no local dependencies except Docker.

### Local Testing

For local testing without Docker:

```bash
pip install tavern[pytest]
```

Tests require internet access to reach
[DummyJSON](https://dummyjson.com/) API endpoints.

## Running Tests

### Option 1: Docker Compose (Recommended)

```bash
# From the root directory
make integration-test

# Or run directly
docker compose -f docker-compose.integration.yaml up --build --abort-on-container-exit
```

This approach:

- Builds the API aggregator service in Docker
- Starts the service with the integration test configuration
- Runs all Tavern tests in a separate container
- Automatically shuts down when tests complete

### Option 2: Local Testing

```bash
# From the root directory
make integration-test-local

# Or manually:
# 1. Start the service
API_AGGREGATOR_CONFIG_PATH=test/integration/config.yaml ./api-aggregator

# 2. Run tests (in another terminal)
cd test/integration && API_BASE_URL=http://localhost:8080 pytest *.tavern.yaml -v
```

## Test Configuration

All test scenarios are configured in a single `config.yaml` file that
defines:

- **10 different endpoints** covering all transformation types
- **DummyJSON backend integrations** for realistic data
- **Various transformation combinations** to test complex scenarios

### Example Endpoint Configuration

```yaml
endpoints:
    # Basic merge example
    - endpoint: /users/{id}
      method: GET
      timeout: 5s
      backends:
          - host: https://dummyjson.com
            url_pattern: /users/{id}
          - host: https://dummyjson.com
            url_pattern: /posts/user/{id}
            group: posts

    # Complex transformations example
    - endpoint: /complex-user/{id}
      backends:
          - host: https://dummyjson.com
            url_pattern: /users/{id}
            group: profile
            transformations:
                filter:
                    allow: [id, firstName, lastName, email, age]
                mapping:
                    firstName: first_name
                    lastName: last_name
```

## Test Scenarios

### 1. Basic Merge (`basic_merge.tavern.yaml`)

- **Endpoint**: `/users/{id}`
- **Tests**: Merging user data with posts
- **Validates**: Response structure, aggregation headers

### 2. Filtering (`filtering.tavern.yaml`)

- **Endpoint**: `/products/{id}`
- **Tests**: Allow list filtering to specific fields
- **Validates**: Only allowed fields present, no forbidden fields

### 3. Grouping (`grouping.tavern.yaml`)

- **Endpoint**: `/user-data/{id}`
- **Tests**: Multiple backends grouped separately
- **Validates**: Each response in correct group structure

### 4. Field Mapping (`mapping.tavern.yaml`)

- **Endpoint**: `/product-info/{id}`
- **Tests**: Field renaming transformations
- **Validates**: New field names, original names removed

### 5. Target Extraction (`targeting.tavern.yaml`)

- **Endpoint**: `/all-products`
- **Tests**: Extracting arrays from wrapper objects
- **Validates**: Direct array response without wrapper

### 6. Deny List Filtering (`deny_filter.tavern.yaml`)

- **Endpoint**: `/users-filtered/{id}`
- **Tests**: Filtering out sensitive fields
- **Validates**: Sensitive data removed, including nested fields

### 7. Complex Scenario (`complex_scenario.tavern.yaml`)

- **Endpoint**: `/complex-user/{id}`
- **Tests**: Multiple transformations combined
- **Validates**: Grouping + filtering + mapping + targeting together

### 8. Timeout Handling (`timeout.tavern.yaml`)

- **Endpoint**: `/timeout-test/{id}`
- **Tests**: Very short timeout (100ms) with multiple backends
- **Validates**: Partial responses, proper headers

### 9. Response Concatenation (`concat.tavern.yaml`)

- **Endpoint**: `/user-activities/{id}` and `/user-mixed/{id}`
- **Tests**: Appending multiple backend responses to arrays
- **Validates**: Array structure, multiple items, mixed with grouping

## Test Validation

Each test validates:

- **HTTP Status Codes**: 200 for successful requests
- **Response Headers**: Content-Type and X-API-Aggregation-Completed
- **JSON Structure**: Using pykwalify schemas and jmespath queries
- **Field Presence/Absence**: Ensuring transformations work correctly
- **Data Types**: Validating response data types match expectations

## Example Test Structure

```yaml
test_name: Test field mapping functionality

stages:
    - name: Test field mapping
      request:
          url: http://localhost:8080/product-info/1
          method: GET
      response:
          status_code: 200
          headers:
              content-type: application/json
              x-api-aggregation-completed: "true"
          json:
              id: !anyint
              product_name: !anystr # mapped from title


              cost: !anyfloat # mapped from price


          verify_response_with:
              function: tavern.testutils.helpers:validate_pykwalify
              extra_kwargs:
                  schema:
                      type: map
                      mapping:
                          id: { type: int, required: true }
                          product_name: { type: str, required: true }
                          cost: { type: float, required: true }
```

## Troubleshooting

### Service Not Starting

Ensure the API aggregator builds and config file path is correct:

```bash
go build -o api-aggregator ./cmd/api-aggregator
API_AGGREGATOR_CONFIG_PATH=tests/integration/config.yaml ./api-aggregator
```

### DummyJSON Connection Issues

Test direct access to DummyJSON:

```bash
curl https://dummyjson.com/users/1
```

### Test Failures

Run individual tests for debugging:

```bash
pytest basic_merge.tavern.yaml -v -s
```

### Port Conflicts

Ensure port 8080 is available:

```bash
lsof -i :8080
```

## CI/CD Integration

For continuous integration, use Docker Compose for the simplest setup:

```bash
# Example CI commands
make integration-test

# Or directly
docker compose -f docker-compose.integration.yaml up --build --abort-on-container-exit
```

This approach requires only Docker and automatically handles:

- Service building and configuration
- Network setup between services
- Test execution and cleanup
- No additional dependencies or environment setup
