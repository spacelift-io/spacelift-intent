// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type updateArgs struct {
	ResourceID      string         `json:"resource_id"`
	Provider        *string        `json:"provider,omitempty"`
	ProviderVersion *string        `json:"provider_version,omitempty"`
	Config          map[string]any `json:"config"`
}

func Update(storage types.Storage, providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-resources-update"),
		Description: "Update an existing OpenTofu resource by ID with a new configuration. " +
			"MEDIUM risk operation that requires policy evaluation and user approval for " +
			"configuration changes. Handles internal planning and application automatically " +
			"through the MCP abstraction layer. Validates configuration against existing state, " +
			"enforces policies, and manages state persistence. " +
			"\n\nArgument Handling: Set unknown/optional arguments to appropriate defaults: " +
			"strings to null or '', booleans to null or false, numbers to null or 0, arrays " +
			"to null or [], objects to null or {}. Ensure ALL required arguments are provided. " +
			"\n\nPresentation: Present results using Infrastructure Configuration Analysis " +
			"format with MODIFY section and risk assessment. On errors, use OpenTofu MCP Server " +
			"error format with root cause analysis and recommended fixes.",
		Annotations: i.ToolAnnotations("Update resource with a new configuration", i.Idempotent|i.OpenWorld),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "Unique identifier of the resource to update",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (optional, uses stored value if not specified)",
				},
				"provider_version": map[string]any{
					"type":        "string",
					"description": "Provider version (optional, uses stored value if not specified)",
				},
				"config": map[string]any{
					"type":        "object",
					"description": "New configuration parameters for the resource",
				},
			},
			Required: []string{"resource_id", "config"},
		},
	}, Handler: update(storage, providerManager)}
}

func update(storage types.Storage, providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args updateArgs) (*mcp.CallToolResult, error) {
		// Get the current state from database
		record, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get current state: %v", err)), nil
		}

		if record == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' not found", args.ResourceID)), nil
		}

		// Use provided provider/version or fall back to stored values
		providerConfig := record.GetProvider()
		if args.Provider != nil && *args.Provider != "" {
			providerConfig.Name = *args.Provider
		}

		if args.ProviderVersion != nil && *args.ProviderVersion != "" {
			providerConfig.Version = *args.ProviderVersion
		}

		// Parse the stored state
		input := types.ResourceOperationInput{
			ResourceID:      args.ResourceID,
			ResourceType:    record.ResourceType,
			Provider:        providerConfig.Name,
			ProviderVersion: providerConfig.Version,
			Operation:       "update",
			CurrentState:    record.State,
			ProposedState:   args.Config,
		}

		operation, err := newResourceOperation(input)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create operation resource: %v", err)), nil
		}

		defer func() {
			if err != nil {
				errMessage := err.Error()
				operation.Failed = &errMessage
			}
			storage.SaveResourceOperation(ctx, operation)
		}()

		// Update the resource using the provider manager
		state, err := providerManager.UpdateResource(ctx, providerConfig, record.ResourceType, record.State, args.Config)
		if err != nil {
			err = fmt.Errorf("failed to update resource: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Handle empty state case
		if len(state) == 0 {
			return mcp.NewToolResultText("{}"), nil
		}

		// Update state in database with potentially new provider/version
		updatedRecord := types.StateRecord{
			ResourceID:      args.ResourceID,
			Provider:        providerConfig.Name,
			ProviderVersion: providerConfig.Version,
			ResourceType:    record.ResourceType,
			State:           state,
			CreatedAt:       record.CreatedAt, // Keep original creation time
		}

		// Add operation context for automatic history tracking
		ctx = context.WithValue(ctx, types.OperationContextKey, "update")
		ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

		if err = storage.SaveState(ctx, updatedRecord); err != nil {
			err = fmt.Errorf("failed to save updated state: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		return RespondJSON(map[string]any{
			"provider":         providerConfig.Name,
			"provider_version": providerConfig.Version,
			"resource_id":      args.ResourceID,
			"result":           state,
			"status":           "updated",
		})
	})
}
