// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type operationsArgs struct {
	ResourceID   *string `json:"resource_id"`
	Provider     *string `json:"provider"`
	ResourceType *string `json:"resource_type"`
	Page         *int    `json:"page"`
	PageSize     *int    `json:"page_size"`
}

const (
	defaultOperationsPageSize = 20
	maxOperationsPageSize     = 100
)

func Operations(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-resources-operations"),
		Description: "List operations performed on resources with optional filtering by resource ID, " +
			"provider, or resource type. Essential for Discovery Phase - use this to review " +
			"infrastructure operation history and understand what actions have been performed " +
			"on managed resources. LOW risk read-only operation for auditing and analysis. " +
			"\n\nPresentation: Present operation history with clear status indicators " +
			"(SUCCEEDED/FAILED), timestamps, and policy evaluation results (allow/deny). " +
			"Include both human-readable summary and JSON format for programmatic access. " +
			"\n\nCritical for tracking resource lifecycle events and maintaining operational " +
			"visibility of infrastructure changes.",
		Annotations: i.ToolAnnotations("List operations on resources", i.OpenWorld),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "Filter by resource unique identifier, if not provided, all resources will be returned",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Filter by provider name (e.g., 'hashicorp/aws', 'hashicorp/random'), if not provided, all resources will be returned",
				},
				"resource_type": map[string]any{
					"type":        "string",
					"description": "Filter by resource type (e.g., 'random_string', 'aws_instance'), if not provided, all resources will be returned",
				},
				"page": map[string]any{
					"type":        "integer",
					"description": "Page number to retrieve (1-based). Defaults to page 1 if omitted.",
					"minimum":     1,
				},
				"page_size": map[string]any{
					"type":        "integer",
					"description": "Number of operations per page. Defaults to 20 and is capped at 100.",
					"minimum":     1,
					"maximum":     100,
				},
			},
			Required: []string{},
		},
	}, Handler: operations(storage)}
}

func operations(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args operationsArgs) (*mcp.CallToolResult, error) {
		currentPage := 1
		if args.Page != nil {
			if *args.Page < 1 {
				return mcp.NewToolResultError("Invalid page: must be greater than or equal to 1"), nil
			}
			currentPage = *args.Page
		}

		pageSize := defaultOperationsPageSize
		if args.PageSize != nil {
			if *args.PageSize < 1 {
				return mcp.NewToolResultError("Invalid page_size: must be greater than or equal to 1"), nil
			}
			pageSize = *args.PageSize
			if pageSize > maxOperationsPageSize {
				pageSize = maxOperationsPageSize
			}
		}

		offset := (currentPage - 1) * pageSize
		limitWithLookahead := pageSize + 1

		operations, err := storage.ListResourceOperations(ctx, types.ResourceOperationsArgs{
			ResourceID:   args.ResourceID,
			Provider:     args.Provider,
			ResourceType: args.ResourceType,
			Limit:        &limitWithLookahead,
			Offset:       offset,
		})

		if err != nil {
			return mcp.NewToolResultError("Failed to list operations: " + err.Error()), nil
		}

		hasMore := false
		if len(operations) > pageSize {
			hasMore = true
			operations = operations[:pageSize]
		}

		var output string
		output += fmt.Sprintf("Operations (page %d, page size %d):\n", currentPage, pageSize)
		if len(operations) == 0 {
			output += "No operations found for the provided filters on this page.\n"
		} else {
			for _, op := range operations {
				output += fmt.Sprintf("- ID: %s, created at: %s\n", op.ID, op.CreatedAt)
				output += fmt.Sprintf("  - Resource ID: %s\n", op.ResourceID)
				output += fmt.Sprintf("  - Resource Type: %s\n", op.ResourceType)
				output += fmt.Sprintf("  - Provider: %s\n", op.Provider)
				output += fmt.Sprintf("  - Operation: %s\n", op.Operation)
				if op.Failed != nil && *op.Failed != "" {
					output += fmt.Sprintf("  - FAILED: %s\n", *op.Failed)
				} else {
					output += "  - SUCCEEDED\n"
				}
				output += "\n"
			}
		}

		if hasMore {
			output += fmt.Sprintf("More operations are available. Request page %d to view additional results.\n", currentPage+1)
		}

		responseJSON, err := json.Marshal(map[string]any{
			"page":       currentPage,
			"page_size":  pageSize,
			"has_more":   hasMore,
			"operations": operations,
		})
		if err != nil {
			return mcp.NewToolResultError("Failed to marshal operations: " + err.Error()), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{Type: "text", Text: output},
				mcp.TextContent{Type: "text", Text: "JSON format:\n" + string(responseJSON)},
			},
		}, nil
	})
}
