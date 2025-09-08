package resources

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	i "spacelift-intent-mcp/tools/internal"
	"spacelift-intent-mcp/types"
)

type resumeArgs struct {
	OperationID string `json:"operation_id"`
}

func Resume(storage types.Storage, providerManager types.ProviderManager, policyEvaluator interface{}) (mcp.Tool, i.ToolHandler) {
	return mcp.Tool{
		Name: string("lifecycle-resources-resume"),
		Description: "Resume a suspended operation that was halted due to policy evaluation or other reasons. " +
			"MEDIUM risk operation that continues the previously suspended resource operation " +
			"(create/update/delete). Use this to proceed with operations that require manual approval " +
			"or have been temporarily halted for compliance review.",
		Annotations: i.ToolAnnotations("Resume a suspended operation", i.OPEN_WORLD),
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"operation_id": map[string]any{
					"type":        "string",
					"description": "Unique identifier for this suspended operation (must be unique and intuitive)",
				},
			},
			Required: []string{"operation_id"},
		},
	}, resume(storage, providerManager, policyEvaluator)
}

func resume(_ types.Storage, _ types.ProviderManager, _ interface{}) i.ToolHandler {
	return mcp.NewTypedToolHandler(func(_ context.Context, _ mcp.CallToolRequest, _ resumeArgs) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("Not supported"), nil
	})
}
