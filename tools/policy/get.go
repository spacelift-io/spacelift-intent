package policy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type getArgs struct {
	ID string `json:"id"`
}

// Get retrieves a policy by ID
func Get(storage types.Storage) (mcp.Tool, i.ToolHandler) {
	tool := mcp.Tool{
		Name:        string("policy-get"),
		Description: "Get a specific OPA/Rego policy by ID. LOW risk read-only operation for reviewing governance controls and compliance rules. Use this to understand policy logic, check enforcement status, and analyze validation rules before making infrastructure changes.",
		Annotations: i.ToolAnnotations("Get policy", i.READONLY|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Policy ID",
				},
			},
			Required: []string{"id"},
		},
	}

	handler := mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args getArgs) (*mcp.CallToolResult, error) {

		policy, err := storage.GetPolicy(ctx, args.ID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get policy: %v", err)), nil
		}

		if policy == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Policy not found: %s", args.ID)), nil
		}

		// Format output
		status := "enabled"
		if !policy.Enabled {
			status = "disabled"
		}

		output := "Policy Details:\n"
		output += "ID: " + policy.ID + "\n"
		output += "Name: " + policy.Name + "\n"
		output += "Status: " + status + "\n"
		if policy.Description != "" {
			output += "Description: " + policy.Description + "\n"
		}
		output += "Created: " + policy.CreatedAt + "\n"
		if policy.UpdatedAt != policy.CreatedAt {
			output += "Updated: " + policy.UpdatedAt + "\n"
		}
		output += "\nRego Code:\n" + policy.RegoCode

		// Also return as JSON for programmatic access
		policyJSON, _ := json.MarshalIndent(policy, "", "  ")

		result := fmt.Sprintf("%s\n\nJSON format:\n%s", output, string(policyJSON))
		return mcp.NewToolResultText(result), nil
	})

	return tool, handler
}
