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
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/TrueTickets/api-aggregator/internal/config"
	"github.com/TrueTickets/api-aggregator/internal/telemetry"
)

const (
	shutdownTimeoutSeconds  = 5
	httpReadTimeoutSeconds  = 5
	httpWriteTimeoutSeconds = 30
	httpIdleTimeoutSeconds  = 120
	httpMaxHeaderBytesShift = 20 // 1MB
)

func main() {
	// Get config path for reloading
	configPath := os.Getenv("API_AGGREGATOR_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Load configuration
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize logger with format and level
	setupLogger(cfg.LogFormat)
	setupLogLevel(cfg.LogLevel)

	// Initialize telemetry
	tel := initializeTelemetry(cfg)
	defer shutdownTelemetry(tel)

	// Create and start server with reloading capability
	runReloadableServer(cfg, tel, configPath)
}

func runReloadableServer(cfg *config.Config, tel *telemetry.Provider, configPath string) {
	// Create reloadable server
	reloadableSrv := newReloadableServer(cfg, tel, configPath)

	// Create HTTP server
	httpServer := createHTTPServer(cfg, reloadableSrv)

	// Start server and wait for shutdown with config reloading
	startServerAndWaitWithReload(httpServer, cfg, reloadableSrv)
}

func createHTTPServer(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: httpReadTimeoutSeconds * time.Second,
		WriteTimeout:      httpWriteTimeoutSeconds * time.Second,
		IdleTimeout:       httpIdleTimeoutSeconds * time.Second,
		MaxHeaderBytes:    1 << httpMaxHeaderBytesShift, // 1MB
	}
}

func startServerAndWaitWithReload(httpServer *http.Server, cfg *config.Config, reloadableSrv *reloadableServer) {
	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Info().Str("port", cfg.Port).Msg("Starting API aggregator server")
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGHUP:
				log.Info().Msg("Reload signal received")
				if err := reloadableSrv.reload(); err != nil {
					log.Error().Err(err).Msg("Failed to reload configuration")
				}
			case os.Interrupt, syscall.SIGTERM:
				log.Info().Msg("Shutdown signal received")
				gracefulShutdown(httpServer, cfg)
				return
			}
		case err := <-errChan:
			log.Fatal().Err(err).Msg("Server error")
		}
	}
}

func gracefulShutdown(httpServer *http.Server, cfg *config.Config) {
	log.Info().Msg("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	log.Info().Msg("Server stopped")
}
