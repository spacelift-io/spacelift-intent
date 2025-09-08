package project

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
)

// Current retrieves the current Intent project
func Current() i.Tool {
	tool := mcp.Tool{
		Name:        string("project-current"),
		Description: "Get the current Intent project. LOW risk read-only operation for retrieving project configuration. Use this to understand project setup, check variable values, and analyze configuration before executing resource management tools.",
		Annotations: i.ToolAnnotations("Get current intent project", i.READONLY|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
			Required:   []string{},
		},
	}

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("Not implemented"), nil
	}

	return i.Tool{Tool: tool, Handler: handler}
}
