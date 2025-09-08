package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent/tools/internal"
	"spacelift-intent/types"
)

type refreshArgs struct {
	ResourceID string `json:"resource_id"`
}

func Refresh(storage types.Storage, providerManager types.ProviderManager) i.Tool {
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
		Annotations: i.ToolAnnotations("Refresh an existing resource", i.IDEMPOTENT|i.OPEN_WORLD),
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
	}, Handler: refresh(storage, providerManager)}
}

func refresh(storage types.Storage, providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args refreshArgs) (*mcp.CallToolResult, error) {
		// Get the current state from database
		record, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get state: %v", err)), nil
		}

		if record == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' not found", args.ResourceID)), nil
		}

		// Parse the stored state
		var currentState map[string]any
		if err := json.Unmarshal([]byte(record.State), &currentState); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse stored state: %v", err)), nil
		}

		input := types.ResourceOperationInput{
			ResourceID:   args.ResourceID,
			Provider:     record.Provider,
			ResourceType: record.ResourceType,
			Operation:    "refresh",
			CurrentState: currentState,
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
		refreshedState, err := providerManager.RefreshResource(ctx, record.Provider, record.ResourceType, currentState)
		if err != nil {
			err = fmt.Errorf("Failed to refresh resource: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Determine status and message based on state changes
		var status, message string
		var responseState map[string]any

		if len(refreshedState) == 0 {
			status, message = "warning", "Resource appears to have been deleted externally"
			responseState = map[string]any{}
		} else if !reflect.DeepEqual(currentState, refreshedState) {
			status, message = "warning", "Resource has drifted - changes detected during refresh"
			responseState = refreshedState
		} else {
			status, message = "refreshed", "Resource was already fresh - no changes detected"
			responseState = refreshedState
		}

		// Save state if we have refreshed data
		if len(refreshedState) > 0 {
			stateBytes, errJSON := json.Marshal(refreshedState)
			if errJSON != nil {
				err = fmt.Errorf("Failed to marshal refreshed state for storage: %w", errJSON)
				return mcp.NewToolResultError(err.Error()), nil
			}

			updatedRecord := types.StateRecord{
				ResourceID:   args.ResourceID,
				Provider:     record.Provider,
				Version:      record.Version,
				ResourceType: record.ResourceType,
				State:        string(stateBytes),
			}

			// Add operation context for automatic history tracking
			ctx = context.WithValue(ctx, types.OperationContextKey, "refresh")
			ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

			operation.ProposedState = refreshedState

			if err = storage.SaveState(ctx, updatedRecord); err != nil {
				err = fmt.Errorf("Failed to save refreshed state: %w", err)
				return mcp.NewToolResultError(err.Error()), nil
			}

		}

		return RespondJSON(map[string]any{
			"resource_id": args.ResourceID,
			"status":      status,
			"message":     message,
			"result":      responseState,
		})
	},
	)
}
