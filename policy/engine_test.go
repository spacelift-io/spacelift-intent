package policy

import (
	"context"
	"testing"

	"spacelift-intent-mcp/types"
)

func TestEngine_EvaluatePolicies_AllowAlways(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	// Policy that always allows
	policy := types.Policy{
		ID:   "test-allow-always",
		Name: "Allow Always Test",
		RegoCode: `package policy
allow["always"] { true }`,
		Enabled: true,
	}

	// Empty input (doesn't matter for this test)
	input := types.PolicyInput{
		Resource: types.ResourceOperationInput{},
	}

	result, err := engine.EvaluatePolicies(ctx, []types.Policy{policy}, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have one allow message
	if len(result.Allow) != 1 {
		t.Errorf("Expected 1 allow result, got %d", len(result.Allow))
	}

	if len(result.Allow) > 0 && result.Allow[0] != "always" {
		t.Errorf("Expected allow message 'always', got '%s'", result.Allow[0])
	}

	// Should have no deny messages
	if len(result.Deny) != 0 {
		t.Errorf("Expected 0 deny results, got %d", len(result.Deny))
	}
}

func TestEngine_EvaluatePolicies_DenyAlways(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	// Policy that always denies
	policy := types.Policy{
		ID:   "test-deny-always",
		Name: "Deny Always Test",
		RegoCode: `package policy
deny["never allowed"] { true }`,
		Enabled: true,
	}

	input := types.PolicyInput{
		Resource: types.ResourceOperationInput{},
	}

	result, err := engine.EvaluatePolicies(ctx, []types.Policy{policy}, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have one deny message
	if len(result.Deny) != 1 {
		t.Errorf("Expected 1 deny result, got %d", len(result.Deny))
	}

	if len(result.Deny) > 0 && result.Deny[0] != "never allowed" {
		t.Errorf("Expected deny message 'never allowed', got '%s'", result.Deny[0])
	}

	// Should have no allow messages
	if len(result.Allow) != 0 {
		t.Errorf("Expected 0 allow results, got %d", len(result.Allow))
	}
}

func TestEngine_EvaluatePolicies_MultipleRules(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	// Policy with multiple allow and deny rules
	policy := types.Policy{
		ID:   "test-multiple-rules",
		Name: "Multiple Rules Test",
		RegoCode: `package policy
allow["rule1"] { true }
allow["rule2"] { true }
deny["violation1"] { true }
deny["violation2"] { true }`,
		Enabled: true,
	}

	input := types.PolicyInput{
		Resource: types.ResourceOperationInput{},
	}

	result, err := engine.EvaluatePolicies(ctx, []types.Policy{policy}, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have two allow messages
	if len(result.Allow) != 2 {
		t.Errorf("Expected 2 allow results, got %d", len(result.Allow))
	}

	// Should have two deny messages
	if len(result.Deny) != 2 {
		t.Errorf("Expected 2 deny results, got %d", len(result.Deny))
	}

	// Check specific messages exist
	allowMap := make(map[string]bool)
	for _, msg := range result.Allow {
		allowMap[msg] = true
	}
	if !allowMap["rule1"] || !allowMap["rule2"] {
		t.Errorf("Expected allow messages 'rule1' and 'rule2', got %v", result.Allow)
	}

	denyMap := make(map[string]bool)
	for _, msg := range result.Deny {
		denyMap[msg] = true
	}
	if !denyMap["violation1"] || !denyMap["violation2"] {
		t.Errorf("Expected deny messages 'violation1' and 'violation2', got %v", result.Deny)
	}
}

func TestEngine_EvaluatePolicies_ConditionalPolicy(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	// Policy with conditional logic
	policy := types.Policy{
		ID:   "test-conditional",
		Name: "Conditional Policy Test",
		RegoCode: `package policy
allow["aws allowed"] { input.resource.provider == "aws" }
deny["random not allowed"] { input.resource.provider == "random" }`,
		Enabled: true,
	}

	// Test with AWS provider - should allow
	input := types.PolicyInput{
		Resource: types.ResourceOperationInput{
			Provider: "aws",
		},
	}

	result, err := engine.EvaluatePolicies(ctx, []types.Policy{policy}, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Allow) != 1 || result.Allow[0] != "aws allowed" {
		t.Errorf("Expected allow 'aws allowed', got %v", result.Allow)
	}
	if len(result.Deny) != 0 {
		t.Errorf("Expected no deny results, got %v", result.Deny)
	}

	// Test with random provider - should deny
	input.Resource.Provider = "random"
	result, err = engine.EvaluatePolicies(ctx, []types.Policy{policy}, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Deny) != 1 || result.Deny[0] != "random not allowed" {
		t.Errorf("Expected deny 'random not allowed', got %v", result.Deny)
	}
	if len(result.Allow) != 0 {
		t.Errorf("Expected no allow results, got %v", result.Allow)
	}
}

func TestEngine_EvaluatePolicies_DisabledPolicy(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	// Disabled policy should not be evaluated
	policy := types.Policy{
		ID:   "test-disabled",
		Name: "Disabled Policy Test",
		RegoCode: `package policy
deny["should not appear"] { true }`,
		Enabled: false, // Disabled
	}

	input := types.PolicyInput{
		Resource: types.ResourceOperationInput{},
	}

	result, err := engine.EvaluatePolicies(ctx, []types.Policy{policy}, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have no results since policy is disabled
	if len(result.Allow) != 0 {
		t.Errorf("Expected 0 allow results, got %d", len(result.Allow))
	}
	if len(result.Deny) != 0 {
		t.Errorf("Expected 0 deny results, got %d", len(result.Deny))
	}
}

func TestEngine_EvaluatePolicies_MultiplePolicies(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	// Multiple policies - results should be merged
	policy1 := types.Policy{
		ID:   "test-policy-1",
		Name: "Policy 1",
		RegoCode: `package policy
allow["from policy 1"] { true }`,
		Enabled: true,
	}

	policy2 := types.Policy{
		ID:   "test-policy-2",
		Name: "Policy 2",
		RegoCode: `package policy
deny["from policy 2"] { true }`,
		Enabled: true,
	}

	input := types.PolicyInput{
		Resource: types.ResourceOperationInput{},
	}

	result, err := engine.EvaluatePolicies(ctx, []types.Policy{policy1, policy2}, input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should merge results from both policies
	if len(result.Allow) != 1 || result.Allow[0] != "from policy 1" {
		t.Errorf("Expected allow 'from policy 1', got %v", result.Allow)
	}
	if len(result.Deny) != 1 || result.Deny[0] != "from policy 2" {
		t.Errorf("Expected deny 'from policy 2', got %v", result.Deny)
	}
}

func TestEngine_ValidatePolicy_ValidPolicy(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	validPolicy := `package policy
allow["valid"] { true }`

	err := engine.ValidatePolicy(ctx, validPolicy)
	if err != nil {
		t.Errorf("Expected valid policy to pass validation, got error: %v", err)
	}
}

func TestEngine_ValidatePolicy_InvalidPolicy(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	invalidPolicy := `package policy
invalid syntax here`

	err := engine.ValidatePolicy(ctx, invalidPolicy)
	if err == nil {
		t.Error("Expected invalid policy to fail validation")
	}
}
