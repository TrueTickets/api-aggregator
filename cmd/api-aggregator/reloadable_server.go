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
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/TrueTickets/api-aggregator/internal/config"
	"github.com/TrueTickets/api-aggregator/internal/server"
	"github.com/TrueTickets/api-aggregator/internal/telemetry"
)

// reloadableServer wraps the server with config reloading capability
type reloadableServer struct {
	mu         sync.RWMutex
	server     *server.Server
	cfg        *config.Config
	tel        *telemetry.Provider
	configPath string
}

// newReloadableServer creates a new reloadable server
func newReloadableServer(cfg *config.Config, tel *telemetry.Provider, configPath string) *reloadableServer {
	srv := server.New(server.Config{
		Config: cfg,
		Tracer: tel.Tracer(),
		Meter:  tel.Meter(),
		Logger: log.Logger,
	})

	return &reloadableServer{
		server:     srv,
		cfg:        cfg,
		tel:        tel,
		configPath: configPath,
	}
}

// ServeHTTP implements http.Handler by delegating to the current server
func (rs *reloadableServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	rs.server.ServeHTTP(w, r)
}

// reload reloads the configuration and recreates the server
func (rs *reloadableServer) reload() error {
	log.Info().Msg("Reloading configuration...")

	// Load new config
	newCfg, err := config.LoadConfig(rs.configPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to reload configuration")
		return err
	}

	// Update log format and level
	setupLogger(newCfg.LogFormat)
	setupLogLevel(newCfg.LogLevel)

	// Create new server with updated config
	newServer := server.New(server.Config{
		Config: newCfg,
		Tracer: rs.tel.Tracer(),
		Meter:  rs.tel.Meter(),
		Logger: log.Logger,
	})

	// Update server atomically
	rs.mu.Lock()
	rs.server = newServer
	rs.cfg = newCfg
	rs.mu.Unlock()

	log.Info().Msg("Configuration reloaded successfully")
	return nil
}
