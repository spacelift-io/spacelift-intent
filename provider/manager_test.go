package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	plannedState, err := adapter.PlanResource(ctx, "hashicorp/random", "random_password", nil, config)
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

	createdState, err := adapter.CreateResource(ctx, "hashicorp/random", "random_string", config)
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

	// First create a resource to have current state
	initialConfig := map[string]any{
		"length":  6,
		"special": false,
	}

	currentState, err := adapter.CreateResource(ctx, "hashicorp/random", "random_string", initialConfig)
	require.NoError(t, err)
	require.NotEmpty(t, currentState["result"])

	// Now update it with new config
	newConfig := map[string]any{
		"length":  10,
		"special": true,
	}

	updatedState, err := adapter.UpdateResource(ctx, "hashicorp/random", "random_string", currentState, newConfig)
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

	// First create a resource to delete
	config := map[string]any{
		"length":  8,
		"special": false,
	}

	state, err := adapter.CreateResource(ctx, "hashicorp/random", "random_string", config)
	require.NoError(t, err)
	require.NotEmpty(t, state["result"])

	// Now delete the resource
	// TODO(michal): why do we need state to delete the resource?
	err = adapter.DeleteResource(ctx, "hashicorp/random", "random_string", state)
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

	// First create a resource to refresh
	config := map[string]any{
		"length":  12,
		"special": true,
	}

	state, err := adapter.CreateResource(ctx, "hashicorp/random", "random_string", config)
	require.NoError(t, err)
	require.NotEmpty(t, state["result"])

	// Now refresh the resource
	refreshedState, err := adapter.RefreshResource(ctx, "hashicorp/random", "random_string", state)
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

	// Test ListResources with random provider
	resources, err := adapter.ListResources(ctx, "hashicorp/random")
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

	// Test DescribeResource with random_string
	description, err := adapter.DescribeResource(ctx, "hashicorp/random", "random_string")
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
	assert.Equal(t, "object", keepersProp["type"])
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

	state, err := adapter.CreateResource(ctx, "hashicorp/random", "random_string", config)
	require.NoError(t, err)
	require.NotEmpty(t, state["id"])

	resourceID := state["id"].(string)

	// Now import the resource using its ID
	importedState, err := adapter.ImportResource(ctx, "hashicorp/random", "random_string", resourceID)
	require.NoError(t, err)

	// Basic validations
	assert.NotNil(t, importedState)
	assert.Equal(t, resourceID, importedState["id"])
	assert.Equal(t, state["result"], importedState["result"]) // Should be the same random string
	assert.EqualValues(t, 8, importedState["length"])

	t.Logf("Imported random_string resource: %+v", importedState)
}

func TestProviderManagerSuite(t *testing.T) {
	t.Run("PlanResource", TestPlanResource)
	t.Run("CreateResource", TestCreateResource)
	t.Run("UpdateResource", TestUpdateResource)
	t.Run("DeleteResource", TestDeleteResource)
	t.Run("RefreshResource", TestRefreshResource)
	t.Run("ListResources", TestListResources)
	t.Run("DescribeResource", TestDescribeResource)
	t.Run("ImportResource", TestImportResource)
}

func TestCreateCloudFrontDistributionResource(t *testing.T) {
	// Load AWS environment variables from .env.aws file
	envVars, err := loadEnvFile("/Users/michalgolinski/spacelift/exp-infra-as-intent/.env.aws")
	if err != nil {
		t.Fatalf("Warning: Could not load .env.aws file: %v", err)
	} else {
		t.Logf("Loaded environment variables: %v", envVars)
	}

	tmpDir := "/Users/michalgolinski/spacelift/exp-infra-as-intent/provider/test-providers"
	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	registryClient := registry.NewOpenTofuClient()
	// os.Setenv("USE_OPENTOFU_PROVIDER_LIB", "true")
	adapter := NewAdaptiveManager(tmpDir, registryClient)
	ctx := context.Background()
	defer adapter.Cleanup(ctx)

	// Parse the JSON configuration
	configJSON := `{
    "enabled": true,
    "default_cache_behavior": [{
        "allowed_methods": ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"],
        "cached_methods": ["GET", "HEAD"],
        "target_origin_id": "S3-my-cloudfront-origin-ef1e932e",
        "viewer_protocol_policy": "redirect-to-https",
        "compress": true,
        "forwarded_values": [
          {
            "query_string": false,
            "cookies": [
              {
                "forward": "none"
              }
            ]
          }
        ],
        "min_ttl": 0,
        "default_ttl": 3600,
        "max_ttl": 86400,
        "smooth_streaming": false
      }],
    "origin":[
      {
        "domain_name": "my-cloudfront-origin-ef1e932e.s3.eu-west-1.amazonaws.com",
        "origin_id": "S3-my-cloudfront-origin-ef1e932e",
        "origin_access_control_id": "E13S6M2AB1BGFT",
        "origin_path": "",
        "connection_attempts": 3,
        "connection_timeout": 10
      }],
    "restrictions": [
      {
        "geo_restriction": [
          {
			"locations": [],
            "restriction_type": "none"
          }
        ]
      }
    ],
    "viewer_certificate":[
      {
        "cloudfront_default_certificate": true
      }],
    "comment": "Simple CloudFront Distribution",
    "default_root_object": "index.html",
    "price_class": "PriceClass_All",
    "http_version": "http2",
    "is_ipv6_enabled": false,
    "wait_for_deployment": true,
    "retain_on_delete": false,
    "staging": false,
    "tags": {
      "Name": "Simple CloudFront Distribution",
      "Environment": "demo"
    }
  }`

	var config map[string]any
	err = json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		t.Fatalf("Failed to parse config JSON: %v", err)
	}

	// Plan the resource first
	response, err := adapter.PlanResource(ctx, "hashicorp/aws", "aws_cloudfront_distribution", nil, config)
	if err != nil {
		t.Fatalf("PlanResource failed: %v", err)
	}

	spew.Dump(response)
	// // Apply the resource
	// result, err := adapter.CreateResource(ctx, "hashicorp/aws", "aws_cloudfront_distribution", nil, config)
	// if err != nil {
	// 	t.Fatalf("ApplyResource failed: %v", err)
	// }

	// if result == nil {
	// 	t.Fatal("Expected non-nil result")
	// }

	// // Output the result in the same format as the MCP server would return to the LLM
	// jsonResult, err := json.Marshal(result)
	// if err != nil {
	// 	t.Fatalf("Failed to marshal result: %v", err)
	// }

	// t.Logf("MCP Server JSON Response:\n%s", string(jsonResult))

}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var envVars []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
		envVars = append(envVars, key)
	}

	return envVars, scanner.Err()
}
