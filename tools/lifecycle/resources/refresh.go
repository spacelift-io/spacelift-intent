// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"
	"reflect"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/tools/provider"
	"github.com/spacelift-io/spacelift-intent/types"
)

type refreshArgs struct {
	ResourceID string `json:"resource_id"`
}

func Refresh(storage types.Storage, providerManager types.ProviderManager, registryClient types.RegistryClient) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-resources-refresh"),
		Description: "Refresh an existing resource by reading its current state using the provider. " +
			"Essential for Verification Phase - use this to detect resource drift and ensure " +
			"actual infrastructure matches expected state. MEDIUM risk operation that updates " +
			"stored state with current provider values. " +
			"\n\nPresentation: Present drift detection results with clear status indicators " +
			"(FRESH/DRIFTED/DELETED). Use structured format showing detected changes and their " +
			"impact. \n\nCritical for monitoring resource health and identifying external " +
			"changes that may affect infrastructure consistency.",
		Annotations: i.ToolAnnotations("Refresh an existing resource", i.Idempotent|i.OpenWorld),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "The unique identifier of the resource to refresh",
				},
			},
			Required: []string{"resource_id"},
		},
	}, Handler: refresh(storage, providerManager, registryClient)}
}

func refresh(storage types.Storage, providerManager types.ProviderManager, registryClient types.RegistryClient) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args refreshArgs) (*mcp.CallToolResult, error) {
		// Get the current state from database
		record, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get state: %v", err)), nil
		}

		if record == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' not found", args.ResourceID)), nil
		}

		// If provider version is missing, search for it
		if record.ProviderVersion == "" {
			providerResult, err := provider.SearchForProvider(ctx, registryClient, record.Provider)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve provider version: %v", err)), nil
			}
			record.ProviderVersion = providerResult.Provider.Version
		}

		// Parse the stored state
		input := types.ResourceOperationInput{
			ResourceID:      args.ResourceID,
			Provider:        record.GetProvider().Name,
			ProviderVersion: record.GetProvider().Version,
			ResourceType:    record.ResourceType,
			Operation:       "refresh",
			CurrentState:    record.State,
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

		// Refresh the resource using the provider manager
		stateResult, err := providerManager.RefreshResource(ctx, record.GetProvider(), record.ResourceType, record.State)
		if err != nil {
			err = fmt.Errorf("failed to refresh resource: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Determine status and message based on state changes
		var status, message string
		var responseState map[string]any

		if len(stateResult) == 0 {
			status, message = "warning", "Resource appears to have been deleted externally"
		} else if !reflect.DeepEqual(record.State, stateResult) {
			status, message = "warning", "Resource has drifted - changes detected during refresh"
		} else {
			status, message = "refreshed", "Resource was already fresh - no changes detected"
		}

		// Save state if we have refreshed data
		if len(stateResult) > 0 {
			updatedRecord := types.StateRecord{
				ResourceID:      args.ResourceID,
				Provider:        record.Provider,
				ProviderVersion: record.ProviderVersion,
				ResourceType:    record.ResourceType,
				State:           stateResult,
			}

			// Add operation context for automatic history tracking
			ctx = context.WithValue(ctx, types.OperationContextKey, "refresh")
			ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

			operation.ProposedState = stateResult

			if err = storage.SaveState(ctx, updatedRecord); err != nil {
				err = fmt.Errorf("failed to save refreshed state: %w", err)
				return mcp.NewToolResultError(err.Error()), nil
			}

		}

		return RespondJSON(map[string]any{
			"provider":         record.GetProvider().Name,
			"provider_version": record.GetProvider().Version,
			"resource_id":      args.ResourceID,
			"status":           status,
			"message":          message,
			"result":           responseState,
		})
	},
	)
}
