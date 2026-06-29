// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

// Package instructions provides embedded claude-instructions.md content
// for MCP server integration, containing comprehensive guidance for AI agents
// working with the github.com/spacelift-io/spacelift-intent server's abstraction layer.
package instructions

import (
	_ "embed"
	"fmt"

	"github.com/spacelift-io/spacelift-intent/allowlist"
)

//go:embed claude-instructions.md
var instructions string

func Get() string {
	return instructions
}

// GetWithAllowlist returns the base instructions with a short summary of the
// configured allowlist appended, so the model is told upfront which providers
// are usable. Returns the base instructions unchanged when no allowlist is set.
func GetWithAllowlist(al *allowlist.Allowlist) string {
	if !al.Enabled() {
		return instructions
	}
	return fmt.Sprintf("%s\n\n## Provider Allowlist\n\nOnly the following providers are permitted by the deployer; calls referencing any other provider will be rejected. Do not attempt to use providers outside this list:\n\n%s\n", instructions, al.Summary())
}
