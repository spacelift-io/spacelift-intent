package dependencies

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

func Add(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name: string("lifecycle-resources-dependencies-add"),
		Description: `Add a dependency relationship between two resources with optional explanation and field mappings. 
**LOW risk** operation for establishing resource ordering and dependency chains. 
Essential for ensuring proper creation/destruction sequence and preventing dependency violations.
Use this after resource creation to document relationships that affect deployment order and infrastructure stability.

**Example**:
instance
       ↓ 
instance-profile 
       ↓ 
ec2-role 
---
Dependencies:
1. from_resource_id = instance; to_resource_id = instance_profile
2. from_resource_id = instance_profile; to_resource_id = ec2-role
ec2-role is a dependency for the instance_profile, and instance_profile is a dependent for ec2-role (relies on it)`,
		Annotations: i.ToolAnnotations("Explicitly add a dependency between two resources", i.IDEMPOTENT),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"from_resource_id": map[string]any{
					"type":        "string",
					"description": "The resource that depends on another",
				},
				"to_resource_id": map[string]any{
					"type":        "string",
					"description": "The resource that is depended upon",
				},
				"dependency_type": map[string]any{
					"type":        "string",
					"description": "Type of dependency: 'explicit', 'implicit', or 'data_source'",
					"enum":        []string{"explicit", "implicit", "data_source"},
				},
				"explanation": map[string]any{
					"type":        "string",
					"description": "Optional explanation of why this dependency exists",
				},
				"field_mappings": map[string]any{
					"type":        "array",
					"description": "Optional array of field mappings showing which fields impact which",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"source_field": map[string]any{
								"type":        "string",
								"description": "Field in the dependent resource",
							},
							"target_field": map[string]any{
								"type":        "string",
								"description": "Field in the dependency target",
							},
							"description": map[string]any{
								"type":        "string",
								"description": "How this field dependency works",
							},
						},
						"required": []string{"source_field", "target_field"},
					},
				},
			},
			Required: []string{"from_resource_id", "to_resource_id", "dependency_type"},
		},
	}, Handler: add(storage)}
}

func add(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, edge types.DependencyEdge) (*mcp.CallToolResult, error) {
		if err := storage.AddDependency(ctx, edge); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to add dependency: %v", err)), nil
		}

		return i.RespondJSON(struct {
			types.DependencyEdge
			Status string `json:"status"`
		}{edge, "added"})
	})
}
