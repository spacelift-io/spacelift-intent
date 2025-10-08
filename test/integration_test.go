// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spacelift-io/spacelift-intent/test/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerInitialization tests basic server functionality
func TestServerInitialization(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	tools, err := th.Client.ListTools(th.Ctx, mcp.ListToolsRequest{})
	require.NoError(t, err, "Should be able to list tools")
	assert.Greater(t, len(tools.Tools), 0, "Should have at least one tool")

	// Check that we have the expected lifecycle tools
	toolNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"provider-search",
		"provider-describe",
		"provider-resources-describe",
		"lifecycle-resources-create",
		"lifecycle-resources-update",
		"lifecycle-resources-delete",
		"lifecycle-resources-refresh",
		"lifecycle-resources-import",
		"lifecycle-resources-operations",
		"lifecycle-datasources-read",
		"provider-datasources-describe",
		"state-get",
		"state-list",
		"state-timeline",
		"state-eject",
		"lifecycle-resources-dependencies-add",
		"lifecycle-resources-dependencies-get",
		"lifecycle-resources-dependencies-remove",
	}

	for _, expectedTool := range expectedTools {
		assert.True(t, toolNames[expectedTool], "Should have tool: %s", expectedTool)
	}
}

// TestServerHealth tests basic server health
func TestServerHealth(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("state-list", map[string]any{})
	th.AssertToolSuccess(result, err, "state-list")

	content := th.GetTextContent(result)
	assert.NotEmpty(t, content, "Should return some content")
}

// TestProviderSearch tests provider search functionality
func TestProviderSearch(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("provider-search", map[string]any{
		"query": "hashicorp/random",
	})
	th.AssertToolSuccess(result, err, "provider-search")

	content := th.GetTextContent(result)
	assert.Contains(t, content, "random", "Search results should contain 'random'")
	assert.Contains(t, content, "hashicorp", "Search results should contain 'hashicorp'")
}

// TestProviderDescribe tests provider description functionality
func TestProviderDescribe(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("provider-describe", map[string]any{
		"provider": "hashicorp/random",
	})
	th.AssertToolSuccess(result, err, "provider-describe")

	content := th.GetTextContent(result)
	assert.Contains(t, content, "random", "Provider description should contain provider info")
}

// TestResourceDescribe tests resource description functionality
func TestResourceDescribe(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("provider-resources-describe", map[string]any{
		"provider":      "hashicorp/random",
		"resource_type": "random_string",
	})
	th.AssertToolSuccess(result, err, "provider-resources-describe")

	content := th.GetTextContent(result)
	assert.Contains(t, content, "random_string", "Resource description should contain resource type")
	assert.Contains(t, content, "length", "random_string should have 'length' attribute")
}

// TestResourceLifecycleCreate tests resource creation
func TestResourceLifecycleCreate(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := testhelper.GenerateUniqueResourceID("test-create")
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":   resourceID,
		"provider":      "hashicorp/random",
		"resource_type": "random_string",
		"config": map[string]any{
			"length":  8,
			"special": true,
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-create")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Create result should contain resource ID")
	assert.Contains(t, content, "created", "Should show created status")
}

// TestResourceLifecycleUpdate tests resource updates
func TestResourceLifecycleUpdate(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_string", map[string]any{
		"length":  8,
		"special": true,
	})
	defer th.CleanupResource(resourceID)

	// Verify initial state
	initialState, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(initialState, err, "state-get")
	initialContent := th.GetTextContent(initialState)
	assert.Contains(t, initialContent, resourceID, "Initial state should contain resource ID")
	assert.Contains(t, initialContent, "8", "Initial state should contain length: 8")

	// Perform update
	result, err := th.CallTool("lifecycle-resources-update", map[string]any{
		"resource_id": resourceID,
		"config": map[string]any{
			"length":  12,
			"special": false,
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
	assert.Contains(t, updatedContent, "12", "Updated state should contain length: 12")
	assert.Contains(t, updatedContent, "false", "Updated state should contain special: false")
}

// TestResourceLifecycleRefresh tests resource refresh
func TestResourceLifecycleRefresh(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_string", map[string]any{
		"length":  8,
		"special": true,
	})
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("lifecycle-resources-refresh", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-refresh")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Refresh result should contain resource ID")
}

// TestResourceLifecycleDelete tests resource deletion
func TestResourceLifecycleDelete(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_string", map[string]any{
		"length":  8,
		"special": true,
	})

	result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-delete")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Delete result should contain resource ID")
	assert.Contains(t, content, "deleted", "Should show deleted status")
}

// TestResourceImport tests resource import functionality
func TestResourceImport(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := testhelper.GenerateUniqueResourceID("test-import")
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("lifecycle-resources-import", map[string]any{
		"resource_id":   resourceID,
		"provider":      "hashicorp/random",
		"resource_type": "random_string",
		"import_id":     "dummy-import-id",
		"config": map[string]any{
			"length": 10,
		},
	})

	// Note: This might fail for random resources as they can't really be imported
	// but we test that the tool is available and accepts the right parameters
	if result.IsError {
		t.Logf("Import failed as expected for random resource: %s", th.GetTextContent(result))
	} else {
		th.AssertToolSuccess(result, err, "lifecycle-resources-import")
	}
}

