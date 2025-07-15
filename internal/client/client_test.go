// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestClient_Request_Logging(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message": "success", "data": [1, 2, 3]}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		logLevel zerolog.Level
		expected string
	}{
		{
			name:     "debug level logs request",
			logLevel: zerolog.DebugLevel,
			expected: "outgoing backend request",
		},
		{
			name:     "trace level logs request and response",
			logLevel: zerolog.TraceLevel,
			expected: "backend response received",
		},
		{
			name:     "info level no logging",
			logLevel: zerolog.InfoLevel,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer to capture log output
			var logBuf bytes.Buffer
			logger := zerolog.New(&logBuf).Level(tt.logLevel)

			// Create client
			client := New(Config{
				HTTPClient: &http.Client{},
				Tracer:     noop.NewTracerProvider().Tracer("test"),
				Logger:     logger,
			})

			// Make request
			result, err := client.Get(context.Background(), server.URL, "json", nil)

			// Verify result
			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Check logging output
			logOutput := logBuf.String()
			if tt.expected != "" {
				assert.Contains(t, logOutput, tt.expected)
			} else {
				assert.Empty(t, logOutput)
			}

			// For trace level, should contain both request and response logs
			if tt.logLevel == zerolog.TraceLevel {
				assert.Contains(t, logOutput, "outgoing backend request")
				assert.Contains(t, logOutput, "backend response received")
				assert.Contains(t, logOutput, "200") // status code
			}
		})
	}
}
