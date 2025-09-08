package project

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
)

// Describe retrieves a Intent project by ID
func Describe() i.Tool {
	tool := mcp.Tool{
		Name:        string("project-describe"),
		Description: "Describe a Intent project by ID. LOW risk read-only operation for retrieving project configuration. Use this to understand project setup, check variable values, and analyze configuration before executing resource management tools.",
		Annotations: i.ToolAnnotations("Describe intent project", i.READONLY|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Project ID to describe",
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