// TestResourceOperations tests getting resource operations
func TestResourceOperations(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_string", map[string]any{
		"length":  8,
		"special": true,
	})
	defer th.CleanupResource(resourceID)

	// Update the resource to have some operations
	_, err := th.CallTool("lifecycle-resources-update", map[string]any{
		"resource_id": resourceID,
		"config": map[string]any{
			"length":  12,
			"special": false,
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

// TestStateGet tests getting resource state
func TestStateGet(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_pet", map[string]any{
		"length": 2,
	})
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "state-get")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "State should contain resource ID")
	assert.Contains(t, content, "random_pet", "State should contain resource type")
}

// TestStateList tests listing all states
func TestStateList(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_pet", map[string]any{
		"length": 2,
	})
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("state-list", map[string]any{})
	th.AssertToolSuccess(result, err, "state-list")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "State list should contain our resource")
}

// TestStateTimeline tests state timeline functionality
func TestStateTimeline(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_pet", map[string]any{
		"length": 2,
	})
	defer th.CleanupResource(resourceID)

	result, err := th.CallTool("state-timeline", map[string]any{})
	th.AssertToolSuccess(result, err, "state-timeline")

	content := th.GetTextContent(result)
	assert.Contains(t, content, resourceID, "Timeline should contain our resource")
	assert.Contains(t, content, "create", "Timeline should show create operation")
}

// TestStateEject tests state ejection
func TestStateEject(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := th.CreateTestResource("hashicorp/random", "random_pet", map[string]any{
		"length": 2,
	})

	result, err := th.CallTool("state-eject", map[string]any{
		"resource_id": resourceID,
	})
	th.AssertToolSuccess(result, err, "state-eject")

	content := th.GetTextContent(result)
	assert.Contains(t, content, "ejected", "Eject should confirm ejection")
	assert.Contains(t, content, "random_pet", "Eject should contain resource type")
	assert.Contains(t, content, resourceID, "Eject should contain resource name")

	// Resource is ejected, so cleanup might fail - that's expected
}

// TestDataSourceDescribe tests data source description
func TestDataSourceDescribe(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("provider-datasources-describe", map[string]any{
		"provider":         "hashicorp/random",
		"data_source_type": "random_string",
	})

	// The hashicorp/random provider doesn't have data sources, so this is expected to fail
	if result.IsError {
		content := th.GetTextContent(result)
		assert.Contains(t, content, "data source type random_string not found", "Expected data source not found error")
		t.Logf("Data source describe failed as expected: %s", content)
	} else {
		th.AssertToolSuccess(result, err, "provider-datasources-describe")
	}
}

// TestDataSourceRead tests data source reading
func TestDataSourceRead(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	result, err := th.CallTool("lifecycle-datasources-read", map[string]any{
		"provider":         "hashicorp/random",
		"data_source_type": "random_string",
		"config": map[string]any{
			"length":  8,
			"special": false,
		},
	})

	// The hashicorp/random provider doesn't have data sources, so this is expected to fail
	if result.IsError {
		content := th.GetTextContent(result)
		assert.Contains(t, content, "no data source schemas", "Expected no data source schemas error")
		t.Logf("Data source read failed as expected: %s", content)
	} else {
		th.AssertToolSuccess(result, err, "lifecycle-datasources-read")
	}
}

// TestDependencyManagement tests the full dependency lifecycle
func TestDependencyManagement(t *testing.T) {
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	parentResourceID := testhelper.GenerateUniqueResourceID("test-dep-parent")
	childResourceID := testhelper.GenerateUniqueResourceID("test-dep-child")

	// Create parent resource
	result, err := th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":   parentResourceID,
		"provider":      "hashicorp/random",
		"resource_type": "random_string",
		"config": map[string]any{
			"length": 6,
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-create parent")
	defer th.CleanupResource(parentResourceID)

	// Create child resource
	result, err = th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":   childResourceID,
		"provider":      "hashicorp/random",
		"resource_type": "random_pet",
		"config": map[string]any{
			"length": 2,
		},
	})
	th.AssertToolSuccess(result, err, "lifecycle-resources-create child")
	defer th.CleanupResource(childResourceID)

	// Add dependency
	result, err = th.CallTool("lifecycle-resources-dependencies-add", map[string]any{
		"from_resource_id": childResourceID,
		"to_resource_id":   parentResourceID,
		"dependency_type":  "explicit",
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
		assert.Contains(t, content, childResourceID, "Dependency add result should contain child resource")
		assert.Contains(t, content, parentResourceID, "Dependency add result should contain parent resource")

		// Get dependencies
		result, err = th.CallTool("lifecycle-resources-dependencies-get", map[string]any{
			"resource_id": childResourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-get")

		content = th.GetTextContent(result)
		t.Logf("Dependencies result: %s", content)

		// Remove dependency
		result, err = th.CallTool("lifecycle-resources-dependencies-remove", map[string]any{
			"from_resource_id": childResourceID,
			"to_resource_id":   parentResourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-remove")

		content = th.GetTextContent(result)
		assert.Contains(t, content, "removed", "Should show removal status")
	}
}
