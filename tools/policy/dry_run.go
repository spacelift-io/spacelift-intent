package policy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type dryRunArgs struct {
	PolicyID      string                 `json:"policy_id"`
	ResourceType  string                 `json:"resource_type"`
	Provider      string                 `json:"provider"`
	Operation     string                 `json:"operation"`
	CurrentState  map[string]interface{} `json:"current_state"`
	ProposedState map[string]interface{} `json:"proposed_state"`
}

// DryRun performs a dry run of a policy against sample input
func DryRun(storage types.Storage, engine types.PolicyEngine) (mcp.Tool, i.ToolHandler) {
	return mcp.Tool{
		Name: string("policy-dry-run"),
		Description: "Perform a dry run of a policy against sample input data. " +
			"LOW risk operation for testing policy logic before deployment. Essential for " +
			"Configuration Phase validation - use this to verify policy rules work correctly " +
			"with expected resource configurations. " +
			"\n\nPresentation: Present results using structured format with Violations, " +
			"Warnings, and Debug sections. Include both human-readable summary and JSON format " +
			"for programmatic access. " +
			"\n\nCritical for ensuring policies catch violations without blocking legitimate operations.",
		Annotations: i.ToolAnnotations("Perform a dry run of a policy against sample input data", i.OPEN_WORLD),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"policy_id": map[string]any{
					"type":        "string",
					"description": "Policy ID to test",
				},
				"resource_type": map[string]any{
					"type":        "string",
					"description": "Resource type (e.g., 'aws_instance')",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider name (e.g., 'aws')",
				},
				"operation": map[string]any{
					"type":        "string",
					"description": "Operation type",
					"enum":        []string{"create", "update", "delete", "import", "refresh"},
				},
				"current_state": map[string]any{
					"type":        "object",
					"description": "Current resource state (for update/delete operations)",
				},
				"proposed_state": map[string]any{
					"type":        "object",
					"description": "Proposed resource state (for create/update operations)",
				},
			},
			Required: []string{"policy_id", "resource_type", "provider", "operation"},
		},
	}, dryRun(storage, engine)
}

func dryRun(storage types.Storage, engine types.PolicyEngine) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args dryRunArgs) (*mcp.CallToolResult, error) {
		// Build policy input
		input := types.PolicyInput{
			Resource: types.ResourceOperationInput{
				ResourceType:  args.ResourceType,
				Provider:      args.Provider,
				Operation:     args.Operation,
				CurrentState:  args.CurrentState,
				ProposedState: args.ProposedState,
			},
		}

		// Get and evaluate policy
		policy, err := storage.GetPolicy(ctx, args.PolicyID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get policy: %v", err)), nil
		}

		if policy == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Policy not found: %s", args.PolicyID)), nil
		}

		result, err := engine.EvaluatePolicies(ctx, []types.Policy{*policy}, input)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Policy evaluation failed: %v", err)), nil
		}

		// Format output
		output := fmt.Sprintf("Dry run results for policy '%s':\n\n", policy.Name)

		output += "Input:\n"
		output += "  Resource:\n"
		output += fmt.Sprintf("    Type: %s\n", args.ResourceType)
		output += fmt.Sprintf("    Provider: %s\n", args.Provider)
		output += fmt.Sprintf("    Operation: %s\n", args.Operation)

		if args.CurrentState != nil {
			currStateJSON, _ := json.MarshalIndent(args.CurrentState, "  ", "  ")
			output += fmt.Sprintf("    Current State: %s\n", string(currStateJSON))
		}

		if args.ProposedState != nil {
			propStateJSON, _ := json.MarshalIndent(args.ProposedState, "  ", "  ")
			output += fmt.Sprintf("    Proposed State: %s\n", string(propStateJSON))
		}

		output += "\nResults:\n"

		if len(result.Deny) > 0 {
			output += "  Denials:\n"
			for _, denyMessage := range result.Deny {
				output += fmt.Sprintf("    - %s\n", denyMessage)
			}
		} else {
			output += "  Denials: None\n"
		}

		if len(result.Allow) > 0 {
			output += "  Allowances:\n"
			for _, allowMessage := range result.Allow {
				output += fmt.Sprintf("    - %s\n", allowMessage)
			}
		} else {
			output += "  Allowances: None\n"
		}

		// Also return as JSON for programmatic access
		resultJSON, _ := json.MarshalIndent(result, "", "  ")

		// Return multiple content items
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: output,
				},
				mcp.TextContent{
					Type: "text",
					Text: "JSON format:\n" + string(resultJSON),
				},
			},
		}, nil
	})
}
