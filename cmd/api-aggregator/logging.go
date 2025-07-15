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
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogger(format string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if format == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		// Default to JSON format
		log.Logger = log.Output(os.Stderr)
	}
}

func setupLogLevel(logLevel string) {
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Warn().Err(err).Str("level", logLevel).Msg("Invalid log level, using info")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
}
