package resources

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type operationsArgs struct {
	ResourceID   *string `json:"resource_id"`
	Provider     *string `json:"provider"`
	ResourceType *string `json:"resource_type"`
}

func Operations(storage types.Storage) (mcp.Tool, i.ToolHandler) {
	return mcp.Tool{
		Name: string("lifecycle-resources-operations"),
		Description: "List operations performed on resources with optional filtering by resource ID, " +
			"provider, or resource type. Essential for Discovery Phase - use this to review " +
			"infrastructure operation history and understand what actions have been performed " +
			"on managed resources. LOW risk read-only operation for auditing and analysis. " +
			"\n\nPresentation: Present operation history with clear status indicators " +
			"(SUCCEEDED/FAILED), timestamps, and policy evaluation results (allow/deny). " +
			"Include both human-readable summary and JSON format for programmatic access. " +
			"\n\nCritical for tracking resource lifecycle events and maintaining operational " +
			"visibility of infrastructure changes.",
		Annotations: i.ToolAnnotations("List operations on resources", i.OPEN_WORLD),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "Filter by resource unique identifier, if not provided, all resources will be returned",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Filter by provider name (e.g., 'hashicorp/aws', 'hashicorp/random'), if not provided, all resources will be returned",
				},
				"resource_type": map[string]any{
					"type":        "string",
					"description": "Filter by resource type (e.g., 'random_string', 'aws_instance'), if not provided, all resources will be returned",
				},
			},
			Required: []string{},
		},
	}, operations(storage)
}

func operations(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args operationsArgs) (*mcp.CallToolResult, error) {

		operations, err := storage.ListResourceOperations(ctx, types.ResourceOperationsArgs{
			ResourceID:   args.ResourceID,
			Provider:     args.Provider,
			ResourceType: args.ResourceType,
		})

		outputJSON, err := json.Marshal(operations)
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal operations: " + err.Error()), nil
		}

		output := "Operations:\n"
		for _, op := range operations {
			output += "- ID: " + op.ID + ", created at: " + op.CreatedAt + "\n"
			output += "  - Resource ID: " + op.ResourceID + "\n"
			output += "  - Resource Type: " + op.ResourceType + "\n"
			output += "  - Provider: " + op.Provider + "\n"
			output += "  - Operation: " + op.Operation + "\n"
			if op.Failed != nil && *op.Failed != "" {
				output += "  - FAILED: " + *op.Failed + "\n"
			} else {
				output += "  - SUCCEEDED\n"
			}
			if len(op.Allow) > 0 {
				output += "  - Allow:\n"
				for _, allow := range op.Allow {
					output += "    - " + allow + "\n"
				}
			}
			if len(op.Deny) > 0 {
				output += "  - Deny:\n"
				for _, deny := range op.Deny {
					output += "    - " + deny + "\n"
				}
			}

			output += "\n"
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: output,
				},
				mcp.TextContent{
					Type: "text",
					Text: "JSON format:\n" + string(outputJSON),
				},
			},
		}, nil
	})
}
