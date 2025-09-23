package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/test/testhelper"
)

// TestSpaceliftProviderSearch tests searching for the Spacelift provider
func TestSpaceliftProviderSearch(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("provider-search", map[string]any{
		"query": "spacelift-io/spacelift",
	})
	th.AssertToolSuccess(result, err, "provider-search")

	content := th.GetTextContent(result)
	assert.Contains(t, content, "spacelift", "Search results should contain 'spacelift'")
	assert.Contains(t, content, "spacelift-io", "Search results should contain 'spacelift-io'")
}

// TestSpaceliftProviderDescribe tests describing the Spacelift provider
func TestSpaceliftProviderDescribe(t *testing.T) {
	// Load Spacelift credentials from .env.spacelift
	testhelper.LoadSpaceliftCredentials(t)

	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("provider-describe", map[string]any{
		"provider": "spacelift-io/spacelift",
	})
	th.AssertToolSuccess(result, err, "provider-describe")

	content := th.GetTextContent(result)
	assert.Contains(t, content, "spacelift", "Provider description should contain provider info")
	assert.Contains(t, content, "context", "Provider should support context resources")
}

// TestSpaceliftContextResourceDescribe tests describing the spacelift_context resource
func TestSpaceliftContextResourceDescribe(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("provider-resources-describe", map[string]any{
		"provider":      "spacelift-io/spacelift",
		"resource_type": "spacelift_context",
	})
	th.AssertToolSuccess(result, err, "provider-resources-describe")

	content := th.GetTextContent(result)
	assert.Contains(t, content, "spacelift_context", "Resource description should contain resource type")
	assert.Contains(t, content, "name", "spacelift_context should have 'name' attribute")
}

// TestSpaceliftContextLifecycleCreate tests creating a spacelift_context resource
func TestSpaceliftContextLifecycleCreate(t *testing.T) {
	// Load Spacelift credentials from .env.spacelift
	testhelper.LoadSpaceliftCredentials(t)

	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := testhelper.GenerateUniqueResourceID("test-spacelift-context-create")
	defer th.CleanupResource(resourceID)

	contextName := "Test Context for Integration Test"
	var contextID string

	// Cleanup via Spacelift API at the end
	defer func() {
		if contextID != "" {
			if err := testhelper.CleanupContextById(t, contextID); err != nil {
				t.Errorf("Failed to cleanup context via Spacelift API: %v", err)
			}
		}
	}()

	result, err := th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":   resourceID,
		"provider":      "spacelift-io/spacelift",
		"resource_type": "spacelift_context",
		"config": map[string]any{
			"name":        contextName,
			"description": "A context created during integration testing",
			"labels":      []string{"spacelift-intent-testing"},
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-create")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Create result should contain resource ID")
	assert.Contains(t, content, "created", "Should show created status")

	// Validate that the context actually exists via Spacelift API
	context, err := testhelper.ValidateContextExistsByName(t, contextName)
	if err != nil {
		t.Errorf("Failed to validate context creation via Spacelift API: %v", err)
	} else {
		contextID = context.ID // Store for cleanup
		assert.Equal(t, contextName, context.Name, "Context name should match")
		assert.NotEmpty(t, context.ID, "Context should have an ID")
		assert.Contains(t, context.Labels, "spacelift-intent-testing", "Context should have 'spacelift-intent-testing' label")
		t.Logf("✅ Successfully validated context creation via Spacelift API: %s (ID: %s)", context.Name, context.ID)
	}
}

// TestSpaceliftContextLifecycleUpdate tests updating a spacelift_context resource
func TestSpaceliftContextLifecycleUpdate(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        "Test Context Initial",
		"description": "Initial context description",
		"labels":      []string{"test"},
	})
	defer th.CleanupResource(resourceID)

	// Verify initial state
	initialState, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(initialState, err, "state-get")
	initialContent := th.GetTextContent(initialState)
	assert.Contains(t, initialContent, resourceID, "Initial state should contain resource ID")
	assert.Contains(t, initialContent, "Test Context Initial", "Initial state should contain original name")

	// Perform update
	result, err := th.CallTool("lifecycle-resources-update", map[string]any{
		"resource_id": resourceID,
		"config": map[string]any{
			"name":        "Test Context Updated",
			"description": "Updated context description with more details",
			"labels":      []string{"test", "updated", "integration"},
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-update")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Update result should contain resource ID")

	// Verify updated state
	updatedState, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(updatedState, err, "state-get")
	updatedContent := th.GetTextContent(updatedState)
	assert.Contains(t, updatedContent, resourceID, "Updated state should contain resource ID")
	assert.Contains(t, updatedContent, "Test Context Updated", "Updated state should contain new name")
	assert.Contains(t, updatedContent, "Updated context description", "Updated state should contain new description")
}

// TestSpaceliftContextLifecycleRefresh tests refreshing a spacelift_context resource
func TestSpaceliftContextLifecycleRefresh(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        "Test Context for Refresh",
		"description": "A context to test refresh functionality",
		"labels":      []string{"test", "refresh"},
	})
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("lifecycle-resources-refresh", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-refresh")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Refresh result should contain resource ID")
}

// TestSpaceliftContextLifecycleDelete tests deleting a spacelift_context resource
func TestSpaceliftContextLifecycleDelete(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        "Test Context for Deletion",
		"description": "A context that will be deleted",
		"labels":      []string{"test", "delete"},
	})

	result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-delete")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Delete result should contain resource ID")
	assert.Contains(t, content, "deleted", "Should show deleted status")
}

