// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

// Package internal provides utilities for creating MCP tool annotations with hint flags.
package internal

import "github.com/modelcontextprotocol/go-sdk/mcp"

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
//
// Note: ReadOnlyHint and IdempotentHint are bool, while DestructiveHint and OpenWorldHint
// are *bool per the MCP SDK's ToolAnnotations struct definition. This asymmetry is intentional
// as the SDK uses pointer types for optional hints that default to unknown/unspecified.
func ToolAnnotations(title string, hints AnnotationHint) mcp.ToolAnnotations {
	return mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    hints&Readonly != 0,
		DestructiveHint: PtrTo(hints&Destructive != 0),
		IdempotentHint:  hints&Idempotent != 0,
		OpenWorldHint:   PtrTo(hints&OpenWorld != 0),
	}
}
