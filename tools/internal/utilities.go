// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RespondJSON marshals the input to JSON and returns it as a text content result.
func RespondJSON(input any) (*mcp.CallToolResult, error) {
	result, err := json.Marshal(input)
	if err != nil {
		return NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return NewToolResultText(string(result)), nil
}

// NewToolResultText creates a successful tool result with text content.
func NewToolResultText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// NewToolResultError creates an error tool result with text content.
func NewToolResultError(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: true,
	}
}

// PtrTo returns a pointer to the given value.
func PtrTo[T any](v T) *T { return &v }
