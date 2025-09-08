package policy

import (
	"context"
	"fmt"

	"spacelift-intent-mcp/types"
)

// Evaluator handles policy evaluation logic
type Evaluator struct {
	storage types.Storage
	engine  types.PolicyEngine
}

// NewEvaluator creates a new policy evaluator
func NewEvaluator(storage types.Storage, engine types.PolicyEngine) *Evaluator {
	return &Evaluator{
		storage: storage,
		engine:  engine,
	}
}

// Evaluate loads and evaluates all enabled policies
func (e *Evaluator) Evaluate(ctx context.Context, input types.PolicyInput) (*types.PolicyResult, error) {
	policies, err := e.storage.ListPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	// Filter to only enabled policies
	var enabledPolicies []types.Policy
	for _, policy := range policies {
		if policy.Enabled {
			enabledPolicies = append(enabledPolicies, policy)
		}
	}

	if len(enabledPolicies) == 0 {
		return nil, nil
	}

	result, err := e.engine.EvaluatePolicies(ctx, enabledPolicies, input)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	return &result, nil
}
