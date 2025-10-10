// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type describeArgs struct {
	Provider string `json:"provider"`
	Version  string `json:"provider_version"`
}

func (args describeArgs) GetProvider() *types.ProviderConfig {
	return &types.ProviderConfig{
		Name:    args.Provider,
		Version: args.Version,
	}
}

func Describe(providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("provider-describe"),
		Description: "Show the provider configuration, supported resources and supported data sources. " +
			"\n\nMANDATORY PREREQUISITE: You MUST call provider-search first to discover the provider and its available versions before using this tool. " +
			"Do not assume provider names or versions - always search first. " +
			"\n\nUse this tool after finding a provider to understand its capabilities before resource creation - essential for discovering available resource types, data sources, and configuration requirements. Critical for the Configuration Phase workflow to validate resource definitions and ensure proper provider argument handling.",
		Annotations: i.ToolAnnotations("Show the provider config", i.Readonly|i.Idempotent),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'hashicorp/aws', 'spacelift-io/spacelift')",
				},
				"provider_version": map[string]any{
					"type":        "string",
					"description": "Provider version (e.g., '5.0.0')",
				},
			},
			Required: []string{"provider", "provider_version"},
		},
	}, Handler: describe(providerManager)}
}

func describe(providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args describeArgs) (*mcp.CallToolResult, error) {
		versions, err := providerManager.GetProviderVersions(ctx, args.Provider)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get provider versions: %v", err)), nil
		}
		versionsStrings := make([]string, len(versions))
		for i, v := range versions {
			versionsStrings[i] = v.Version
		}

		found := false
		availableVersions := make([]string, 0, len(versions))
		for _, v := range versions {
			availableVersions = append(availableVersions, v.Version)
			if v.Version == args.Version {
				found = true
			}
		}
		if !found {
			return mcp.NewToolResultError(fmt.Sprintf("Provider version '%s' not found for provider '%s'. Available versions: %v", args.Version, args.Provider, availableVersions)), nil
		}

		schema, confErr, err := providerManager.DescribeProvider(ctx, args.GetProvider())
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get provider schema: %v", err)), nil
		}

		dataSourceTypes := make([]string, 0, len(schema.DataSources))
		for dataSourceType := range schema.DataSources {
			dataSourceTypes = append(dataSourceTypes, dataSourceType)
		}

		resourceTypes := make([]string, 0, len(schema.Resources))
		for resourceType := range schema.Resources {
			resourceTypes = append(resourceTypes, resourceType)
		}

		return i.RespondJSON(map[string]any{
			"provider": map[string]any{
				"provider": args.Provider,
				"required": schema.Provider.Required,
				"version":  schema.Version,
			},
			"data_source_types": dataSourceTypes,
			"resource_types":    resourceTypes,
			"config_error":      confErr,
		})
	})
}
