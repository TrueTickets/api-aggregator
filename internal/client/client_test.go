// Copyright (c) 2025-2026 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.NotNil(t, result.Data)
			assert.Equal(t, http.StatusOK, result.StatusCode)

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

func TestClient_Request_FailureAfterResponse(t *testing.T) {
	const marker = "SENSITIVE-BODY-MARKER"

	tests := []struct {
		name          string
		encoding      string
		handler       http.HandlerFunc
		expectedCode  int
		expectedError string
	}{
		{
			name:     "error status",
			encoding: "json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, err := w.Write([]byte(`{"error": "` + marker + `"}`))
				assert.NoError(t, err)
			},
			expectedCode:  http.StatusServiceUnavailable,
			expectedError: "backend returned status 503",
		},
		{
			name:     "body read failure",
			encoding: "json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "100")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"a"`))
				assert.NoError(t, err)
			},
			expectedCode:  http.StatusOK,
			expectedError: "failed to read response body: unexpected EOF",
		},
		{
			name:     "decode failure",
			encoding: "yaml",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("a: [1"))
				assert.NoError(t, err)
			},
			expectedCode:  http.StatusOK,
			expectedError: "failed to parse yaml response: yaml: line 1: did not find expected ',' or ']'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := New(Config{
				HTTPClient: &http.Client{},
				Tracer:     noop.NewTracerProvider().Tracer("test"),
				Logger:     zerolog.Nop(),
			})

			result, err := client.Get(context.Background(), server.URL, tt.encoding, nil)

			require.Error(t, err)
			assert.Equal(t, tt.expectedError, err.Error())
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedCode, result.StatusCode)
			assert.Nil(t, result.Data)
		})
	}
}

func TestClient_Request_RedirectLoop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient: &http.Client{},
		Tracer:     noop.NewTracerProvider().Tracer("test"),
		Logger:     zerolog.Nop(),
	})

	result, err := client.Get(context.Background(), server.URL, "json", nil)

	require.EqualError(t, err, `request failed: Get "/loop": stopped after 10 redirects`)
	require.NotNil(t, result)
	assert.Equal(t, http.StatusFound, result.StatusCode)
	assert.Nil(t, result.Data)
}

func TestClient_Request_TimeoutBeforeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := New(Config{
		HTTPClient: &http.Client{},
		Tracer:     noop.NewTracerProvider().Tracer("test"),
		Logger:     zerolog.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := client.Get(ctx, server.URL, "json", nil)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, result)
}
