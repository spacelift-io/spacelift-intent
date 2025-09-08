package provider

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type describeArgs struct {
	Provider string `json:"provider"`
}

func Describe(providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("provider-describe"),
		Description: "Show the provider configuration, supported resources and supported data sources. Use this tool after finding a provider to understand its capabilities before resource creation - essential for discovering available resource types, data sources, and configuration requirements. Critical for the Configuration Phase workflow to validate resource definitions and ensure proper provider argument handling.",
		Annotations: i.ToolAnnotations("Show the provider config", i.READONLY|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'hashicorp/aws', 'spacelift-io/spacelift')",
				},
			},
			Required: []string{"provider"},
		},
	}, Handler: describe(providerManager)}
}

func describe(providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args describeArgs) (*mcp.CallToolResult, error) {
		// List data sources using provider manager
		dataSourceTypes, err := providerManager.ListDataSources(ctx, args.Provider)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list data sources: %v", err)), nil
		}

		resourceTypes, err := providerManager.ListResources(ctx, args.Provider)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list resources: %v", err)), nil
		}

		// TODO: implement, what the actual fuck, Claude?
		providerConfig := map[string]any{
			"properties": map[string]any{},
			"required":   []string{},
		}

		return i.RespondJSON(map[string]any{
			"provider":          args.Provider,
			"provider_config":   providerConfig,
			"data_source_types": dataSourceTypes,
			"resource_types":    resourceTypes,
		})
	})
}
