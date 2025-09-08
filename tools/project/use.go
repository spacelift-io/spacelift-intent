package project

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
)

// Use activates a Intent project for the current session
func Use() i.Tool {
	tool := mcp.Tool{
		Name:        string("project-use"),
		Description: "Activate a Intent project for the current session. MEDIUM risk operation that applies project configuration to subsequent infrastructure operations. Use this to set up the execution environment with required credentials, configuration, and dependencies before performing resource operations.",
		Annotations: i.ToolAnnotations("Use intent project", i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Project ID to activate",
				},
			},
			Required: []string{"id"},
		},
	}

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("Not implemented"), nil
	}

	return i.Tool{Tool: tool, Handler: handler}
}
