package policy

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type deleteArgs struct {
	ID string `json:"id"`
}

// Delete deletes a policy
func Delete(storage types.Storage) (mcp.Tool, i.ToolHandler) {
	tool := mcp.Tool{
		Name:        string("policy-delete"),
		Description: "Delete an OPA/Rego policy. MEDIUM risk operation that removes governance controls and compliance checks. Use with caution - removing policies can disable important safety validations that prevent configuration violations. Verify no active enforcement before deletion.",
		Annotations: i.ToolAnnotations("Delete policy", i.DESTRUCTIVE),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Policy ID to delete",
				},
			},
			Required: []string{"id"},
		},
	}

	handler := mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args deleteArgs) (*mcp.CallToolResult, error) {

		// Check if policy exists
		policy, err := storage.GetPolicy(ctx, args.ID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to check policy existence: %v", err)), nil
		}

		if policy == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Policy not found: %s", args.ID)), nil
		}

		// Delete policy
		if err := storage.DeletePolicy(ctx, args.ID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete policy: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Policy deleted successfully: %s (%s)", policy.Name, args.ID)), nil
	})

	return tool, handler
}
