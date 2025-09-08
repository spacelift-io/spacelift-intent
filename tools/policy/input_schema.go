package policy

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type inputSchemaArgs struct{}

// InputSchema provides information about the expected input schema for policies
func InputSchema() (mcp.Tool, i.ToolHandler) {
	return mcp.Tool{
		Name:        string("policy-input-schema"),
		Description: "Get the input schema that policies receive for evaluation. LOW risk read-only operation essential for policy development. Use this to understand the data structure available to policies for validation logic, including resource types, operations, and state information.",
		Annotations: i.ToolAnnotations("Get the input schema that policies receive for evaluation", i.OPEN_WORLD),
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
	}, inputSchema()
}

func inputSchema() i.ToolHandler {
	return mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, args inputSchemaArgs) (*mcp.CallToolResult, error) {
		// Create a sample input to show the schema
		sampleInput := types.PolicyInput{
			Resource: types.ResourceOperationInput{
				ResourceID:   "my-instance",
				ResourceType: "aws_instance",
				Provider:     "aws",
				Operation:    "create",
				CurrentState: map[string]any{
					"id":            "i-1234567890abcdef0",
					"instance_type": "t2.micro",
					"tags": map[string]any{
						"Name": "old-server",
					},
				},
				ProposedState: map[string]any{
					"instance_type": "t3.small",
					"tags": map[string]any{
						"Name":        "new-server",
						"Environment": "production",
					},
				},
			},
		}

		inputJSON, _ := json.MarshalIndent(sampleInput, "", "  ")

		output := "Policy Input Schema:\n\n"
		output += "Policies receive the following input structure for evaluation:\n\n"
		output += "```json\n" + string(inputJSON) + "\n```\n\n"

		output += "Field Descriptions:\n"
		output += "- resource: The resource being operated on (e.g., 'my-instance')\n"
		output += "  - resource_type: The type of resource being operated on (e.g., 'aws_instance', 'azurerm_virtual_machine')\n"
		output += "  - provider: The provider name (e.g., 'aws', 'azurerm', 'google')\n"
		output += "  - operation: The operation being performed ('create', 'update', 'delete', 'import', 'refresh')\n"
		output += "  - current_state: The current/previous state of the resource (null for create operations)\n"
		output += "  - proposed_state: The proposed new state of the resource (null for delete operations)\n\n"

		output += "Expected Policy Output Schema:\n\n"
		output += "Policies must return an object with the following structure:\n\n"
		output += "```json\n"
		output += "{\n"
		output += "  \"deny\": [\n"
		output += "    \"Description of the denial\"\n"
		output += "  ],\n"
		output += "  \"allow\": [\n"
		output += "    \"Description of the allowance\"\n"
		output += "  ]\n"
		output += "}\n"
		output += "```\n\n"

		output += "Example Rego Policy:\n\n"
		output += "```rego\n"
		output += "package policy\n\n"
		output += "deny[message] {\n"
		output += "    input.resource.operation == \"create\"\n"
		output += "    input.resource.resource_type == \"aws_instance\"\n"
		output += "    input.resource.proposed_state.instance_type == \"t2.large\"\n"
		output += "    message := \"t2.large instances are not allowed in production\"\n"
		output += "}\n\n"
		output += "allow[message] {\n"
		output += "    input.resource.operation == \"create\"\n"
		output += "    input.resource.resource_type == \"aws_instance\"\n"
		output += "    input.resource.proposed_state.instance_type == \"t2.nano\"\n"
		output += "    input.resource.proposed_state.region == \"us-west-2\"\n"
		output += "    message := \"allow t2.nano instances in us-west-2\"\n"
		output += "}\n\n"
		output += "```"

		return mcp.NewToolResultText(output), nil
	})
}
