package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type untrackArgs struct {
	ResourceID string `json:"resource_id"`
}

func Untrack(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("lifecycle-resources-untrack"),
		Description: "Remove a resource from the state - stop managing lifecycle without deleting the actual resource. MEDIUM risk operation that removes MCP management while preserving actual infrastructure. Use this to transition resources back to manual management or transfer to different management systems. Critical Safety Protocol: verify no dependents exist before untracking to avoid breaking dependency chains.",
		Annotations: i.ToolAnnotations("Remove a resource from the state", i.DESTRUCTIVE|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "The unique identifier of the resource to untrack from state",
				},
			},
			Required: []string{"resource_id"},
		},
	}, Handler: untrack(storage)}
}

func untrack(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args untrackArgs) (*mcp.CallToolResult, error) {
		// Check if the resource exists in state
		record, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get state: %v", err)), nil
		}

		if record == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' not found in state", args.ResourceID)), nil
		}

		// Parse the stored state for history
		var state map[string]any
		json.Unmarshal([]byte(record.State), &state)

		// Add operation context for automatic history tracking
		ctx = context.WithValue(ctx, types.OperationContextKey, "untrack")
		ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

		// Remove the state from database (dependencies automatically cleaned up via CASCADE)
		if err := storage.DeleteState(ctx, args.ResourceID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to untrack resource from state: %v", err)), nil
		}

		return i.RespondJSON(map[string]any{
			"resource_id":   args.ResourceID,
			"status":        "untracked",
			"message":       fmt.Sprintf("Resource '%s' untracked from state (actual resource preserved)", args.ResourceID),
			"provider":      record.Provider,
			"resource_type": record.ResourceType,
		})
	})
}
