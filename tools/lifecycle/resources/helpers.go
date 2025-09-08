package resources

import (
	"encoding/json"
	"fmt"

	"spacelift-intent-mcp/types"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

func newResourceOperation(input types.ResourceOperationInput) (types.ResourceOperation, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return types.ResourceOperation{}, fmt.Errorf("Failed to generate operation ID: %v", err)
	}

	return types.ResourceOperation{
		ID:                     id.String(),
		ResourceOperationInput: input,
	}, nil
}

// RespondJSON returns a JSON response
func RespondJSON(response map[string]any) (*mcp.CallToolResult, error) {
	result, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
