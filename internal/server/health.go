// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// handleLiveness handles liveness probe requests
func (s *Server) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	s.writeHealthResponse(w, "live")
}

// handleReadiness handles readiness probe requests
func (s *Server) handleReadiness(w http.ResponseWriter, _ *http.Request) {
	s.writeHealthResponse(w, "ready")
}

// writeHealthResponse writes a health check response
func (s *Server) writeHealthResponse(w http.ResponseWriter, checkType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"check":     checkType,
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}); err != nil {
		log.Error().Err(err).Msg("Failed to encode health response")
	}
}
