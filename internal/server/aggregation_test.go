// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/TrueTickets/api-aggregator/internal/config"
)

func TestServer_ProcessHeaders(t *testing.T) {
	s := &Server{}

	tests := []struct {
		name           string
		requestHeaders map[string]string
		removeHeaders  []string
		expected       map[string]string
	}{
		{
			name: "forward all headers when no removal specified",
			requestHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "test-agent",
				"X-Custom":      "custom-value",
				"Content-Type":  "application/json",
			},
			removeHeaders: nil,
			expected: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "test-agent",
				"X-Custom":      "custom-value",
				"Content-Type":  "application/json",
			},
		},
		{
			name: "remove specific headers",
			requestHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "test-agent",
				"X-Custom":      "custom-value",
				"Content-Type":  "application/json",
			},
			removeHeaders: []string{"Authorization", "X-Custom"},
			expected: map[string]string{
				"User-Agent":   "test-agent",
				"Content-Type": "application/json",
			},
		},
		{
			name: "remove headers case insensitive",
			requestHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "test-agent",
				"X-Custom":      "custom-value",
			},
			removeHeaders: []string{"authorization", "x-custom"},
			expected: map[string]string{
				"User-Agent": "test-agent",
			},
		},
		{
			name: "skip system headers automatically",
			requestHeaders: map[string]string{
				"Authorization":     "Bearer token123",
				"Host":              "example.com",
				"Content-Length":    "100",
				"Transfer-Encoding": "chunked",
				"Connection":        "keep-alive",
				"Upgrade":           "websocket",
				"Accept-Encoding":   "gzip, deflate",
			},
			removeHeaders: nil,
			expected: map[string]string{
				"Authorization": "Bearer token123",
			},
		},
		{
			name: "handle non-existent headers in remove list",
			requestHeaders: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "test-agent",
			},
			removeHeaders: []string{"Non-Existent", "Another-Missing"},
			expected: map[string]string{
				"Authorization": "Bearer token123",
				"User-Agent":    "test-agent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request with headers
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			for key, value := range tt.requestHeaders {
				req.Header.Set(key, value)
			}

			// Process headers
			result := s.processHeaders(req, tt.removeHeaders)

			// Verify result
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServer_ShouldForwardBody(t *testing.T) {
	s := &Server{}

	tests := []struct {
		method   string
		expected bool
	}{
		{http.MethodPost, true},
		{http.MethodPut, true},
		{http.MethodPatch, true},
		{http.MethodGet, false},
		{http.MethodDelete, false},
		{http.MethodHead, false},
		{http.MethodOptions, false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := s.shouldForwardBody(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServer_RequestBodyForwarding(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		requestBody        string
		expectedStatusCode int
		verifyBody         bool
	}{
		{
			name:               "POST request with JSON body",
			method:             http.MethodPost,
			requestBody:        `{"name": "test", "value": 123}`,
			expectedStatusCode: http.StatusOK,
			verifyBody:         true,
		},
		{
			name:               "PUT request with JSON body",
			method:             http.MethodPut,
			requestBody:        `{"id": 1, "name": "updated"}`,
			expectedStatusCode: http.StatusOK,
			verifyBody:         true,
		},
		{
			name:               "PATCH request with JSON body",
			method:             http.MethodPatch,
			requestBody:        `{"field": "value"}`,
			expectedStatusCode: http.StatusOK,
			verifyBody:         true,
		},
		{
			name:               "GET request with no body",
			method:             http.MethodGet,
			requestBody:        "",
			expectedStatusCode: http.StatusOK,
			verifyBody:         false,
		},
		{
			name:               "DELETE request with no body",
			method:             http.MethodDelete,
			requestBody:        "",
			expectedStatusCode: http.StatusOK,
			verifyBody:         false,
		},
		{
			name:               "POST request with empty body",
			method:             http.MethodPost,
			requestBody:        "",
			expectedStatusCode: http.StatusOK,
			verifyBody:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRequestBodyForwarding(t, tt.method, tt.requestBody, tt.expectedStatusCode, tt.verifyBody)
		})
	}
}

// testRequestBodyForwarding is a helper function to test request body forwarding
func testRequestBodyForwarding(t *testing.T, method, requestBody string, expectedStatusCode int, verifyBody bool) {
	var receivedBody []byte
	var receivedContentType string

	// Create mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the received body
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			receivedBody = body
		}
		receivedContentType = r.Header.Get("Content-Type")

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status": "success"}`))
		require.NoError(t, err)
	}))
	defer backendServer.Close()

	// Create server
	cfg := createTestConfig(method, backendServer.URL)
	server := createTestServer(cfg)

	// Create test request
	var body io.Reader
	if requestBody != "" {
		body = strings.NewReader(requestBody)
	}
	req := httptest.NewRequest(method, "/test", body)
	if requestBody != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()

	// Handle request
	server.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, expectedStatusCode, w.Code)

	// Verify body forwarding
	if verifyBody && requestBody != "" {
		assert.Equal(t, requestBody, string(receivedBody))
		assert.Equal(t, "application/json", receivedContentType)
	} else if !verifyBody {
		assert.Empty(t, receivedBody)
	}
}

