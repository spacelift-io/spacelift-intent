package project

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
)

// List retrieves all available Intent projects
func List() (mcp.Tool, i.ToolHandler) {
	tool := mcp.Tool{
		Name:        string("project-list"),
		Description: "List all available Intent projects. LOW risk read-only operation for discovering projects. Use this to explore available projects, check project names and IDs, and understand project organization before selecting projects to use or manage.",
		Annotations: i.ToolAnnotations("List intent projects", i.READONLY|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("Not implemented"), nil
	}

	return tool, handler
}
