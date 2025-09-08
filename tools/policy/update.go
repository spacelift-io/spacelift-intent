package policy

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type updateArgs struct {
	ID          string  `json:"id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	RegoCode    *string `json:"rego_code"`
	Enabled     *bool   `json:"enabled"`
}

// Update updates an existing policy
func Update(storage types.Storage, engine types.PolicyEngine) (mcp.Tool, i.ToolHandler) {
	return mcp.Tool{
		Name:        string("policy-update"),
		Description: "Update an existing OPA/Rego policy. MEDIUM risk operation that modifies governance controls and compliance checks. Validates updated policy syntax before applying changes. Use with caution - policy changes can affect enforcement behavior for all resource operations.",
		Annotations: i.ToolAnnotations("Update an existing OPA/Rego policy", i.OPEN_WORLD),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Policy ID to update",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Human-readable name for the policy",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Description of what the policy validates",
				},
				"rego_code": map[string]any{
					"type":        "string",
					"description": "The Rego policy code. Must define 'package policy' and return {violations: [], warnings: [], debug: []} structure",
				},
				"enabled": map[string]any{
					"type":        "boolean",
					"description": "Whether the policy is enabled",
				},
			},
			Required: []string{"id"},
		},
	}, update(storage, engine)
}

func update(storage types.Storage, engine types.PolicyEngine) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args updateArgs) (*mcp.CallToolResult, error) {
		// Get existing policy
		existingPolicy, err := storage.GetPolicy(ctx, args.ID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get existing policy: %v", err)), nil
		}

		if existingPolicy == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Policy not found: %s", args.ID)), nil
		}

		// Create updated policy from existing one
		updatedPolicy := *existingPolicy

		// Update fields if provided
		if args.Name != nil {
			updatedPolicy.Name = *args.Name
		}
		if args.Description != nil {
			updatedPolicy.Description = *args.Description
		}
		if args.RegoCode != nil {
			// Validate the new policy code
			if err := engine.ValidatePolicy(ctx, *args.RegoCode); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Policy validation failed: %v", err)), nil
			}
			updatedPolicy.RegoCode = *args.RegoCode
		}
		if args.Enabled != nil {
			updatedPolicy.Enabled = *args.Enabled
		}

		// Save updated policy
		if err := storage.SavePolicy(ctx, updatedPolicy); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update policy: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Policy updated successfully: %s", args.ID)), nil
	})
}
