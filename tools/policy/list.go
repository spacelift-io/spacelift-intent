package policy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type listArgs struct{}

// List lists all policies
func List(storage types.Storage) (mcp.Tool, i.ToolHandler) {
	tool := mcp.Tool{
		Name:        string("policy-list"),
		Description: "List all OPA/Rego policies. LOW risk read-only operation for reviewing governance controls and compliance landscape. Essential for Discovery Phase - use this to understand what policies are active and enforced before making infrastructure changes.",
		Annotations: i.ToolAnnotations("List policies", i.READONLY|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}

	handler := mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args listArgs) (*mcp.CallToolResult, error) {
		policies, err := storage.ListPolicies(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list policies: %v", err)), nil
		}

		if len(policies) == 0 {
			return mcp.NewToolResultText("No policies found"), nil
		}

		// Format output
		output := "Policies:\n"
		for _, policy := range policies {
			status := "enabled"
			if !policy.Enabled {
				status = "disabled"
			}
			output += fmt.Sprintf("- %s (%s) - %s\n", policy.Name, policy.ID, status)
			if policy.Description != "" {
				output += fmt.Sprintf("  Description: %s\n", policy.Description)
			}
			output += fmt.Sprintf("  Created: %s\n", policy.CreatedAt)
			if policy.UpdatedAt != policy.CreatedAt {
				output += fmt.Sprintf("  Updated: %s\n", policy.UpdatedAt)
			}
			output += "\n"
		}

		// Also return as JSON for programmatic access
		policiesJSON, _ := json.MarshalIndent(policies, "", "  ")

		result := fmt.Sprintf("%s\nJSON format:\n%s", output, string(policiesJSON))
		return mcp.NewToolResultText(result), nil
	})

	return tool, handler
}
