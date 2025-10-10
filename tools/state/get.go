// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package state

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type getArgs struct {
	ResourceID string `json:"resource_id"`
}

func Get(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("state-get"),
		Description: "Get the stored state for a resource by ID including its dependencies " +
			"and dependents. Essential for Discovery Phase - use this to understand current " +
			"infrastructure state and dependency relationships before making changes. LOW risk " +
			"read-only operation for state analysis. " +
			"\n\nPresentation: Present state information with clear resource details, " +
			"dependency mapping, and impact analysis formatting. " +
			"\n\nCritical for Safety Protocol to verify state consistency and review what " +
			"resources will be affected by changes.",
		Annotations: i.ToolAnnotations("Get resource state", i.Readonly|i.Idempotent),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "The unique identifier of the resource",
				},
			},
			Required: []string{"resource_id"},
		},
	}, Handler: get(storage)}
}

func get(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args getArgs) (*mcp.CallToolResult, error) {
		record, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get state: %v", err)), nil
		}

		if record == nil {
			return mcp.NewToolResultError(fmt.Sprintf("No state found for ID '%s'", args.ResourceID)), nil
		}

		// Get dependencies (what this resource depends on)
		var dependencyIDs []string
		if dependencies, err := storage.GetDependencies(ctx, args.ResourceID); err == nil {
			for _, dep := range dependencies {
				dependencyIDs = append(dependencyIDs, dep.ToResourceID)
			}
		}

		// Get dependents (what depends on this resource)
		var dependentIDs []string
		if dependents, err := storage.GetDependents(ctx, args.ResourceID); err == nil {
			for _, dep := range dependents {
				dependentIDs = append(dependentIDs, dep.FromResourceID)
			}
		}

		return i.RespondJSON(map[string]any{
			"resource_id":      record.ResourceID,
			"provider":         record.Provider,
			"provider_version": record.ProviderVersion,
			"resource_type":    record.ResourceType,
			"state":            record.State,
			"created_at":       record.CreatedAt,
			"dependencies":     dependencyIDs,
			"dependents":       dependentIDs,
		})
	})
}
