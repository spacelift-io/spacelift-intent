package provider_test

import (
	"encoding/json"
	"testing"

	"github.com/spacelift-io/spacelift-intent/test/testhelper"
	"github.com/spacelift-io/spacelift-intent/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateResource_TopLevelConfigMerge_E2E verifies that UpdateResource correctly
// implements top-level config merging (not recursive merging) by testing keeper
// add/remove/change operations on a random_uuid resource.
//
// This test validates the refactoring in PR #46 that simplified config merge logic
// to only merge at the top level, enabling proper removal of nested items like
// map keys (e.g., removing AWS resource tags or random_uuid keepers).
func TestUpdateResource_TopLevelConfigMerge_E2E(t *testing.T) {
	// Setup test helper with shared provider cache
	th := testhelper.NewTestHelper(t)
	defer th.Cleanup()

	resourceID := "test-uuid-merge"
	provider := "hashicorp/random"
	providerVersion := "3.7.2"
	resourceType := "random_uuid"

	// Ensure cleanup happens even if test fails
	defer func() {
		th.CleanupResource(resourceID)
	}()

	// Phase 1: Create resource with 3 keepers
	t.Run("create with 3 keepers", func(t *testing.T) {
		createResult, err := th.CallTool("lifecycle-resources-create", map[string]any{
			"resource_id":      resourceID,
			"provider":         provider,
			"provider_version": providerVersion,
			"resource_type":    resourceType,
			"config": map[string]any{
				"keepers": map[string]any{
					"environment": "production",
					"application": "web-app",
					"version":     "v1.0.0",
				},
			},
		})
		th.AssertToolSuccess(createResult, err, "lifecycle-resources-create")

		// Verify initial state - check ALL fields
		stateResult, err := th.CallTool("state-get", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(stateResult, err, "state-get")

		var stateRecord types.StateRecord
		stateContent := th.GetTextContent(stateResult)
		err = json.Unmarshal([]byte(stateContent), &stateRecord)
		require.NoError(t, err, "should parse state JSON")

		// Assert metadata fields
		assert.Equal(t, resourceID, stateRecord.ResourceID)
		assert.Equal(t, provider, stateRecord.Provider)
		assert.Equal(t, providerVersion, stateRecord.ProviderVersion)
		assert.Equal(t, resourceType, stateRecord.ResourceType)
		assert.NotEmpty(t, stateRecord.CreatedAt, "created_at should be set")

		// Assert state fields
		require.NotNil(t, stateRecord.State, "state should not be nil")
		keepers, ok := stateRecord.State["keepers"].(map[string]any)
		require.True(t, ok, "keepers should be a map")
		assert.Len(t, keepers, 3, "should have exactly 3 keepers")
		assert.Equal(t, "production", keepers["environment"])
		assert.Equal(t, "web-app", keepers["application"])
		assert.Equal(t, "v1.0.0", keepers["version"])

		// Verify UUID fields are present
		assert.NotEmpty(t, stateRecord.State["id"], "id should be set")
		assert.NotEmpty(t, stateRecord.State["result"], "result should be set")
	})

	// Phase 2: Update - remove 1 keeper, add 1 keeper, change 1 keeper
	// This is the critical test: verifies top-level merge (not recursive)
	t.Run("update with keeper changes", func(t *testing.T) {
		updateResult, err := th.CallTool("lifecycle-resources-update", map[string]any{
			"resource_id": resourceID,
			"config": map[string]any{
				"keepers": map[string]any{
					"environment": "production", // unchanged
					"version":     "v2.0.0",     // changed
					"region":      "us-east-1",  // added
					// "application" removed - this tests top-level merge!
				},
			},
		})
		th.AssertToolSuccess(updateResult, err, "lifecycle-resources-update")

		// Verify updated state - critical assertions for top-level merge
		stateResult, err := th.CallTool("state-get", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(stateResult, err, "state-get")

		var stateRecord types.StateRecord
		stateContent := th.GetTextContent(stateResult)
		err = json.Unmarshal([]byte(stateContent), &stateRecord)
		require.NoError(t, err, "should parse state JSON")

		// Assert metadata fields remain consistent
		assert.Equal(t, resourceID, stateRecord.ResourceID)
		assert.Equal(t, provider, stateRecord.Provider)
		assert.Equal(t, providerVersion, stateRecord.ProviderVersion)
		assert.Equal(t, resourceType, stateRecord.ResourceType)

		// Assert state fields - verify top-level merge behavior
		keepers, ok := stateRecord.State["keepers"].(map[string]any)
		require.True(t, ok, "keepers should be a map")
		assert.Len(t, keepers, 3, "should have exactly 3 keepers after update")

		// Verify expected keys exist with correct values
		assert.Equal(t, "production", keepers["environment"], "unchanged keeper should remain")
		assert.Equal(t, "v2.0.0", keepers["version"], "changed keeper should have new value")
		assert.Equal(t, "us-east-1", keepers["region"], "new keeper should be added")

		// CRITICAL ASSERTION: Verify removed key is gone (proves top-level merge)
		assert.NotContains(t, keepers, "application",
			"removed keeper should not exist - this proves top-level merge, not recursive")

		// Verify UUID fields still present (may have changed due to keeper change)
		assert.NotEmpty(t, stateRecord.State["id"])
		assert.NotEmpty(t, stateRecord.State["result"])
	})

	// Phase 3: Update to single keeper
	// Further validates top-level merge by removing all but one keeper
	t.Run("update to single keeper", func(t *testing.T) {
		updateResult, err := th.CallTool("lifecycle-resources-update", map[string]any{
			"resource_id": resourceID,
			"config": map[string]any{
				"keepers": map[string]any{
					"environment": "staging", // only one keeper, different value
					// All other keepers removed
				},
			},
		})
		th.AssertToolSuccess(updateResult, err, "lifecycle-resources-update")

		// Verify final state - all other keepers should be removed
		stateResult, err := th.CallTool("state-get", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(stateResult, err, "state-get")

		var stateRecord types.StateRecord
		stateContent := th.GetTextContent(stateResult)
		err = json.Unmarshal([]byte(stateContent), &stateRecord)
		require.NoError(t, err, "should parse state JSON")

		// Assert metadata fields remain consistent
		assert.Equal(t, resourceID, stateRecord.ResourceID)
		assert.Equal(t, provider, stateRecord.Provider)
		assert.Equal(t, providerVersion, stateRecord.ProviderVersion)
		assert.Equal(t, resourceType, stateRecord.ResourceType)

		// Assert state fields - verify complete replacement
		keepers, ok := stateRecord.State["keepers"].(map[string]any)
		require.True(t, ok, "keepers should be a map")
		assert.Len(t, keepers, 1, "should have exactly 1 keeper after final update")
		assert.Equal(t, "staging", keepers["environment"], "keeper should have updated value")

		// Verify all other keepers are gone
		assert.NotContains(t, keepers, "version", "version keeper should be removed")
		assert.NotContains(t, keepers, "region", "region keeper should be removed")
		assert.NotContains(t, keepers, "application", "application keeper should still be gone")

		// Verify UUID fields still present
		assert.NotEmpty(t, stateRecord.State["id"])
		assert.NotEmpty(t, stateRecord.State["result"])
	})
}
