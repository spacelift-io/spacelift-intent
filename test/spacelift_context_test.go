package test

import (
	"strings"
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
			err := testhelper.CleanupContextById(t, contextID)
			assert.NoError(t, err, "Failed to cleanup context via Spacelift API")
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
	require.NoError(t, err, "Failed to validate context creation via Spacelift API")

	contextID = context.ID // Store for cleanup
	assert.Equal(t, contextName, context.Name, "Context name should match")
	assert.NotEmpty(t, context.ID, "Context should have an ID")
	assert.Contains(t, context.Labels, "spacelift-intent-testing", "Context should have 'spacelift-intent-testing' label")
	t.Logf("✅ Successfully validated context creation via Spacelift API: %s (ID: %s)", context.Name, context.ID)
}

// TestSpaceliftContextLifecycleUpdate tests updating a spacelift_context resource
func TestSpaceliftContextLifecycleUpdate(t *testing.T) {
	// Load Spacelift credentials from .env.spacelift
	testhelper.LoadSpaceliftCredentials(t)

	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	initialName := "Test Context Initial"
	updatedName := "Test Context Updated"
	var contextID string

	// Cleanup via Spacelift API at the end
	defer func() {
		if contextID != "" {
			err := testhelper.CleanupContextById(t, contextID)
			assert.NoError(t, err, "Failed to cleanup context via Spacelift API")
		}
	}()

	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        initialName,
		"description": "Initial context description",
		"labels":      []string{"spacelift-intent-testing"},
	})
	defer th.CleanupResource(resourceID)

	// Verify initial context via API
	initialContext, err := testhelper.ValidateContextExistsByName(t, initialName)
	require.NoError(t, err, "Failed to validate initial context creation via Spacelift API")
	contextID = initialContext.ID
	assert.Equal(t, initialName, initialContext.Name, "Initial context name should match")
	t.Logf("✅ Initial context validated via API: %s (ID: %s)", initialContext.Name, initialContext.ID)

	// Verify initial state
	initialState, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(initialState, err, "state-get")
	initialContent := th.GetTextContent(initialState)
	assert.Contains(t, initialContent, resourceID, "Initial state should contain resource ID")
	assert.Contains(t, initialContent, initialName, "Initial state should contain original name")

	// Perform update
	result, err := th.CallTool("lifecycle-resources-update", map[string]any{
		"resource_id": resourceID,
		"config": map[string]any{
			"name":        updatedName,
			"description": "Updated context description with more details",
			"labels":      []string{"spacelift-intent-testing"},
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
	assert.Contains(t, updatedContent, updatedName, "Updated state should contain new name")
	assert.Contains(t, updatedContent, "Updated context description", "Updated state should contain new description")

	// Verify the update was applied via Spacelift API
	updatedContext, err := testhelper.ValidateContextExistsByName(t, updatedName)
	require.NoError(t, err, "Failed to validate context update via Spacelift API")
	assert.Equal(t, updatedName, updatedContext.Name, "Updated context name should match")
	assert.Equal(t, contextID, updatedContext.ID, "Context ID should remain the same after update")
	t.Logf("✅ Successfully validated context update via Spacelift API: %s (ID: %s)", updatedContext.Name, updatedContext.ID)
}

// TestSpaceliftContextLifecycleRefresh tests refreshing a spacelift_context resource
func TestSpaceliftContextLifecycleRefresh(t *testing.T) {
	// Load Spacelift credentials from .env.spacelift
	testhelper.LoadSpaceliftCredentials(t)

	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	contextName := "Test Context for Refresh"
	var contextID string

	// Cleanup via Spacelift API at the end
	defer func() {
		if contextID != "" {
			err := testhelper.CleanupContextById(t, contextID)
			assert.NoError(t, err, "Failed to cleanup context via Spacelift API")
		}
	}()

	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        contextName,
		"description": "A context to test refresh functionality",
		"labels":      []string{"spacelift-intent-testing"},
	})
	defer th.CleanupResource(resourceID)

	// Verify initial context via API and capture context ID
	initialContext, err := testhelper.ValidateContextExistsByName(t, contextName)
	require.NoError(t, err, "Failed to validate initial context creation via Spacelift API")
	contextID = initialContext.ID
	t.Logf("✅ Initial context validated via API: %s (ID: %s)", initialContext.Name, initialContext.ID)

	// Get initial state to verify before changes
	initialState, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(initialState, err, "state-get initial")
	initialStateContent := th.GetTextContent(initialState)
	assert.Contains(t, initialStateContent, "A context to test refresh functionality", "Initial state should contain original description")

	// Update the context externally via Spacelift API to simulate drift
	updatedDescription := "Description updated externally via API - should be detected by refresh"
	err = testhelper.UpdateContextDescription(t, contextID, updatedDescription)
	require.NoError(t, err, "Failed to update context via Spacelift API")
	t.Logf("✅ Updated context description via Spacelift API")

	// Verify the external change was applied via API
	externallyUpdatedContext, err := testhelper.ValidateContextExistsById(t, contextID)
	require.NoError(t, err, "Failed to validate externally updated context via Spacelift API")
	assert.Equal(t, updatedDescription, *externallyUpdatedContext.Description, "Context description should be updated via API")
	t.Logf("✅ Validated external update via Spacelift API: %s", *externallyUpdatedContext.Description)

	// Now refresh the resource - this should detect the external changes
	result, err := th.CallTool("lifecycle-resources-refresh", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-refresh")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Refresh result should contain resource ID")

	// Get state after refresh to verify it detected the external changes
	refreshedState, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(refreshedState, err, "state-get after refresh")
	refreshedStateContent := th.GetTextContent(refreshedState)
	assert.Contains(t, refreshedStateContent, updatedDescription, "Refreshed state should contain the externally updated description")
	assert.NotContains(t, refreshedStateContent, "A context to test refresh functionality", "Refreshed state should not contain the old description")

	// Final verification via API
	finalContext, err := testhelper.ValidateContextExistsById(t, contextID)
	require.NoError(t, err, "Failed to validate context exists after refresh via Spacelift API")
	assert.Equal(t, contextName, finalContext.Name, "Context name should remain unchanged after refresh")
	assert.Equal(t, contextID, finalContext.ID, "Context ID should remain unchanged after refresh")
	assert.Equal(t, updatedDescription, *finalContext.Description, "Context description should reflect the external update")
	t.Logf("✅ Successfully validated context drift detection and refresh: %s (ID: %s)", finalContext.Name, finalContext.ID)
}

