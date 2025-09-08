package resources

import (
	"context"
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

func evaluatePolicies(ctx context.Context, policyEvaluator types.PolicyEvaluator, input types.ResourceOperationInput, operation *types.ResourceOperation) (err error) {
	var policyResult *types.PolicyResult

	defer func() {
		if err != nil {
			errMessage := err.Error()
			operation.Failed = &errMessage
		}
		if policyResult != nil {
			operation.PolicyResult = *policyResult
		}
	}()

	policyResult, err = policyEvaluator.Evaluate(ctx, types.PolicyInput{Resource: input})
	if err != nil {
		return fmt.Errorf("Failed to evaluate policies: %w", err)
	}

	if policyResult == nil {
		// No policies found, by default allow
		return nil
	}

	if isDenied, message := policyResult.IsDenied(); isDenied {
		return fmt.Errorf("Policy violation: %v", message)
	}

	return nil
}

// RespondJSONWithPostPolicyResult returns a JSON response that includes policy results
func RespondJSONWithPolicyResult(response map[string]any, operation *types.ResourceOperation) (*mcp.CallToolResult, error) {
	if len(operation.PolicyResult.Deny) > 0 {
		response["policy_deny"] = operation.PolicyResult.Deny
	}

	if len(operation.PolicyResult.Allow) > 0 {
		response["policy_allow"] = operation.PolicyResult.Allow
	}

	result, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(result)), nil
}
