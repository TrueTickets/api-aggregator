// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/TrueTickets/api-aggregator/internal/client"
	"github.com/TrueTickets/api-aggregator/internal/config"
	"github.com/TrueTickets/api-aggregator/internal/merger"
)

const (
	defaultHTTPTimeoutSeconds = 30
	// Minimum size for compression (1KB)
	compressionMinSize = 1024
)

// Server represents the API aggregation server
type Server struct {
	config *config.Config
	router *chi.Mux
	client *client.Client
	merger *merger.Merger
	tracer trace.Tracer
	meter  metric.Meter
	logger zerolog.Logger
}

// Config holds server configuration
type Config struct {
	Config *config.Config
	Tracer trace.Tracer
	Meter  metric.Meter
	Logger zerolog.Logger
}

// New creates a new server instance
func New(cfg Config) *Server {
	s := &Server{
		config: cfg.Config,
		tracer: cfg.Tracer,
		meter:  cfg.Meter,
		logger: cfg.Logger,
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: defaultHTTPTimeoutSeconds * time.Second,
	}
	s.client = client.New(client.Config{
		HTTPClient: httpClient,
		Tracer:     s.tracer,
		Logger:     s.logger,
	})

	// Create merger
	s.merger = merger.New(merger.Config{
		Tracer: s.tracer,
	})

	// Setup routes
	s.setupRoutes()

	return s
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// setupRoutes sets up the router with configured endpoints
func (s *Server) setupRoutes() {
	s.router = chi.NewMux()

	// Add middleware
	s.router.Use(
		middleware.Recoverer,
		s.compressionMiddleware,
		s.loggingMiddleware,
		s.tracingMiddleware,
	)

	// Add catch-all handler for 404s
	s.router.NotFound(s.handleNotFound)

	// Add health check endpoints
	s.router.Get("/livez", s.handleLiveness)
	s.router.Get("/readyz", s.handleReadiness)

	// Add configured endpoints
	for _, endpoint := range s.config.Endpoints {
		handler := s.createEndpointHandler(endpoint)
		s.router.MethodFunc(endpoint.Method, endpoint.Endpoint, handler)

		log.Info().
			Str("endpoint", endpoint.Endpoint).
			Str("method", endpoint.Method).
			Int("backends", len(endpoint.Backends)).
			Msg("Registered endpoint")
	}
}

// handleNotFound handles requests to non-existent endpoints
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": "endpoint not found",
		"path":  r.URL.Path,
	}); err != nil {
		log.Error().Err(err).Msg("Failed to encode 404 response")
	}
}
