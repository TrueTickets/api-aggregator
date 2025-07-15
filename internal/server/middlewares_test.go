// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/TrueTickets/api-aggregator/internal/config"
)

func TestServer_TraceLogging(t *testing.T) {
	// Create mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"users": [{"id": 1, "name": "John"}]}`))
		require.NoError(t, err)
	}))
	defer backendServer.Close()

	tests := []struct {
		name     string
		logLevel zerolog.Level
		expected string
	}{
		{
			name:     "trace level logs aggregated response",
			logLevel: zerolog.TraceLevel,
			expected: "aggregated response body",
		},
		{
			name:     "debug level no aggregated response logging",
			logLevel: zerolog.DebugLevel,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer to capture log output
			var logBuf bytes.Buffer
			logger := zerolog.New(&logBuf).Level(tt.logLevel)

			// Create test config
			cfg := &config.Config{
				Endpoints: []config.Endpoint{
					{
						Endpoint: "/test",
						Method:   "GET",
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
				Meter:  nil,
				Logger: logger,
			})

			// Create test request
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			req = req.WithContext(context.Background())
			w := httptest.NewRecorder()

			// Handle request
			server.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, http.StatusOK, w.Code)

			// Check logging output
			logOutput := logBuf.String()

			if tt.expected != "" {
				assert.Contains(t, logOutput, tt.expected)
				// For trace level, should contain aggregated response
				if tt.logLevel == zerolog.TraceLevel {
					assert.Contains(t, logOutput, "aggregated_response")
					assert.Contains(t, logOutput, "users")
					assert.Contains(t, logOutput, "John")
					assert.Contains(t, logOutput, "/test") // endpoint
				}
			} else {
				// Should not contain aggregated response logging
				assert.NotContains(t, logOutput, "aggregated response body")
			}
		})
	}
}

func TestServer_CompressionMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		acceptEncoding    string
		responseBody      string
		expectCompressed  bool
		expectContentType string
	}{
		{
			name:              "compress large response with gzip accepted",
			acceptEncoding:    "gzip, deflate",
			responseBody:      strings.Repeat("Hello World! ", 100), // > 1KB
			expectCompressed:  true,
			expectContentType: "application/json",
		},
		{
			name:              "no compression when gzip not accepted",
			acceptEncoding:    "deflate",
			responseBody:      strings.Repeat("Hello World! ", 100),
			expectCompressed:  false,
			expectContentType: "application/json",
		},
		{
			name:              "no compression for small response",
			acceptEncoding:    "gzip",
			responseBody:      "Small response",
			expectCompressed:  false,
			expectContentType: "application/json",
		},
		{
			name:              "compress response at exact threshold",
			acceptEncoding:    "gzip",
			responseBody:      strings.Repeat("x", compressionMinSize),
			expectCompressed:  true,
			expectContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock backend server that returns the test response
			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				resp := map[string]string{"data": tt.responseBody}
				err := json.NewEncoder(w).Encode(resp)
				require.NoError(t, err)
			}))
			defer backendServer.Close()

			// Create test config
			cfg := &config.Config{
				Endpoints: []config.Endpoint{
					{
						Endpoint: "/test",
						Method:   "GET",
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
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			w := httptest.NewRecorder()

			// Handle request
			server.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectContentType, w.Header().Get("Content-Type"))

			if tt.expectCompressed {
				// Verify compression headers
				assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
				assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"))
				assert.Empty(t, w.Header().Get("Content-Length"))

				// Decompress and verify content
				reader, err := gzip.NewReader(w.Body)
				require.NoError(t, err)
				defer func() {
					closeErr := reader.Close()
					assert.NoError(t, closeErr)
				}()

				decompressed, err := io.ReadAll(reader)
				require.NoError(t, err)

				var response map[string]string
				err = json.Unmarshal(decompressed, &response)
				require.NoError(t, err)
				assert.Equal(t, tt.responseBody, response["data"])
			} else {
				// Verify no compression headers
				assert.Empty(t, w.Header().Get("Content-Encoding"))

				// Verify content is not compressed
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.responseBody, response["data"])
			}
		})
	}
}

