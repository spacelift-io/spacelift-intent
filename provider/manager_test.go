// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/types"
)

func TestPlanResource(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	// Test PlanResource function with random provider
	config := map[string]any{
		"length":  16,
		"special": true,
	}

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	plannedState, err := adapter.PlanResource(ctx, providerConfig, "random_password", nil, config)
	require.NoError(t, err)

	// Basic validations
	assert.NotNil(t, plannedState)
	assert.EqualValues(t, 16, plannedState["length"])
	assert.Equal(t, true, plannedState["special"])

	t.Logf("Planned random_password resource: %+v", plannedState)
}

func TestCreateResource(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	// Test CreateResource function with random provider
	config := map[string]any{
		"length":  8,
		"special": false,
	}

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	createdState, err := adapter.CreateResource(ctx, providerConfig, "random_string", config)
	require.NoError(t, err)

	// Basic validations
	assert.NotNil(t, createdState)
	assert.EqualValues(t, 8, createdState["length"])
	assert.Equal(t, false, createdState["special"])
	assert.NotEmpty(t, createdState["result"]) // Should have generated a string
	assert.NotEmpty(t, createdState["id"])     // Should have generated an ID

	t.Logf("Created random_string resource: %+v", createdState)
}

func TestUpdateResource(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	// First create a resource to have current state
	initialConfig := map[string]any{
		"length":  6,
		"special": false,
	}

	currentState, err := adapter.CreateResource(ctx, providerConfig, "random_string", initialConfig)
	require.NoError(t, err)
	require.NotEmpty(t, currentState["result"])

	// Now update it with new config
	newConfig := map[string]any{
		"length":  10,
		"special": true,
	}

	updatedState, err := adapter.UpdateResource(ctx, providerConfig, "random_string", currentState, newConfig)
	require.NoError(t, err)

	// Basic validations
	assert.NotNil(t, updatedState)
	assert.EqualValues(t, 10, updatedState["length"])
	assert.Equal(t, true, updatedState["special"])
	assert.NotEmpty(t, updatedState["result"]) // Should have generated a new string
	assert.NotEmpty(t, updatedState["id"])

	// The result might be the same since random_string doesn't regenerate without keepers
	// But the configuration should be updated
	assert.NotEqual(t, currentState["length"], updatedState["length"])
	assert.NotEqual(t, currentState["special"], updatedState["special"])

	t.Logf("Updated random_string resource: %+v", updatedState)
}

func TestDeleteResource(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	// First create a resource to delete
	config := map[string]any{
		"length":  8,
		"special": false,
	}

	state, err := adapter.CreateResource(ctx, providerConfig, "random_string", config)
	require.NoError(t, err)
	require.NotEmpty(t, state["result"])

	// Now delete the resource
	// TODO(michal): why do we need state to delete the resource?
	err = adapter.DeleteResource(ctx, providerConfig, "random_string", state)
	require.NoError(t, err)

	t.Logf("Successfully deleted random_string resource with id: %v", state["id"])
}

func TestRefreshResource(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	// First create a resource to refresh
	config := map[string]any{
		"length":  12,
		"special": true,
	}

	state, err := adapter.CreateResource(ctx, providerConfig, "random_string", config)
	require.NoError(t, err)
	require.NotEmpty(t, state["result"])

	// Now refresh the resource
	refreshedState, err := adapter.RefreshResource(ctx, providerConfig, "random_string", state)
	require.NoError(t, err)

	// Basic validations
	assert.NotNil(t, refreshedState)
	assert.Equal(t, state["id"], refreshedState["id"])         // ID should remain the same
	assert.Equal(t, state["result"], refreshedState["result"]) // Result should remain the same
	assert.EqualValues(t, 12, refreshedState["length"])
	assert.Equal(t, true, refreshedState["special"])

	t.Logf("Refreshed random_string resource: %+v", refreshedState)
}

