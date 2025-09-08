// Package instructions provides embedded claude-instructions.md content
// for MCP server integration, containing comprehensive guidance for AI agents
// working with the spacelift-intent server's abstraction layer.
package instructions

import (
	_ "embed"
)

//go:embed claude-instructions.md
var instructions string

func Get() string {
	return instructions
}
