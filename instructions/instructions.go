// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

// Package instructions provides embedded claude-instructions.md content
// for MCP server integration, containing comprehensive guidance for AI agents
// working with the github.com/spacelift-io/spacelift-intent server's abstraction layer.
package instructions

import (
	_ "embed"
)

//go:embed claude-instructions.md
var instructions string

func Get() string {
	return instructions
}
