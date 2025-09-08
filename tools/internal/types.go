package internal

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandler is the function signature for MCP tool handlers
type ToolHandler = func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

type Tool struct {
	Tool    mcp.Tool
	Handler ToolHandler
}