func TestServer_ConcurrentBodyForwarding(t *testing.T) {
	requestBody := `{"shared": "data", "count": 42}`
	var mu sync.Mutex
	receivedBodies := make(map[string]string)

	// Create multiple mock backend servers
	backendCount := 3
	backends := make([]config.Backend, backendCount)
	backendServers := make([]*httptest.Server, backendCount)

	// Setup all backend servers
	for i := 0; i < backendCount; i++ {
		backendID := i
		backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Capture the received body
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			mu.Lock()
			receivedBodies[fmt.Sprintf("backend-%d", backendID)] = string(body)
			mu.Unlock()

			// Return success response with backend ID
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"backend": backendID,
				"status":  "success",
			}
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		backendServers[i] = backendServer

		backends[i] = config.Backend{
			Host:       backendServer.URL,
			URLPattern: "/test",
			Encoding:   "json",
		}
	}

	// Clean up all servers at the end
	defer func() {
		for _, server := range backendServers {
			server.Close()
		}
	}()

	// Create test config with multiple backends
	cfg := &config.Config{
		Endpoints: []config.Endpoint{
			{
				Endpoint: "/test",
				Method:   http.MethodPost,
				Timeout:  5 * time.Second,
				Encoding: "json",
				Backends: backends,
			},
		},
	}

	// Create server
	server := New(Config{
		Config: cfg,
		Tracer: noop.NewTracerProvider().Tracer("test"),
		Logger: zerolog.Nop(),
	})

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify all backends received the same body
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, receivedBodies, backendCount)
	for _, body := range receivedBodies {
		assert.Equal(t, requestBody, body)
	}

	// Verify aggregated response contains all backend responses
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response)
}

func TestServer_BodyForwardingWithLargeBody(t *testing.T) {
	// Create a large body (1MB)
	largeBody := strings.Repeat("x", 1024*1024)
	requestBody := `{"data": "` + largeBody + `"}`

	var receivedBody []byte

	// Create mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the received body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBody = body

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"status": "success"}`))
		require.NoError(t, err)
	}))
	defer backendServer.Close()

	// Create test config
	cfg := &config.Config{
		Endpoints: []config.Endpoint{
			{
				Endpoint: "/test",
				Method:   http.MethodPost,
				Timeout:  5 * time.Second,
				Encoding: "json",
				Backends: []config.Backend{
					{
						Host:       backendServer.URL,
						URLPattern: "/test",
						Encoding:   "json",
					},
				},
			},
		},
	}

	// Create server
	server := New(Config{
		Config: cfg,
		Tracer: noop.NewTracerProvider().Tracer("test"),
		Logger: zerolog.Nop(),
	})

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	server.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify body was forwarded correctly
	assert.Equal(t, requestBody, string(receivedBody))
}

func TestServer_ConcatFunctionality(t *testing.T) {
	// Create multiple mock backend servers
	backendServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"id": 1, "name": "Item 1"}`))
		require.NoError(t, err)
	}))
	defer backendServer1.Close()

	backendServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"id": 2, "name": "Item 2"}`))
		require.NoError(t, err)
	}))
	defer backendServer2.Close()

	// Create test config with concat functionality
	cfg := &config.Config{
		Endpoints: []config.Endpoint{
			{
				Endpoint: "/test",
				Method:   "GET",
				Timeout:  5 * time.Second,
				Encoding: "json",
				Backends: []config.Backend{
					{
						Host:       backendServer1.URL,
						URLPattern: "/test",
						Encoding:   "json",
						Concat:     "items",
					},
					{
						Host:       backendServer2.URL,
						URLPattern: "/test",
						Encoding:   "json",
						Concat:     "items",
					},
				},
			},
		},
	}

	// Create server
	server := New(Config{
		Config: cfg,
		Tracer: noop.NewTracerProvider().Tracer("test"),
		Logger: zerolog.Nop(),
	})

	// Create test request
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	// Handle request
	server.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check that items array exists and contains both responses
	items, exists := response["items"]
	assert.True(t, exists, "items key should exist in response")

	itemsArray, ok := items.([]interface{})
	assert.True(t, ok, "items should be an array")
	assert.Len(t, itemsArray, 2, "items array should contain 2 elements")

	// Verify the content of the items
	item1, ok := itemsArray[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(1), item1["id"])
	assert.Equal(t, "Item 1", item1["name"])

	item2, ok := itemsArray[1].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(2), item2["id"])
	assert.Equal(t, "Item 2", item2["name"])
}

// createTestConfig creates a test configuration for request body forwarding tests
func createTestConfig(method, backendURL string) *config.Config {
	return &config.Config{
		Endpoints: []config.Endpoint{
			{
				Endpoint: "/test",
				Method:   method,
				Timeout:  5 * time.Second,
				Encoding: "json",
				Backends: []config.Backend{
					{
						Host:       backendURL,
						URLPattern: "/test",
						Encoding:   "json",
					},
				},
			},
		},
	}
}

// createTestServer creates a test server instance
func createTestServer(cfg *config.Config) *Server {
	return New(Config{
		Config: cfg,
		Tracer: noop.NewTracerProvider().Tracer("test"),
		Logger: zerolog.Nop(),
	})
}
