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

package datasources

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

type describeArgs struct {
	Provider        string  `json:"provider"`
	DataSourceType  string  `json:"data_source_type"`
	ProviderVersion *string `json:"provider_version,omitempty"`
}

func (args describeArgs) GetProvider() *types.ProviderConfig {
	return &types.ProviderConfig{
		Name:    args.Provider,
		Version: args.ProviderVersion,
	}
}

func Describe(providerManager types.ProviderManager) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("provider-datasources-describe"),
		Description: "Get the schema and documentation for a specific data source type. " +
			"Essential for Discovery Phase - use this to understand data source capabilities, " +
			"required arguments, and available outputs before querying external data. Critical " +
			"for Configuration Phase to validate data source definitions and ensure proper " +
			"argument handling. " +
			"\n\nArgument Completion: Use schema to set missing arguments to appropriate " +
			"defaults: strings to null or '', booleans to null or false, numbers to null or 0, " +
			"arrays to null or [], objects to null or {}. " +
			"\n\nLOW risk read-only operation for schema analysis.",
		Annotations: i.ToolAnnotations("Get schema for a specific data source type", i.Readonly|i.Idempotent),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'spacelift-io/spacelift', 'hashicorp/aws')",
				},
				"data_source_type": map[string]any{
					"type":        "string",
					"description": "The type of the data source (e.g., 'spacelift_context', 'random_string', 'aws_ami')",
				},
			},
			Required: []string{"provider", "data_source_type"},
		},
	}, Handler: describe(providerManager)}
}

func describe(providerManager types.ProviderManager) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args describeArgs) (*mcp.CallToolResult, error) {
		// Describe data source using provider manager
		description, err := providerManager.DescribeDataSource(ctx, args.GetProvider(), args.DataSourceType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to describe data source: %v", err)), nil
		}

		return i.RespondJSON(description)
	})
}
