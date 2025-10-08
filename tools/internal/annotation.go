// Package internal provides utilities for creating MCP tool annotations with hint flags.
package internal

import "github.com/mark3labs/mcp-go/mcp"

// AnnotationHint represents bit flags for tool annotation hints.
// Multiple hints can be combined using bitwise OR (|).
type AnnotationHint uint8

const (
	// Readonly indicates the tool only reads data and doesn't modify state.
	Readonly AnnotationHint = 1 << iota
	// Destructive indicates the tool may irreversibly modify or delete data.
	Destructive
	// Idempotent indicates the tool can be safely called multiple times with the same result.
	Idempotent
	// OpenWorld indicates the tool may access external resources or have side effects.
	OpenWorld
)

// ToolAnnotations creates an MCP tool annotation with the specified title and hint flags.
// Hint flags can be combined using bitwise OR (e.g., READONLY|IDEMPOTENT).
// Pass 0 for hints if no special behavior hints are needed.
func ToolAnnotations(title string, hints AnnotationHint) mcp.ToolAnnotation {
	return mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(hints&Readonly != 0),
		DestructiveHint: mcp.ToBoolPtr(hints&Destructive != 0),
		IdempotentHint:  mcp.ToBoolPtr(hints&Idempotent != 0),
		OpenWorldHint:   mcp.ToBoolPtr(hints&OpenWorld != 0),
	}
}