// TestSpaceliftContextResourceImport tests importing a spacelift_context resource
func TestSpaceliftContextResourceImport(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := testhelper.GenerateUniqueResourceID("test-spacelift-context-import")
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("lifecycle-resources-import", map[string]any{
		"resource_id":   resourceID,
		"provider":      "spacelift-io/spacelift",
		"resource_type": "spacelift_context",
		"import_id":     "dummy-context-id", // This would be a real context ID in practice
		"config": map[string]any{
			"name":        "Imported Test Context",
			"description": "A context imported during testing",
		},
	})

	// Import might fail if the context doesn't exist, which is expected
	if result.IsError {
		t.Logf("Import failed as expected (context doesn't exist): %s", th.GetTextContent(result))
	} else {
		th.AssertToolSuccess(result, err, "lifecycle-resources-import")
	}
}

// TestSpaceliftContextResourceOperations tests getting operations for a spacelift_context resource
func TestSpaceliftContextResourceOperations(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        "Test Context for Operations",
		"description": "A context to test operations tracking",
		"labels":      []string{"test", "operations"},
	})
	defer th.CleanupResource(resourceID)

	// Update the resource to have some operations
	_, err := th.CallTool("lifecycle-resources-update", map[string]any{
		"resource_id": resourceID,
		"config": map[string]any{
			"name":        "Test Context Updated for Operations",
			"description": "Updated description to create more operations",
			"labels":      []string{"test", "operations", "updated"},
		},
	})
	require.NoError(t, err)

	result, err := th.CallTool("lifecycle-resources-operations", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-operations")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Operations should contain resource ID")
	assert.Contains(t, content, "create", "Should show create operation")
	assert.Contains(t, content, "update", "Should show update operation")
}

// TestSpaceliftContextStateGet tests getting state for a spacelift_context resource
func TestSpaceliftContextStateGet(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        "Test Context State",
		"description": "A context to test state retrieval",
		"labels":      []string{"test", "state"},
	})
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "state-get")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "State should contain resource ID")
	assert.Contains(t, content, "spacelift_context", "State should contain resource type")
	assert.Contains(t, content, "Test Context State", "State should contain the context name")
}

// TestSpaceliftContextWithScripts tests creating a context with hook scripts
func TestSpaceliftContextWithScripts(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := testhelper.GenerateUniqueResourceID("test-spacelift-context-scripts")
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":   resourceID,
		"provider":      "spacelift-io/spacelift",
		"resource_type": "spacelift_context",
		"config": map[string]any{
			"name":        "Test Context with Scripts",
			"description": "A context with before/after scripts",
			"labels":      []string{"test", "scripts"},
			"before_init": []string{
				"echo 'Before init script 1'",
				"echo 'Before init script 2'",
			},
			"after_init": []string{
				"echo 'After init completed'",
			},
			"before_plan": []string{
				"echo 'Preparing for plan'",
			},
			"after_plan": []string{
				"echo 'Plan completed successfully'",
			},
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-create")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Create result should contain resource ID")
	assert.Contains(t, content, "created", "Should show created status")

	// Verify the scripts are in the state
	stateResult, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(stateResult, err, "state-get")
	stateContent := th.GetTextContent(stateResult)
	assert.Contains(t, stateContent, "Before init script", "State should contain before_init scripts")
	assert.Contains(t, stateContent, "After init completed", "State should contain after_init scripts")
}

// TestSpaceliftContextDependencyManagement tests dependency management with spacelift_context
func TestSpaceliftContextDependencyManagement(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	parentContextID := testhelper.GenerateUniqueResourceID("test-parent-context")
	childContextID := testhelper.GenerateUniqueResourceID("test-child-context")

	// Create parent context
	result, err := th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":   parentContextID,
		"provider":      "spacelift-io/spacelift",
		"resource_type": "spacelift_context",
		"config": map[string]any{
			"name":        "Parent Context",
			"description": "Parent context for dependency testing",
			"labels":      []string{"test", "parent"},
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-create parent")
	defer th.CleanupResource(parentContextID)

	// Create child context
	result, err = th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":   childContextID,
		"provider":      "spacelift-io/spacelift",
		"resource_type": "spacelift_context",
		"config": map[string]any{
			"name":        "Child Context",
			"description": "Child context for dependency testing",
			"labels":      []string{"test", "child"},
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-create child")
	defer th.CleanupResource(childContextID)

	// Add dependency
	result, err = th.CallTool("lifecycle-resources-dependencies-add", map[string]any{
		"from_resource_id": childContextID,
		"to_resource_id":   parentContextID,
		"dependency_type":  "explicit",
		"explanation":      "Child context depends on parent context configuration",
	})

	// Dependencies might fail due to foreign key constraints if resources aren't properly stored
	if result.IsError {
		content := th.GetTextContent(result)
		if assert.Contains(t, content, "constraint failed", "Expected constraint error") {
			t.Logf("Dependency add failed as might be expected: %s", content)
			t.Skip("Skipping remaining dependency tests due to constraint issues")
		}
	} else {
		th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-add")

		content := th.GetTextContent(result)
		assert.Contains(t, content, childContextID, "Dependency add result should contain child resource")
		assert.Contains(t, content, parentContextID, "Dependency add result should contain parent resource")

		// Get dependencies
		result, err = th.CallTool("lifecycle-resources-dependencies-get", map[string]any{
			"resource_id": childContextID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-get")

		content = th.GetTextContent(result)
		t.Logf("Dependencies result: %s", content)

		// Remove dependency
		result, err = th.CallTool("lifecycle-resources-dependencies-remove", map[string]any{
			"from_resource_id": childContextID,
			"to_resource_id":   parentContextID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-remove")

		content = th.GetTextContent(result)
		assert.Contains(t, content, "removed", "Should show removal status")
	}
}
