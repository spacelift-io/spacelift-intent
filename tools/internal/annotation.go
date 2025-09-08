// Package internal provides utilities for creating MCP tool annotations with hint flags.
package internal

import "github.com/mark3labs/mcp-go/mcp"

// AnnotationHint represents bit flags for tool annotation hints.
// Multiple hints can be combined using bitwise OR (|).
type AnnotationHint uint8

const (
	// READONLY indicates the tool only reads data and doesn't modify state.
	READONLY AnnotationHint = 1 << iota
	// DESTRUCTIVE indicates the tool may irreversibly modify or delete data.
	DESTRUCTIVE
	// IDEMPOTENT indicates the tool can be safely called multiple times with the same result.
	IDEMPOTENT
	// OPEN_WORLD indicates the tool may access external resources or have side effects.
	OPEN_WORLD
)

// ToolAnnotations creates an MCP tool annotation with the specified title and hint flags.
// Hint flags can be combined using bitwise OR (e.g., READONLY|IDEMPOTENT).
// Pass 0 for hints if no special behavior hints are needed.
func ToolAnnotations(title string, hints AnnotationHint) mcp.ToolAnnotation {
	return mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(hints&READONLY != 0),
		DestructiveHint: mcp.ToBoolPtr(hints&DESTRUCTIVE != 0),
		IdempotentHint:  mcp.ToBoolPtr(hints&IDEMPOTENT != 0),
		OpenWorldHint:   mcp.ToBoolPtr(hints&OPEN_WORLD != 0),
	}
}
