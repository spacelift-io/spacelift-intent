// Copyright 2025 Spacelift, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
