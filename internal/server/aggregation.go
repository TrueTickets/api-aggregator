// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/TrueTickets/api-aggregator/internal/client"
	"github.com/TrueTickets/api-aggregator/internal/config"
	"github.com/TrueTickets/api-aggregator/internal/types"
)

// createEndpointHandler creates a handler for a configured endpoint
func (s *Server) createEndpointHandler(endpoint config.Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Create context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, endpoint.Timeout)
		defer cancel()

		// Extract path parameters, and aggregate responses
		routeCtx := chi.RouteContext(ctx)
		pathParams := make(map[string]string)
		for i, key := range routeCtx.URLParams.Keys {
			pathParams[key] = routeCtx.URLParams.Values[i]
		}
		responses := s.aggregateBackends(timeoutCtx, endpoint, pathParams, r)

		// Handle case where all backends failed
		if !s.hasSuccessfulResponse(responses) {
			s.writeErrorResponse(w, "All backends failed")
			return
		}

		// Merge responses and write success response
		s.processMergedResponse(w, endpoint, responses)
	}
}

// hasSuccessfulResponse checks if any backend response was successful
func (s *Server) hasSuccessfulResponse(responses []types.BackendResponse) bool {
	for _, resp := range responses {
		if resp.Error == nil {
			return true
		}
	}
	return false
}

// writeErrorResponse writes an error response with the given message
func (s *Server) writeErrorResponse(w http.ResponseWriter, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": errorMsg,
	}); err != nil {
		log.Error().Err(err).Msg("Failed to encode error response")
	}
}

// processMergedResponse merges backend responses and writes the success response
func (s *Server) processMergedResponse(
	w http.ResponseWriter,
	endpoint config.Endpoint,
	responses []types.BackendResponse,
) {
	// Merge responses
	mergedData, allCompleted := s.merger.Merge(responses)

	// Log aggregated response at trace level
	s.logAggregatedResponse(endpoint, mergedData, allCompleted)

	// Set response headers
	w.Header().Set("X-API-Aggregation-Completed", fmt.Sprintf("%t", allCompleted))
	w.Header().Set("Content-Type", "application/json")

	// Write response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(mergedData); err != nil {
		log.Error().Err(err).Msg("Failed to encode merged response")
	}
}

// logAggregatedResponse logs the aggregated response at trace level
func (s *Server) logAggregatedResponse(endpoint config.Endpoint, mergedData interface{}, allCompleted bool) {
	if s.logger.GetLevel() <= zerolog.TraceLevel {
		if responseBytes, err := json.Marshal(mergedData); err == nil {
			s.logger.Trace().
				Str("endpoint", endpoint.Endpoint).
				Str("method", endpoint.Method).
				Bool("all_completed", allCompleted).
				RawJSON("aggregated_response", responseBytes).
				Msg("aggregated response body")
		}
	}
}

// aggregateBackends makes requests to all backends concurrently
func (s *Server) aggregateBackends(
	ctx context.Context,
	endpoint config.Endpoint,
	pathParams map[string]string,
	r *http.Request,
) []types.BackendResponse {

	// Read request body once if needed
	var bodyBytes []byte
	var err error
	if s.shouldForwardBody(r.Method) && r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to read request body")
			// Continue without body rather than failing the entire request
		}
		if err := r.Body.Close(); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to close request body")
		}
	}

	var wg sync.WaitGroup
	responses := make([]types.BackendResponse, len(endpoint.Backends))

	for i, backend := range endpoint.Backends {
		wg.Add(1)
		go func(idx int, be config.Backend) {
			defer wg.Done()

			// Build URL by replacing path parameters
			url := s.buildURL(be, pathParams)

			// Create body reader for each backend
			var body io.Reader
			if len(bodyBytes) > 0 {
				body = bytes.NewReader(bodyBytes)
			}

			// Make request
			data, err := s.client.Request(ctx, client.RequestConfig{
				Method:   endpoint.Method,
				URL:      url,
				Encoding: be.Encoding,
				Headers:  s.processHeaders(r, be.RemoveHeaders),
				Body:     body,
			})

			responses[idx] = types.BackendResponse{
				Backend: be,
				Data:    data,
				Error:   err,
			}
		}(i, backend)
	}

	wg.Wait()
	return responses
}

// shouldForwardBody determines if request body should be forwarded based on HTTP method
func (s *Server) shouldForwardBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

// buildURL builds the full URL for a backend request
func (s *Server) buildURL(backend config.Backend, pathParams map[string]string) string {
	baseURL := backend.Host
	urlPattern := backend.URLPattern

	// Replace path parameters
	for key, value := range pathParams {
		placeholder := "{" + key + "}"
		urlPattern = strings.ReplaceAll(urlPattern, placeholder, value)
	}

	// Combine base URL and pattern
	if strings.HasSuffix(baseURL, "/") && strings.HasPrefix(urlPattern, "/") {
		return baseURL + urlPattern[1:]
	} else if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(urlPattern, "/") {
		return baseURL + "/" + urlPattern
	}
	return baseURL + urlPattern
}

// processHeaders processes headers from the original request, forwarding all and removing specified ones
func (s *Server) processHeaders(r *http.Request, removeHeaders []string) map[string]string {
	headers := make(map[string]string)

	// Create a set of headers to remove for efficient lookup
	removeHeadersSet := make(map[string]bool)
	for _, header := range removeHeaders {
		removeHeadersSet[strings.ToLower(header)] = true
	}

	// Forward all headers from the original request
	for name, values := range r.Header {
		// Skip headers that should be removed
		if removeHeadersSet[strings.ToLower(name)] {
			continue
		}

		// Skip headers that shouldn't be forwarded to backends
		switch strings.ToLower(name) {
		case "host", "content-length", "transfer-encoding", "connection", "upgrade", "accept-encoding":
			continue
		}

		// Use the first value if multiple values exist
		if len(values) > 0 {
			headers[name] = values[0]
		}
	}

	return headers
}
