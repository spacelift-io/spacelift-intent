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

package dependencies

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type removeArgs struct {
	From string `json:"from_resource_id"`
	To   string `json:"to_resource_id"`
}

func Remove(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("lifecycle-resources-dependencies-remove"),
		Description: "Explicitly remove a dependency relationship between two resources. MEDIUM risk operation that modifies resource ordering and dependency chains. Use with caution - removing dependencies can affect deployment sequences and potentially cause resource lifecycle issues. Verify impact on dependent resources before removing to maintain infrastructure stability.",
		Annotations: i.ToolAnnotations("Explicitly remove between two resources", i.Destructive|i.Idempotent),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"from_resource_id": map[string]any{
					"type":        "string",
					"description": "The resource that depends on another (source)",
				},
				"to_resource_id": map[string]any{
					"type":        "string",
					"description": "The resource being depended upon (target)",
				},
			},
			Required: []string{"from_resource_id", "to_resource_id"},
		},
	}, Handler: remove(storage)}
}

func remove(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args removeArgs) (*mcp.CallToolResult, error) {
		if err := storage.RemoveDependency(ctx, args.From, args.To); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to remove dependency: %v", err)), nil
		}

		return i.RespondJSON(struct {
			removeArgs
			Status string `json:"status"`
		}{args, "removed"})
	})
}
