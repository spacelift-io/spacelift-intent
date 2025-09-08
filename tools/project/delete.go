package project

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
)

// Delete removes a Intent project
func Delete() (mcp.Tool, i.ToolHandler) {
	tool := mcp.Tool{
		Name:        string("project-delete"),
		Description: "Delete a Intent project by ID. HIGH risk DESTRUCTIVE operation that permanently removes project configuration. This will affect future resource management tools usage. Use with caution and ensure the project is no longer needed before deletion.",
		Annotations: i.ToolAnnotations("Delete intent project", i.DESTRUCTIVE),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Project ID to delete",
				},
			},
			Required: []string{"id"},
		},
	}

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("Not implemented"), nil
	}

	return tool, handler
}