func TestListResources(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	// Test ListResources with random provider
	resources, err := adapter.ListResources(ctx, providerConfig)
	require.NoError(t, err)

	// Basic validations
	assert.NotEmpty(t, resources)
	assert.Contains(t, resources, "random_string")
	assert.Contains(t, resources, "random_password")
	assert.Contains(t, resources, "random_id")

	t.Logf("Random provider has %d resources: %v", len(resources), resources)
}

func TestDescribeResource(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	// Test DescribeResource with random_string
	description, err := adapter.DescribeResource(ctx, providerConfig, "random_string")
	require.NoError(t, err)

	// Debug: dump the actual structure to understand what we're getting
	if len(description.Properties) > 0 {
		lengthProp, exists := description.Properties["length"]
		if exists {
			t.Logf("Length property structure: %+v", lengthProp)
		}
	}

	// Basic validations
	assert.NotNil(t, description)
	assert.Equal(t, "hashicorp/random", description.ProviderName)
	assert.Equal(t, "random_string", description.Type)
	assert.NotEmpty(t, description.Description)
	assert.NotNil(t, description.Properties)
	assert.NotEmpty(t, description.Properties)

	// Verify we have the expected number of properties (14 based on the dump output)
	assert.Equal(t, 14, len(description.Properties))

	// Check for all expected properties from the schema dump
	expectedProps := []string{
		"id", "length", "lower", "min_numeric", "min_special", "min_upper",
		"number", "numeric", "keepers", "min_lower", "override_special",
		"result", "special", "upper",
	}
	for _, prop := range expectedProps {
		assert.Contains(t, description.Properties, prop, "Missing property: %s", prop)
	}

	// Verify required fields - only "length" should be required
	assert.Equal(t, 1, len(description.Required), "Should have exactly 1 required field")
	assert.Contains(t, description.Required, "length", "Length should be required")

	// Verify property types and descriptions for key attributes
	lengthProp := description.Properties["length"].(map[string]any)
	assert.Equal(t, "number", lengthProp["type"])
	assert.Contains(t, lengthProp["description"], "length of the string")
	// Check for the metadata fields my implementation should add
	if required, exists := lengthProp["required"]; exists {
		assert.Equal(t, true, required)
	}
	if usage, exists := lengthProp["usage"]; exists {
		assert.Equal(t, "required", usage)
	}

	specialProp := description.Properties["special"].(map[string]any)
	assert.Equal(t, "boolean", specialProp["type"])
	assert.Contains(t, specialProp["description"], "special characters")
	if required, exists := specialProp["required"]; exists {
		assert.Equal(t, false, required)
	}

	resultProp := description.Properties["result"].(map[string]any)
	assert.Equal(t, "string", resultProp["type"])
	assert.Contains(t, resultProp["description"], "generated random string")
	if usage, exists := resultProp["usage"]; exists {
		assert.Equal(t, "computed", usage)
	}

	idProp := description.Properties["id"].(map[string]any)
	assert.Equal(t, "string", idProp["type"])
	assert.Contains(t, idProp["description"], "generated random string")

	numericProp := description.Properties["numeric"].(map[string]any)
	assert.Equal(t, "boolean", numericProp["type"])
	assert.Contains(t, numericProp["description"], "numeric characters")

	// Verify the keepers property (should be object type)
	keepersProp := description.Properties["keepers"].(map[string]any)
	assert.Equal(t, "map", keepersProp["type"])
	assert.Contains(t, keepersProp["description"], "Arbitrary map of values")

	// Verify numeric properties
	minNumericProp := description.Properties["min_numeric"].(map[string]any)
	assert.Equal(t, "number", minNumericProp["type"])

	// Verify the deprecated "number" field exists
	numberProp := description.Properties["number"].(map[string]any)
	assert.Equal(t, "boolean", numberProp["type"])
	assert.Contains(t, numberProp["description"], "deprecated")

	// Verify description contains provider info
	assert.Contains(t, description.Description, "random permutation")
	assert.Contains(t, description.Description, "cryptographic random number generator")

	t.Logf("✓ random_string resource validation complete:")
	t.Logf("  - Properties: %d", len(description.Properties))
	t.Logf("  - Required fields: %d (%v)", len(description.Required), description.Required)
	t.Logf("  - Description length: %d chars", len(description.Description))
}

