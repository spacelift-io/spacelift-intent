// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewTypedToolHandler creates a ToolHandler that automatically unmarshals
// the request arguments into the specified type T.
func NewTypedToolHandler[T any](handler func(ctx context.Context, req *mcp.CallToolRequest, args T) (*mcp.CallToolResult, error)) ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args T
		if req.Params.Arguments != nil {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return NewToolResultError(fmt.Sprintf("failed to bind arguments: %v", err)), nil
			}
		}
		return handler(ctx, req, args)
	}
}
