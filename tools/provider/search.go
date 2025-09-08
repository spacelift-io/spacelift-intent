package provider

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type searchArgs struct {
	Query string `json:"query"`
}

// Search creates a tool for searching OpenTofu providers in the registry.
func Search(registryClient types.RegistryClient) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("provider-search"),
		Description: "Search for an available provider in the OpenTofu registry. Use this tool to discover providers before resource creation - essential for finding the correct provider namespace (e.g., 'hashicorp/aws', 'hashicorp/random') needed for infrastructure operations. Returns the most popular matching provider with its full address, version, and metadata. Present results with clear provider identification and next steps for schema analysis.",
		Annotations: i.ToolAnnotations("Search for a provider", i.READONLY|i.IDEMPOTENT|i.OPEN_WORLD),
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
