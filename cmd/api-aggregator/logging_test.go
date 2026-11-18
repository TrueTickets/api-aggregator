// Copyright (c) 2026 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupLogger_SeverityField(t *testing.T) {
	levelFieldName := zerolog.LevelFieldName
	timeFieldFormat := zerolog.TimeFieldFormat
	logger := log.Logger
	t.Cleanup(func() {
		zerolog.LevelFieldName = levelFieldName
		zerolog.TimeFieldFormat = timeFieldFormat
		log.Logger = logger
	})

	setupLogger("json")
	var logBuf bytes.Buffer
	captured := log.Logger.Output(&logBuf)
	captured.Warn().Msg("backend request failed")

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(logBuf.Bytes(), &entry))
	assert.Equal(t, "warn", entry["severity"])
	assert.NotContains(t, entry, "level")
}
