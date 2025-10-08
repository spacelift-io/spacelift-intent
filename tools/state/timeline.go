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

func Timeline(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("state-timeline"),
		Description: "Get state timeline events with filtering and pagination. " +
			"Essential for Verification Phase - use this to review infrastructure changes, " +
			"audit operations, and monitor resource lifecycle history. LOW risk read-only " +
			"operation for compliance and troubleshooting. " +
			"\n\nPresentation: Present timeline data with clear chronological formatting, " +
			"operation summaries, and impact analysis. " +
			"\n\nCritical for tracking who made changes, when they occurred, and understanding " +
			"deployment patterns for operational excellence.",
		Annotations: i.ToolAnnotations("Get state timeline events", i.Readonly|i.Idempotent),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "Filter to specific resource (optional - if omitted, returns global timeline)",
				},
				"from_time": map[string]any{
					"type":        "string",
					"description": "Start time in RFC3339 format (optional)",
				},
				"to_time": map[string]any{
					"type":        "string",
					"description": "End time in RFC3339 format (optional)",
				},
				"limit": map[string]any{
					"type":        "number",
					"description": "Maximum number of events to return (default: 50)",
					"default":     50,
				},
				"offset": map[string]any{
					"type":        "number",
					"description": "Number of events to skip for pagination (default: 0)",
					"default":     0,
				},
			},
		},
	}, Handler: timeline(storage)}
}

func timeline(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args types.TimelineQuery) (*mcp.CallToolResult, error) {
		response, err := storage.GetTimeline(ctx, args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get timeline: %v", err)), nil
		}

		return i.RespondJSON(response)
	})
}
