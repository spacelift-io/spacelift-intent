// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package internal

// ToolInputSchema represents a JSON Schema for tool arguments.
// This struct marshals to a JSON Schema compatible format.
type ToolInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}
