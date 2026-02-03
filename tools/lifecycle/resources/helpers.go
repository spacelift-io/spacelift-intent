// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

func newResourceOperation(input types.ResourceOperationInput) (types.ResourceOperation, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return types.ResourceOperation{}, fmt.Errorf("failed to generate operation ID: %v", err)
	}

	return types.ResourceOperation{
		ID:                     id.String(),
		ResourceOperationInput: input,
	}, nil
}

// RespondJSON returns a JSON response
func RespondJSON(response map[string]any) (*mcp.CallToolResult, error) {
	result, err := json.Marshal(response)
	if err != nil {
		return i.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return i.NewToolResultText(string(result)), nil
}

// NewToolResultText creates a text content result (convenience wrapper).
func NewToolResultText(text string) *mcp.CallToolResult {
	return i.NewToolResultText(text)
}
