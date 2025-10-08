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
		// Search providers using registry client
		results, err := registryClient.SearchProviders(ctx, args.Query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search providers: %v", err)), nil
		}

		// Find the most popular provider
		var bestProvider map[string]any
		var maxPopularity float64 = -1

		for _, result := range results {
			if result.Popularity > maxPopularity {
				maxPopularity = result.Popularity
				bestProvider = map[string]any{
					"name":        result.Addr,
					"title":       result.Title,
					"description": result.Description,
					"version":     result.Version,
					"popularity":  result.Popularity,
				}
			}
		}

		if bestProvider == nil {
			return mcp.NewToolResultError("No providers found for query"), nil
		}

		return i.RespondJSON(map[string]any{
			"query":    args.Query,
			"provider": bestProvider,
		})
	})
}
