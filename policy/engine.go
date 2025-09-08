package policy

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"

	"spacelift-intent-mcp/types"
)

// Engine implements the PolicyEngine interface using OPA/Rego
type Engine struct{}

// NewEngine creates a new policy engine
func NewEngine() types.PolicyEngine {
	return &Engine{}
}

// EvaluatePolicies evaluates multiple policies against the provided input
func (e *Engine) EvaluatePolicies(ctx context.Context, policies []types.Policy, input types.PolicyInput) (types.PolicyResult, error) {
	result := types.PolicyResult{}

	// Evaluate each enabled policy
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		policyResult, err := e.evaluateSinglePolicy(ctx, policy, input)
		if err != nil {
			continue
		}

		// Merge results
		result.Deny = append(result.Deny, policyResult.Deny...)
		result.Allow = append(result.Allow, policyResult.Allow...)
	}

	return result, nil
}

// evaluateSinglePolicy evaluates a single policy
func (e *Engine) evaluateSinglePolicy(ctx context.Context, policy types.Policy, input types.PolicyInput) (types.PolicyResult, error) {
	result := types.PolicyResult{}

	// Create and prepare Rego query
	query, err := rego.New(
		rego.Query("data.policy"),
		rego.Module("policy.rego", policy.RegoCode),
	).PrepareForEval(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to prepare policy %s: %w", policy.Name, err)
	}

	// Evaluate policy
	rs, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return result, fmt.Errorf("failed to evaluate policy %s: %w", policy.Name, err)
	}

	// Process results
	if len(rs) > 0 && len(rs[0].Expressions) > 0 {
		if outputMap, ok := rs[0].Expressions[0].Value.(map[string]interface{}); ok {
			result.Deny = extractMessages(outputMap, "deny")
			result.Allow = extractMessages(outputMap, "allow")
		}
	}

	return result, nil
}

// extractMessages extracts messages of a specific type from policy output
func extractMessages(outputMap map[string]interface{}, messageType string) []string {
	var messages []string

	if items, ok := outputMap[messageType].([]interface{}); ok {
		for _, item := range items {
			if itemString, ok := item.(string); ok {
				messages = append(messages, itemString)
			}
		}
	}

	return messages
}

// ValidatePolicy validates that a Rego policy can be compiled
func (e *Engine) ValidatePolicy(ctx context.Context, regoCode string) error {
	// Try to compile the policy
	r := rego.New(
		rego.Query("data.policy"),
		rego.Module("policy.rego", regoCode),
	)

	// Prepare for evaluation (this will catch compilation errors)
	_, err := r.PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	return nil
}
