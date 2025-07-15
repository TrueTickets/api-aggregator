// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package types

import "github.com/TrueTickets/api-aggregator/internal/config"

// BackendResponse represents a response from a backend service
type BackendResponse struct {
	Backend config.Backend
	Data    interface{}
	Error   error
}