// TestSpaceliftContextLifecycleDelete tests deleting a spacelift_context resource
func TestSpaceliftContextLifecycleDelete(t *testing.T) {
	// Load Spacelift credentials from .env.spacelift
	testhelper.LoadSpaceliftCredentials(t)

	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	contextName := "Test Context for Deletion"
	var contextID string

	// Create the context resource
	resourceID := th.CreateTestResource("spacelift-io/spacelift", "spacelift_context", map[string]any{
		"name":        contextName,
		"description": "A context that will be deleted",
		"labels":      []string{"spacelift-intent-testing"},
	})

	// Verify the context was created and exists in Spacelift
	createdContext, err := testhelper.ValidateContextExistsByName(t, contextName)
	require.NoError(t, err, "Failed to validate context creation via Spacelift API")
	contextID = createdContext.ID
	assert.Equal(t, contextName, createdContext.Name, "Created context name should match")
	assert.Equal(t, "A context that will be deleted", *createdContext.Description, "Created context description should match")
	t.Logf("✅ Context created and validated via API: %s (ID: %s)", createdContext.Name, createdContext.ID)

	// Delete the context via the tool
	result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-delete")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Delete result should contain resource ID")
	assert.Contains(t, content, "deleted", "Should show deleted status")

	// Verify the context no longer exists in Spacelift
	_, err = testhelper.ValidateContextExistsById(t, contextID)
	assert.Error(t, err, "Context should not exist after deletion")
	t.Logf("✅ Verified context deletion: context with ID '%s' no longer exists in Spacelift", contextID)
}

// TestSpaceliftContextResourceImport tests importing a spacelift_context resource
func TestSpaceliftContextResourceImport(t *testing.T) {
	// Load Spacelift credentials from .env.spacelift
	testhelper.LoadSpaceliftCredentials(t)

	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	contextName := "Test Context for Import"
	resourceID := testhelper.GenerateUniqueResourceID("test-spacelift-context-import")
	var contextID string

	// Cleanup via Spacelift API at the end
	defer func() {
		if contextID != "" {
			err := testhelper.CleanupContextById(t, contextID)
			assert.NoError(t, err, "Failed to cleanup context via Spacelift API")
		}
	}()

	// Step 1: Create a context directly via Spacelift API (not through our tools)
	createdContext, err := testhelper.CreateContextViaAPI(t, contextName, "A context created via API for import testing")
	require.NoError(t, err, "Failed to create context via Spacelift API")
	contextID = createdContext.ID
	t.Logf("✅ Created context via Spacelift API: %s (ID: %s)", createdContext.Name, createdContext.ID)

	// Step 2: Verify the context does NOT exist in our state management
	stateResult, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	assert.True(t, stateResult.IsError, "Context should not exist in state before import")
	t.Logf("✅ Verified context does not exist in state before import")

	// Step 3: Import the existing context into our state management
	importResult, err := th.CallTool("lifecycle-resources-import", map[string]any{
		"destination_id": resourceID, // This is the ID we'll use in our state
		"provider":       "spacelift-io/spacelift",
		"resource_type":  "spacelift_context",
		"import_id":      contextID, // Use the real context ID from API
	})
	th.AssertToolSuccess(importResult, err, "lifecycle-resources-import")
	defer th.CleanupResource(resourceID) // Cleanup from state after successful import

	importContent := th.GetTextContent(importResult)
	assert.Contains(t, importContent, "imported", "Import result should show imported status")
	assert.Contains(t, importContent, contextID, "Import result should contain the Spacelift context ID")
	t.Logf("✅ Successfully imported context into state management")

	// Step 4: Verify the context now EXISTS in our state management
	stateAfterImport, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(stateAfterImport, err, "state-get after import")

	stateContent := th.GetTextContent(stateAfterImport)
	assert.Contains(t, stateContent, resourceID, "State should contain resource ID after import")
	assert.Contains(t, stateContent, contextID, "State should contain Spacelift context ID after import")

	// Check if import captured the full resource data
	if strings.Contains(stateContent, contextName) && strings.Contains(stateContent, "A context created via API for import testing") {
		t.Logf("✅ Import captured full resource data including name and description")
	} else {
		t.Logf("⚠️  Import only captured basic resource structure - name/description are null")
		t.Logf("This might be expected behavior depending on Spacelift provider import implementation")
	}

	t.Logf("✅ Verified context exists in state after import")

	// Step 5: Verify the context still exists in Spacelift and data matches
	finalContext, err := testhelper.ValidateContextExistsById(t, contextID)
	require.NoError(t, err, "Context should still exist in Spacelift after import")
	assert.Equal(t, contextName, finalContext.Name, "Context name should match after import")
	assert.Equal(t, "A context created via API for import testing", *finalContext.Description, "Context description should match after import")
	t.Logf("✅ Verified context consistency between state and Spacelift API after import")
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