func TestImportResource(t *testing.T) {
	tmpDir := "./test-providers"
	err := os.MkdirAll(tmpDir, 0755)
	require.NoError(t, err)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	// First create a resource to get an ID we can import
	config := map[string]any{
		"length":  8,
		"special": false,
	}

	providerConfig := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	state, err := adapter.CreateResource(ctx, providerConfig, "random_string", config)
	require.NoError(t, err)
	require.NotEmpty(t, state["id"])

	resourceID := state["id"].(string)

	// Now import the resource using its ID
	importedState, err := adapter.ImportResource(ctx, providerConfig, "random_string", resourceID)
	require.NoError(t, err)

	// Basic validations
	assert.NotNil(t, importedState)
	assert.Equal(t, resourceID, importedState["id"])
	assert.Equal(t, state["result"], importedState["result"]) // Should be the same random string
	assert.EqualValues(t, 8, importedState["length"])

	t.Logf("Imported random_string resource: %+v", importedState)
}

func TestMultipleVersionsInCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	registryClient := registry.NewOpenTofuClient()
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	// Load version 3.6.0 of random provider
	providerConfigV360 := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.6.0",
	}

	err = adapter.LoadProvider(ctx, providerConfigV360)
	require.NoError(t, err)

	// Load version 3.5.1 of random provider
	providerConfigV351 := &types.ProviderConfig{
		Name:    "hashicorp/random",
		Version: "3.5.1",
	}

	err = adapter.LoadProvider(ctx, providerConfigV351)
	require.NoError(t, err)

	// Verify that two different provider binaries were downloaded to separate directories
	// The directories should be named with the versioned provider name
	expectedDirV360 := tmpDir + "/hashicorp_random@3.6.0"
	expectedDirV351 := tmpDir + "/hashicorp_random@3.5.1"

	_, err = os.Stat(expectedDirV360)
	require.NoError(t, err, "v3.6.0 provider directory should exist: %s", expectedDirV360)

	_, err = os.Stat(expectedDirV351)
	require.NoError(t, err, "v3.5.1 provider directory should exist: %s", expectedDirV351)

	// Verify loading the same provider version again doesn't fail (idempotent)
	err = adapter.LoadProvider(ctx, providerConfigV360)
	require.NoError(t, err, "Loading already-cached provider should be idempotent")

	// Create resources with v3.6.0
	configV360 := map[string]any{
		"length":  8,
		"special": false,
	}

	stateV360, err := adapter.CreateResource(ctx, providerConfigV360, "random_string", configV360)
	require.NoError(t, err)

	require.NotEmpty(t, stateV360["result"])
	assert.EqualValues(t, 8, stateV360["length"])
	assert.Equal(t, false, stateV360["special"])

	// Create resources with v3.5.1
	configV351 := map[string]any{
		"length":  10,
		"special": true,
	}

	stateV351, err := adapter.CreateResource(ctx, providerConfigV351, "random_string", configV351)
	require.NoError(t, err)

	require.NotEmpty(t, stateV351["result"])
	assert.EqualValues(t, 10, stateV351["length"])
	assert.Equal(t, true, stateV351["special"])

	// Verify we can use both versions independently after creation
	// Refresh with v3.6.0
	refreshedV360, err := adapter.RefreshResource(ctx, providerConfigV360, "random_string", stateV360)
	require.NoError(t, err)

	assert.Equal(t, stateV360["id"], refreshedV360["id"])
	assert.Equal(t, stateV360["result"], refreshedV360["result"])

	// Refresh with v3.5.1
	refreshedV351, err := adapter.RefreshResource(ctx, providerConfigV351, "random_string", stateV351)
	require.NoError(t, err)

	assert.Equal(t, stateV351["id"], refreshedV351["id"])
	assert.Equal(t, stateV351["result"], refreshedV351["result"])
}