func TestServer_CompressionWithDifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectCompress bool
	}{
		{
			name:           "compress 200 OK response",
			statusCode:     http.StatusOK,
			responseBody:   strings.Repeat("Success ", 200),
			expectCompress: true,
		},
		{
			name:           "compress 404 error response",
			statusCode:     http.StatusNotFound,
			responseBody:   strings.Repeat("Not Found ", 200),
			expectCompress: true,
		},
		{
			name:           "compress 500 error response",
			statusCode:     http.StatusInternalServerError,
			responseBody:   strings.Repeat("Internal Error ", 200),
			expectCompress: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock backend server
			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				resp := map[string]string{"message": tt.responseBody}
				err := json.NewEncoder(w).Encode(resp)
				require.NoError(t, err)
			}))
			defer backendServer.Close()

			// Create test config
			cfg := &config.Config{
				Endpoints: []config.Endpoint{
					{
						Endpoint: "/test",
						Method:   "GET",
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

			// Create test request with gzip support
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			req.Header.Set("Accept-Encoding", "gzip")
			w := httptest.NewRecorder()

			// Handle request
			server.ServeHTTP(w, req)

			// Status codes >= 400 from backends result in 500 from aggregator
			expectedStatus := http.StatusOK
			if tt.statusCode >= 400 {
				expectedStatus = http.StatusInternalServerError
			}
			assert.Equal(t, expectedStatus, w.Code)

			if tt.expectCompress && tt.statusCode < 400 {
				// Verify compression
				assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

				// Decompress and verify
				reader, err := gzip.NewReader(w.Body)
				require.NoError(t, err)
				defer func() {
					closeErr := reader.Close()
					assert.NoError(t, closeErr)
				}()

				decompressed, err := io.ReadAll(reader)
				require.NoError(t, err)

				var response map[string]string
				err = json.Unmarshal(decompressed, &response)
				require.NoError(t, err)
				assert.Equal(t, tt.responseBody, response["message"])
			}
		})
	}
}

func TestServer_CompressionPerformance(t *testing.T) {
	// Create large response data
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = strings.Repeat("value ", 10)
	}

	// Create mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(largeData)
		require.NoError(t, err)
	}))
	defer backendServer.Close()

	// Create test config
	cfg := &config.Config{
		Endpoints: []config.Endpoint{
			{
				Endpoint: "/test",
				Method:   "GET",
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

	// Test with compression
	reqWithGzip := httptest.NewRequest("GET", "/test", http.NoBody)
	reqWithGzip.Header.Set("Accept-Encoding", "gzip")
	wWithGzip := httptest.NewRecorder()
	server.ServeHTTP(wWithGzip, reqWithGzip)

	// Test without compression
	reqNoGzip := httptest.NewRequest("GET", "/test", http.NoBody)
	wNoGzip := httptest.NewRecorder()
	server.ServeHTTP(wNoGzip, reqNoGzip)

	// Verify compressed response is smaller
	assert.Equal(t, http.StatusOK, wWithGzip.Code)
	assert.Equal(t, http.StatusOK, wNoGzip.Code)
	assert.Equal(t, "gzip", wWithGzip.Header().Get("Content-Encoding"))
	assert.Empty(t, wNoGzip.Header().Get("Content-Encoding"))

	compressedSize := wWithGzip.Body.Len()
	uncompressedSize := wNoGzip.Body.Len()

	t.Logf("Uncompressed size: %d bytes", uncompressedSize)
	t.Logf("Compressed size: %d bytes", compressedSize)
	t.Logf("Compression ratio: %.2f%%", float64(compressedSize)/float64(uncompressedSize)*100)

	// Verify significant compression
	assert.Less(t, compressedSize, uncompressedSize/2, "Compressed size should be less than 50% of uncompressed")
}
