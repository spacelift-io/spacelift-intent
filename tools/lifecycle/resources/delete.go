package resources

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type deleteArgs struct {
	ResourceID string `json:"resource_id"`
}

func Delete(storage types.Storage, providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-resources-delete"),
		Description: "Delete an existing resource by its ID and remove it from the state. " +
			"HIGH RISK destructive operation that requires explicit user approval. " +
			"Check for dependents before deletion to avoid breaking dependent resources. " +
			"\n\nSafety Protocol: Verify business/service impact before proceeding. Present using " +
			"HIGH RISK OPERATION format with potential impact, estimated downtime, and rollback " +
			"complexity analysis. Require explicit 'CONFIRM' response before proceeding. " +
			"\n\nPresentation: Present results using Infrastructure Configuration Analysis " +
			"format with REMOVE section.",
		Annotations: i.ToolAnnotations("Delete a managed resource", i.Destructive|i.Idempotent|i.OpenWorld),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "The unique identifier of the resource to delete",
				},
			},
			Required: []string{"resource_id"},
		},
	}, Handler: deleteResource(storage, providerManager)}
}

func deleteResource(storage types.Storage, providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args deleteArgs) (*mcp.CallToolResult, error) {
		// Get the current state from database
		record, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get state: %v", err)), nil
		}

		if record == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' not found", args.ResourceID)), nil
		}

		input := types.ResourceOperationInput{
			ResourceID:    record.ResourceID,
			ResourceType:  record.ResourceType,
			Provider:      record.Provider,
			Operation:     "delete",
			CurrentState:  record.State,
			ProposedState: nil, // no proposed state for delete
		}

		operation, err := newResourceOperation(input)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create operation resource: %v", err)), nil
		}

		defer func() {
			if err != nil {
				errMessage := fmt.Sprintf("Failed to delete resource: %v", err)
				operation.Failed = &errMessage
			}
			storage.SaveResourceOperation(ctx, operation)
		}()

		// Delete the resource using the provider manager
		err = providerManager.DeleteResource(ctx, record.GetProvider(), record.ResourceType, record.State)
		if err != nil {
			err = fmt.Errorf("failed to delete resource: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Add operation context for automatic history tracking
		ctx = context.WithValue(ctx, types.OperationContextKey, "delete")
		ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

		// Delete the state from database (dependencies automatically cleaned up via CASCADE)
		if err = storage.DeleteState(ctx, args.ResourceID); err != nil {
			err = fmt.Errorf("failed to delete state from database: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		return RespondJSON(map[string]any{
			"resource_id": args.ResourceID,
			"status":      "deleted",
		})
	})
}
