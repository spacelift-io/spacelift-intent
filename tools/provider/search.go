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

type searchArgs struct {
	Query string `json:"query"`
}

// Search creates a tool for searching OpenTofu providers in the registry.
func Search(registryClient types.RegistryClient) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("provider-search"),
		Description: "Search for an available provider in the OpenTofu registry. Use this tool to discover providers before resource creation - essential for finding the correct provider namespace (e.g., 'hashicorp/aws', 'hashicorp/random') needed for infrastructure operations. Returns the most popular matching provider with its full address, version, and metadata. Present results with clear provider identification and next steps for schema analysis.",
		Annotations: i.ToolAnnotations("Search for a provider", i.Readonly|i.Idempotent|i.OpenWorld),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query for provider names (e.g., 'aws', 'google', 'random')",
				},
			},
			Required: []string{"query"},
		},
	}, Handler: search(registryClient)}
}

func search(registryClient types.RegistryClient) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args searchArgs) (*mcp.CallToolResult, error) {
		result, err := SearchForProvider(ctx, registryClient, args.Query)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return i.RespondJSON(result)
	})
}

// SearchForProvider searches for a provider and returns the best match
func SearchForProvider(ctx context.Context, registryClient types.RegistryClient, providerName string) (*types.ProviderSearchToolResult, error) {
	results, err := registryClient.SearchProviders(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to search providers: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no providers found for query: %s", providerName)
	}

	// Find the most popular provider
	var bestProvider *types.ProviderSearchResult
	var maxPopularity float64 = -1

	for i := range results {
		if results[i].Popularity > maxPopularity {
			maxPopularity = results[i].Popularity
			bestProvider = &results[i]
		}
	}

	if bestProvider == nil {
		return nil, fmt.Errorf("no suitable provider found for query: %s", providerName)
	}

	return &types.ProviderSearchToolResult{
		Query:    providerName,
		Provider: *bestProvider,
	}, nil
}
