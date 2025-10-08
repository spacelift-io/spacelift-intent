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

package state

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

// TODO: paginate, maybe add some filtering criteria?
func List(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("state-list"),
		Description: "List all stored resource states. Essential for Discovery Phase - use this to understand the complete infrastructure inventory and identify existing resources before making changes. LOW risk read-only operation for workspace analysis. Present inventory using structured format with resource counts, types, and status summaries. Critical for Safety Protocol to check current workspace status and verify state consistency before deployment operations.",
		Annotations: i.ToolAnnotations("List managed resource states", i.Readonly|i.Idempotent),
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}, Handler: list(storage)}
}

func list(storage types.Storage) i.ToolHandler {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		records, err := storage.ListStates(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list states: %v", err)), nil
		}

		return i.RespondJSON(map[string]any{
			"states": records,
			"count":  len(records),
		})
	}
}
