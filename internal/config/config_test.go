// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		expected    *Config
	}{
		{
			name: "valid config",
			configYAML: `
timeout: 5s
port: "9000"
log_level: "debug"
endpoints:
  - endpoint: "/test"
    method: GET
    backends:
      - url_pattern: "/api/test"
        host: "http://example.com"
`,
			expectError: false,
			expected: &Config{
				Timeout:         5 * time.Second,
				Port:            "9000",
				LogLevel:        "debug",
				ShutdownTimeout: 15 * time.Second,
				ServiceName:     "api-aggregator",
				Endpoints: []Endpoint{
					{
						Endpoint: "/test",
						Method:   "GET",
						Timeout:  5 * time.Second,
						Encoding: "json",
						Backends: []Backend{
							{
								URLPattern: "/api/test",
								Encoding:   "json",
								Host:       "http://example.com",
							},
						},
					},
				},
			},
		},
		{
			name: "missing endpoints",
			configYAML: `
timeout: 5s
port: "9000"
endpoints: []
`,
			expectError: true,
		},
		{
			name: "invalid encoding",
			configYAML: `
endpoints:
  - endpoint: "/test"
    encoding: "invalid"
    backends:
      - url_pattern: "/api/test"
        host: "http://example.com"
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
			require.NoError(t, err)
			defer func() {
				if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
					t.Logf("Failed to remove temp file: %v", removeErr)
				}
			}()

			_, err = tmpFile.WriteString(tt.configYAML)
			require.NoError(t, err)
			require.NoError(t, tmpFile.Close())

			// Load config
			cfg, err := LoadConfig(tmpFile.Name())

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.Timeout, cfg.Timeout)
			assert.Equal(t, tt.expected.Port, cfg.Port)
			assert.Equal(t, tt.expected.LogLevel, cfg.LogLevel)
			assert.Equal(t, tt.expected.ServiceName, cfg.ServiceName)
			assert.Len(t, cfg.Endpoints, len(tt.expected.Endpoints))

			if len(cfg.Endpoints) > 0 {
				assert.Equal(t, tt.expected.Endpoints[0].Endpoint, cfg.Endpoints[0].Endpoint)
				assert.Equal(t, tt.expected.Endpoints[0].Method, cfg.Endpoints[0].Method)
				assert.Equal(t, tt.expected.Endpoints[0].Encoding, cfg.Endpoints[0].Encoding)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	configYAML := `
endpoints:
  - endpoint: "/test"
    backends:
      - url_pattern: "/test"
        host: "http://example.com"
`

	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			t.Logf("Failed to remove temp file: %v", removeErr)
		}
	}()

	_, err = tmpFile.WriteString(configYAML)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	assert.Equal(t, "api-aggregator", cfg.ServiceName)
	assert.Equal(t, 10*time.Second, cfg.Timeout)

	// Check endpoint defaults
	assert.Equal(t, "GET", cfg.Endpoints[0].Method)
	assert.Equal(t, "json", cfg.Endpoints[0].Encoding)
	assert.Equal(t, 10*time.Second, cfg.Endpoints[0].Timeout)

	// Check backend defaults
	assert.Equal(t, "json", cfg.Endpoints[0].Backends[0].Encoding)
	assert.Equal(t, "/test", cfg.Endpoints[0].Backends[0].URLPattern) // Should default to endpoint path
	assert.Empty(t, cfg.Endpoints[0].Backends[0].RemoveHeaders)
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Create a temporary config file
	configYAML := `
endpoints:
  - endpoint: "/test"
    backends:
      - url_pattern: "/api/test"
        host: "http://example.com"
`
	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			t.Logf("Failed to remove temp file: %v", removeErr)
		}
	}()

	_, err = tmpFile.WriteString(configYAML)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// Set environment variables
	require.NoError(t, os.Setenv("API_AGGREGATOR_CONFIG_PATH", tmpFile.Name()))
	require.NoError(t, os.Setenv("API_AGGREGATOR_PORT", "9090"))
	require.NoError(t, os.Setenv("API_AGGREGATOR_LOG_LEVEL", "debug"))
	require.NoError(t, os.Setenv("API_AGGREGATOR_TRACING_ENABLED", "true"))
	require.NoError(t, os.Setenv("API_AGGREGATOR_SERVICE_NAME", "test-service"))
	defer func() {
		if unsetErr := os.Unsetenv("API_AGGREGATOR_CONFIG_PATH"); unsetErr != nil {
			t.Logf("Failed to unset env var: %v", unsetErr)
		}
		if unsetErr := os.Unsetenv("API_AGGREGATOR_PORT"); unsetErr != nil {
			t.Logf("Failed to unset env var: %v", unsetErr)
		}
		if unsetErr := os.Unsetenv("API_AGGREGATOR_LOG_LEVEL"); unsetErr != nil {
			t.Logf("Failed to unset env var: %v", unsetErr)
		}
		if unsetErr := os.Unsetenv("API_AGGREGATOR_TRACING_ENABLED"); unsetErr != nil {
			t.Logf("Failed to unset env var: %v", unsetErr)
		}
		if unsetErr := os.Unsetenv("API_AGGREGATOR_SERVICE_NAME"); unsetErr != nil {
			t.Logf("Failed to unset env var: %v", unsetErr)
		}
	}()

	cfg, err := LoadConfigFromEnv()
	require.NoError(t, err)

	// Check that environment variables override config values
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.TracingEnabled)
	assert.Equal(t, "test-service", cfg.ServiceName)
}

func TestBackendRemoveHeaders(t *testing.T) {
	configYAML := `
endpoints:
  - endpoint: "/test"
    backends:
      - url_pattern: "/api/test"
        host: "http://example.com"
        remove_headers:
          - "Authorization"
          - "X-Custom-Header"
`

	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			t.Logf("Failed to remove temp file: %v", removeErr)
		}
	}()

	_, err = tmpFile.WriteString(configYAML)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Check that remove_headers is properly loaded
	expected := []string{"Authorization", "X-Custom-Header"}
	assert.Equal(t, expected, cfg.Endpoints[0].Backends[0].RemoveHeaders)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing endpoint path",
			configYAML: `
endpoints:
  - method: GET
    backends:
      - url_pattern: "/api/test"
        host: "http://example.com"
`,
			expectError: true,
			errorMsg:    "endpoint path is required",
		},
		{
			name: "missing backend",
			configYAML: `
endpoints:
  - endpoint: "/test"
    method: GET
    backends: []
`,
			expectError: true,
			errorMsg:    "at least one backend is required",
		},
		{
			name: "url_pattern optional - should work",
			configYAML: `
endpoints:
  - endpoint: "/test"
    method: GET
    backends:
      - url_pattern: "/test"
        host: "http://example.com"
`,
			expectError: false,
		},
		{
			name: "missing host",
			configYAML: `
endpoints:
  - endpoint: "/test"
    method: GET
    backends:
      - url_pattern: "/api/test"
`,
			expectError: true,
			errorMsg:    "host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
			require.NoError(t, err)
			defer func() {
				if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
					t.Logf("Failed to remove temp file: %v", removeErr)
				}
			}()

			_, err = tmpFile.WriteString(tt.configYAML)
			require.NoError(t, err)
			require.NoError(t, tmpFile.Close())

			_, err = LoadConfig(tmpFile.Name())
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
