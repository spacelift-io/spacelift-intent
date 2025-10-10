// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package datasources

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type readArgs struct {
	ProviderName    string         `json:"provider"`
	DataSourceType  string         `json:"data_source_type"`
	Config          map[string]any `json:"config"`
	ProviderVersion string         `json:"provider_version"`
	ProviderConfig  map[string]any `json:"provider_config,omitempty"`
}

func (args readArgs) GetProvider() *types.ProviderConfig {
	return &types.ProviderConfig{
		Name:    args.ProviderName,
		Version: args.ProviderVersion,
		Config:  args.ProviderConfig,
	}
}

func Read(providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-datasources-read"),
		Description: "Read data from a data source of any type from any provider. " +
			"\n\nMANDATORY PREREQUISITE: You MUST call provider-search first to discover the provider and its available versions before using this tool. " +
			"Do not assume provider names or versions - always search first. " +
			"\n\nLOW risk read-only operation for querying external data during Discovery Phase. " +
			"Use this to retrieve existing infrastructure information, configuration values, " +
			"and external references needed for resource configuration. Does not modify state - " +
			"purely informational data retrieval through provider APIs. " +
			"\n\nArgument Handling: Set unknown/optional arguments to appropriate defaults: " +
			"strings to null or '', booleans to null or false, numbers to null or 0, arrays " +
			"to null or [], objects to null or {}. Ensure ALL required arguments are provided.",
		Annotations: i.ToolAnnotations("Read data from a data source", i.Readonly|i.Idempotent|i.OpenWorld),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'hashicorp/aws', 'hashicorp/random')",
				},
				"data_source_type": map[string]any{
					"type":        "string",
					"description": "The OpenTofu data source type to read (e.g., 'random_string', 'aws_ami')",
				},
				"config": map[string]any{
					"type":        "object",
					"description": "Configuration parameters for the data source",
				},
				"provider_version": map[string]any{
					"type":        "string",
					"description": "Provider version (e.g., '5.0.0')",
				},
			},
			Required: []string{"provider", "data_source_type", "config", "provider_version"},
		},
	}, Handler: read(providerManager)}
}

func read(providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args readArgs) (*mcp.CallToolResult, error) {
		// Read data source using provider manager
		stateResult, err := providerManager.ReadDataSource(ctx, args.GetProvider(), args.DataSourceType, args.Config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read data source: %v", err)), nil
		}

		// Handle empty state case
		if len(stateResult) == 0 {
			return mcp.NewToolResultText("{}"), nil
		}

		return i.RespondJSON(map[string]any{
			"provider":         args.GetProvider().Name,
			"provider_version": args.GetProvider().Version,
			"result":           stateResult,
		})
	})
}
