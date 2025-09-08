package project

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
)

// Create creates a new Intent project
func Create() (mcp.Tool, i.ToolHandler) {
	tool := mcp.Tool{
		Name:        string("project-create"),
		Description: "Create a new reusable project. MEDIUM risk operation that creates persistent intent project. Use this to define reusable environment for resources management.",
		Annotations: i.ToolAnnotations("Create intent project", 0),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Project name",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Optional project description",
				},
			},
			Required: []string{"name"},
		},
	}

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("Not implemented"), nil
	}

	return tool, handler
}
