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

package state

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type ejectArgs struct {
	ResourceID string `json:"resource_id"`
}

func Eject(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("state-eject"),
		Description: "Remove a resource from the state - stop managing lifecycle without deleting the actual resource. MEDIUM risk operation that removes MCP management while preserving actual infrastructure. Use this to transition resources back to manual management or transfer to different management systems. Critical Safety Protocol: verify no dependents exist before ejection to avoid breaking dependency chains.",
		Annotations: i.ToolAnnotations("Remove a resource from the state", i.Destructive|i.Idempotent),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "The unique identifier of the resource to eject from state",
				},
			},
			Required: []string{"resource_id"},
		},
	}, Handler: eject(storage)}
}

func eject(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args ejectArgs) (*mcp.CallToolResult, error) {
		// Check if the resource exists in state
		record, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get state: %v", err)), nil
		}

		if record == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' not found in state", args.ResourceID)), nil
		}

		// Add operation context for automatic history tracking
		ctx = context.WithValue(ctx, types.OperationContextKey, "eject")
		ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

		// Remove the state from database (dependencies automatically cleaned up via CASCADE)
		if err := storage.DeleteState(ctx, args.ResourceID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to eject resource from state: %v", err)), nil
		}

		return i.RespondJSON(map[string]any{
			"resource_id":   args.ResourceID,
			"status":        "ejected",
			"message":       fmt.Sprintf("Resource '%s' ejected from state (actual resource preserved)", args.ResourceID),
			"provider":      record.Provider,
			"resource_type": record.ResourceType,
		})
	})
}
