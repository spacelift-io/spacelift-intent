package resources

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type createArgs struct {
	ResourceID      string         `json:"resource_id"`
	Provider        string         `json:"provider"`
	ResourceType    string         `json:"resource_type"`
	Config          map[string]any `json:"config"`
	ProviderVersion *string        `json:"provider_version,omitempty"`
	ProviderConfig  map[string]any `json:"provider_config,omitempty"`
}

func (args createArgs) GetProvider() *types.ProviderConfig {
	return &types.ProviderConfig{
		Name:    args.Provider,
		Version: args.ProviderVersion,
		Config:  args.ProviderConfig,
	}
}

func Create(storage types.Storage, providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-resources-create"),
		Description: "Create a new managed resource of any type from any provider with required ID, " +
			"then store in the state. Core tool for the Execution Phase - handles internal " +
			"planning and application automatically through the MCP abstraction layer. " +
			"Validates configuration, enforces policies, and manages state persistence. " +
			"\n\nArgument Handling: Ensure ALL required arguments are provided in config to " +
			"prevent 'expected X arguments, got Y' errors. Set unknown/optional arguments to " +
			"appropriate defaults: strings to null or '', booleans to null or false, numbers " +
			"to null or 0, arrays to null or [], objects to null or {}. " +
			"\n\nPresentation: Present successful results using Infrastructure Configuration " +
			"Analysis format with CREATE section, risk assessment, and next steps. For provider " +
			"argument mismatches, use Provider Argument Count Mismatch format with auto-resolution " +
			"strategy. \n\nHint: may want to add a dependency if applicable.",
		Annotations: i.ToolAnnotations("Create a new managed resource", i.OPEN_WORLD),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "Unique identifier for this resource instance (must be unique and intuitive)",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'hashicorp/aws', 'hashicorp/random')",
				},
				"resource_type": map[string]any{
					"type":        "string",
					"description": "The resource type to create (e.g., 'random_string', 'aws_instance')",
				},
				"config": map[string]any{
					"type":        "object",
					"description": "Configuration parameters for the resource",
				},
			},
			Required: []string{"resource_id", "provider", "resource_type", "config"},
		},
	}, Handler: create(storage, providerManager)}
}

func create(storage types.Storage, providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args createArgs) (*mcp.CallToolResult, error) {
		// Check if ID already exists
		existingState, err := storage.GetState(ctx, args.ResourceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to check existing state: %v", err)), nil
		}
		if existingState != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' already exists", args.ResourceID)), nil
		}

		input := types.ResourceOperationInput{
			Operation:     "create",
			ResourceID:    args.ResourceID,
			ResourceType:  args.ResourceType,
			Provider:      args.Provider,
			CurrentState:  nil, // no current state for create
			ProposedState: args.Config,
		}

		operation, err := newResourceOperation(input)

		defer func() {
			if err != nil {
				errMessage := err.Error()
				operation.Failed = &errMessage
			}
			storage.SaveResourceOperation(ctx, operation)
		}()

		// Create resource using provider manager
		state, err := providerManager.CreateResource(ctx, args.GetProvider(), args.ResourceType, args.Config)
		if err != nil {
			err = fmt.Errorf("failed to create resource: %w", err)
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
		record := types.StateRecord{
			ResourceID:   args.ResourceID,
			Provider:     args.Provider,
			ResourceType: args.ResourceType,
			Version:      version,
			State:        state,
		}

		// Add operation context for automatic history tracking
		ctx = context.WithValue(ctx, types.OperationContextKey, "create")
		ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

		if err = storage.SaveState(ctx, record); err != nil {
			err = fmt.Errorf("failed to save state: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		operation.Failed = nil

		return RespondJSON(map[string]any{
			"resource_id": args.ResourceID,
			"result":      state,
			"status":      "created",
		})
	})
}
