package dependencies

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "github.com/spacelift-io/spacelift-intent/tools/internal"
	"github.com/spacelift-io/spacelift-intent/types"
)

func Get(storage types.Storage) i.Tool {
	return i.Tool{Tool: mcp.Tool{
		Name:        string("lifecycle-resources-dependencies-get"),
		Description: "Get dependency relationships for a resource, showing both what it depends on (dependencies) and what depends on it (dependents). LOW risk operation for understanding resource relationships and dependency chains. Essential for dependency analysis, troubleshooting circular dependencies, and planning changes that might affect related resources.",
		Annotations: i.ToolAnnotations("Get dependency relationships for a resource", i.READONLY),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"resource_id": map[string]any{
					"type":        "string",
					"description": "The resource ID to get dependencies for (can be used as either from_resource_id or to_resource_id)",
				},
				"direction": map[string]any{
					"type":        "string",
					"description": "Direction of dependencies to retrieve: 'dependencies' (what this resource depends on), 'dependents' (what depends on this resource), or 'both' (default)",
					"enum":        []string{"dependencies", "dependents", "both"},
					"default":     "both",
				},
			},
			Required: []string{"resource_id"},
		},
	}, Handler: get(storage)}
}

type GetDependenciesRequest struct {
	ResourceID string `json:"resource_id"`
	Direction  string `json:"direction"`
}

type GetDependenciesResponse struct {
	ResourceID   string                 `json:"resource_id"`
	Dependencies []types.DependencyEdge `json:"dependencies,omitempty"`
	Dependents   []types.DependencyEdge `json:"dependents,omitempty"`
	Summary      DependencySummary      `json:"summary"`
}

type DependencySummary struct {
	DependenciesCount int `json:"dependencies_count"`
	DependentsCount   int `json:"dependents_count"`
	TotalCount        int `json:"total_count"`
}

func get(storage types.Storage) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, req GetDependenciesRequest) (*mcp.CallToolResult, error) {
		response := GetDependenciesResponse{
			ResourceID: req.ResourceID,
		}

		var err error

		// Get dependencies (what this resource depends on)
		if req.Direction == "dependencies" || req.Direction == "both" {
			response.Dependencies, err = storage.GetDependencies(ctx, req.ResourceID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get dependencies: %v", err)), nil
			}
		}

		// Get dependents (what depends on this resource)
		if req.Direction == "dependents" || req.Direction == "both" {
			response.Dependents, err = storage.GetDependents(ctx, req.ResourceID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get dependents: %v", err)), nil
			}
		}

		// Calculate summary
		response.Summary = DependencySummary{
			DependenciesCount: len(response.Dependencies),
			DependentsCount:   len(response.Dependents),
			TotalCount:        len(response.Dependencies) + len(response.Dependents),
		}

		return i.RespondJSON(response)
	})
}
