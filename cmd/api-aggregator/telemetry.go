// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT
//
// This code is licensed under MIT license (see LICENSE for details)

// Copyright (c) True Tickets, Inc.
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This file is part of API Aggregator.
//
// API Aggregator is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
//
// API Aggregator is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along with API Aggregator. If not, see <https://www.gnu.org/licenses/>.
package main

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/TrueTickets/api-aggregator/internal/config"
	"github.com/TrueTickets/api-aggregator/internal/telemetry"
)

func initializeTelemetry(cfg *config.Config) *telemetry.Provider {
	tel, err := telemetry.NewProvider(telemetry.Config{
		ServiceName:     cfg.ServiceName,
		TracingEnabled:  cfg.TracingEnabled,
		TracingEndpoint: cfg.TracingEndpoint,
		MetricsEnabled:  cfg.MetricsEnabled,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize telemetry")
	}
	return tel
}

func shutdownTelemetry(tel *telemetry.Provider) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer cancel()
	if err := tel.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown telemetry")
	}
}
