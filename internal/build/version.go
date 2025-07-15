// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

// Package build contains build-time information injected during compilation
package build

// ServiceVersion contains the version of the service, injected at build time
//
//nolint:gochecknoglobals // Version needs to be global for build-time injection
var ServiceVersion = "dev"
