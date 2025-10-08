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

package resources

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type importArgs struct {
	ImportID        string         `json:"import_id"`
	DestinationID   string         `json:"destination_id"`
	Provider        string         `json:"provider"`
	ResourceType    string         `json:"resource_type"`
	ProviderVersion *string        `json:"provider_version,omitempty"`
	ProviderConfig  map[string]any `json:"provider_config,omitempty"`
}

func (args importArgs) GetProvider() *types.ProviderConfig {
	return &types.ProviderConfig{
		Name:    args.Provider,
		Version: args.ProviderVersion,
		Config:  args.ProviderConfig,
	}
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
		Annotations: i.ToolAnnotations("Import an external resource resource", i.Idempotent|i.OpenWorld),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"import_id": map[string]any{
					"type":        "string",
					"description": "The provider-specific identifier for the infrastructure you want to import",
				},
				"destination_id": map[string]any{
					"type":        "string",
					"description": "This is the resource name type of identifier we will give the resource in state",
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
			Required: []string{"import_id", "destination_id", "provider", "resource_type"},
		},
	}, Handler: _import(storage, providerManager)}
}

func _import(storage types.Storage, providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args importArgs) (*mcp.CallToolResult, error) {
		// Check if ID already exists
		existingState, err := storage.GetState(ctx, args.DestinationID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to check existing state: %v", err)), nil
		}
		if existingState != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Resource with ID '%s' already exists", args.DestinationID)), nil
		}

		input := types.ResourceOperationInput{
			ResourceID:    args.DestinationID,
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
		state, err := providerManager.ImportResource(ctx, args.GetProvider(), args.ResourceType, args.ImportID)
		if err != nil {
			err = fmt.Errorf("failed to import resource: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Handle empty state case - for import, this indicates the resource doesn't exist
		// TODO: Figure out if we want to handle empty state case for import here or in the providerManager
		if len(state) == 0 {
			err = fmt.Errorf("resource with ID '%s' does not exist or returned empty state", args.ImportID)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Get actual provider version
		version, err := providerManager.GetProviderVersion(ctx, args.Provider)
		if err != nil {
			// Fallback to "latest" if we can't get the version
			version = "latest"
		}

		// Persist state to database
		record := types.StateRecord{
			ResourceID:   args.DestinationID,
			Provider:     args.Provider,
			Version:      version,
			ResourceType: args.ResourceType,
			State:        state,
		}

		// Add operation context for automatic history tracking
		ctx = context.WithValue(ctx, types.OperationContextKey, "import")
		ctx = context.WithValue(ctx, types.ChangedByContextKey, "mcp-user")

		if err = storage.SaveState(ctx, record); err != nil {
			err = fmt.Errorf("failed to save state: %w", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		operation.ProposedState = state

		return RespondJSON(map[string]any{
			"import_id":      args.ImportID,
			"destination_id": args.DestinationID,
			"result":         state,
			"status":         "imported",
			"message":        "resource successfully imported",
		})
	})
}
