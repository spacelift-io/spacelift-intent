package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent/tools/internal"
	"spacelift-intent/types"
)

type importArgs struct {
	ResourceID   string `json:"resource_id"`
	Provider     string `json:"provider"`
	ResourceType string `json:"resource_type"`
}

func Import(storage types.Storage, providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-resources-import"),
		Description: "Import an existing external resource and store in the state. " +
			"MEDIUM risk operation for bringing existing infrastructure under MCP management. " +
			"Use this to adopt pre-existing resources that were created outside of the MCP " +
			"workflow. Validates resource exists and reads current state through provider APIs. " +
			"\n\nArgument Handling: If schema mismatches occur, set unknown/optional arguments " +
			"to appropriate defaults: strings to null or '', booleans to null or false, numbers " +
			"to null or 0, arrays to null or [], objects to null or {}. " +
			"\n\nPresentation: Present results using Infrastructure Configuration Analysis " +
			"format showing imported resources. On errors, use OpenTofu MCP Server error format " +
			"with import-specific troubleshooting guidance. " +
			"\n\nEssential for migrating from manual infrastructure to managed infrastructure as code.",
		Annotations: i.ToolAnnotations("Import an external resource resource", i.IDEMPOTENT|i.OPEN_WORLD),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "Unique identifier for this resource instance in the state (must be unique and intuitive)",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'hashicorp/aws', 'hashicorp/random')",
				},
				"resource_type": map[string]any{
					"type":        "string",
					"description": "The OpenTofu resource type (e.g., 'random_string', 'aws_instance')",
				},
			},
			Required: []string{"resource_id", "provider", "resource_type"},
		},
	}, Handler: _import(storage, providerManager)}
}

func _import(storage types.Storage, providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args importArgs) (*mcp.CallToolResult, error) {
		// Check if ID already exists
		existingState, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to check existing state: %v", err)), nil
		}
		if existingState != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' already exists", args.ResourceID)), nil
		}

		input := types.ResourceOperationInput{
			ResourceID:    args.ResourceID,
			ResourceType:  args.ResourceType,
			Provider:      args.Provider,
			Operation:     "import",
			CurrentState:  nil, // no current state for import
			ProposedState: nil, // no proposed state for import
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

		// Import resource using provider manager
		state, err := providerManager.ImportResource(ctx, args.Provider, args.ResourceType, args.ResourceID)
		if err != nil {
			err = fmt.Errorf("Failed to import resource: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Handle empty state case
		if len(state) == 0 {
			return mcp.NewToolResultText("{}"), nil
		}

		// Get actual provider version
		version, err := providerManager.GetProviderVersion(ctx, args.Provider)
		if err != nil {
			// Fallback to "latest" if we can't get the version
			version = "latest"
		}

		// Persist state to database
		stateBytes, err := json.Marshal(state)
		if err != nil {
			err = fmt.Errorf("Failed to marshal state for storage: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		record := types.StateRecord{
			ResourceID:   args.ResourceID,
			Provider:     args.Provider,
			Version:      version,
			ResourceType: args.ResourceType,
			State:        string(stateBytes),
		}

		// Add operation context for automatic history tracking
		ctx = context.WithValue(ctx, types.OperationContextKey, "import")
		ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

		if err = storage.SaveState(ctx, record); err != nil {
			err = fmt.Errorf("Failed to save state: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		operation.ProposedState = state

		return RespondJSON(map[string]any{
			"resource_id": args.ResourceID,
			"result":      state,
			"status":      "imported",
			"message":     "resource successfully imported",
		})
	})
}
