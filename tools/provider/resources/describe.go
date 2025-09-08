package resources

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type describeArgs struct {
	Provider     string `json:"provider"`
	ResourceType string `json:"resource_type"`
}

func Describe(providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("provider-resources-describe"),
		Description: "Get the schema and documentation for a specific resource type. " +
			"Essential for Configuration Completion Strategy - use this to identify ALL required " +
			"arguments, their types, and validation rules before resource creation. Critical for " +
			"auto-handling provider argument requirements and preventing 'expected X arguments, " +
			"got Y' errors through proper schema analysis. " +
			"\n\nArgument Completion: Use schema to determine appropriate defaults for missing " +
			"arguments: strings to null or '', booleans to null or false, numbers to null or 0, " +
			"arrays to null or [], objects to null or {}. " +
			"\n\nError Handling: When encountering argument mismatches, use Provider Argument " +
			"Count Mismatch format showing expected vs received counts with auto-resolution strategy.",
		Annotations: i.ToolAnnotations("Get schema for a specific resource type", i.READONLY|i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'hashicorp/aws', 'hashicorp/random')",
				},
				"resource_type": map[string]any{
					"type":        "string",
					"description": "The OpenTofu resource type (e.g., 'random_string', 'aws_instance')",
				},
			},
			Required: []string{"provider", "resource_type"},
		},
	}, Handler: describe(providerManager)}
}

func describe(providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args describeArgs) (*mcp.CallToolResult, error) {
		// Describe resource using provider manager
		description, err := providerManager.DescribeResource(ctx, args.Provider, args.ResourceType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to describe resource: %v", err)), nil
		}

		return i.RespondJSON(description)
	})
}
