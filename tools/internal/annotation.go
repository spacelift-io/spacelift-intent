// Copyright 2025 Spacelift, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
