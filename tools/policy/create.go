package policy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type createArgs struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RegoCode    string `json:"rego_code"`
	Enabled     *bool  `json:"enabled"`
}

// Create creates a new policy
func Create(storage types.Storage, engine types.PolicyEngine) (mcp.Tool, i.ToolHandler) {
	tool := mcp.Tool{
		Name:        string("policy-create"),
		Description: "Create a new OPA/Rego policy for resource validation. Essential for Safety Protocol implementation - use this to establish governance controls and compliance checks that are enforced automatically during resource operations. Validates policy syntax before creation to ensure proper enforcement.",
		Annotations: i.ToolAnnotations("Create policy", 0),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
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
					"description": "Whether the policy is enabled (default: true)",
					"default":     true,
				},
			},
			Required: []string{"name", "rego_code"},
		},
	}

	handler := mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args createArgs) (*mcp.CallToolResult, error) {

		// Set default for enabled if not provided
		enabled := true
		if args.Enabled != nil {
			enabled = *args.Enabled
		}

		// Validate the policy code
		if err := engine.ValidatePolicy(ctx, args.RegoCode); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Policy validation failed: %v", err)), nil
		}

		// Generate ID
		id, err := uuid.NewV7()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to generate policy ID: %v", err)), nil
		}

		// Create policy
		policy := types.Policy{
			ID:          id.String(),
			Name:        args.Name,
			Description: args.Description,
			RegoCode:    args.RegoCode,
			Enabled:     enabled,
		}

		if err := storage.SavePolicy(ctx, policy); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to save policy: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Policy created successfully with ID: %s", policy.ID)), nil
	})

	return tool, handler
}
