// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apparentlymart/opentofu-providers/tofuprovider"
	"github.com/apparentlymart/opentofu-providers/tofuprovider/providerops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/types"
)

// TestSchemaConverter_Integration_AWS_Instance tests conversion with AWS instance schema
// This resource has complex nested blocks like ebs_block_device, network_interface, etc.
func TestSchemaConverter_Integration_AWS_Instance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	converter := &SchemaConverter{}

	// Connect to AWS provider
	provider, err := connectToProvider(ctx, types.ProviderConfig{
		Name:    "hashicorp/aws",
		Version: "6.16.0",
	})
	if err != nil {
		t.Skipf("Could not connect to AWS provider: %v (this is expected in CI)", err)
		return
	}
	defer provider.Close()

	// Get provider schema
	schemaReq := &providerops.GetProviderSchemaRequest{}
	schemaResp, err := provider.GetProviderSchema(ctx, schemaReq)
	require.NoError(t, err, "Should be able to get provider schema")

	// Get aws_instance resource schema
	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())
	instanceSchema, found := resourceSchemas["aws_instance"]
	require.True(t, found, "aws_instance resource should exist in schema")

	t.Run("opentofuSchemaToObjectType", func(t *testing.T) {
		result := converter.opentofuSchemaToObjectType(instanceSchema)

		assert.True(t, result.IsObjectType(), "Result should be an object type")
		attrTypes := result.AttributeTypes()

		// Check for basic attributes
		assert.Contains(t, attrTypes, "id", "Should have id attribute")
		assert.Contains(t, attrTypes, "ami", "Should have ami attribute")
		assert.Contains(t, attrTypes, "instance_type", "Should have instance_type attribute")

		// Check for nested blocks
		assert.Contains(t, attrTypes, "ebs_block_device", "Should have ebs_block_device block")
		ebsType := attrTypes["ebs_block_device"]
		assert.True(t, ebsType.IsListType() || ebsType.IsSetType(), "ebs_block_device should be a collection type")

		// Verify nested block is an object
		elemType := ebsType.ElementType()
		assert.True(t, elemType.IsObjectType(), "ebs_block_device elements should be objects")

		// Check nested block attributes
		ebsAttrs := elemType.AttributeTypes()
		assert.Contains(t, ebsAttrs, "device_name", "ebs_block_device should have device_name")
		assert.Contains(t, ebsAttrs, "volume_size", "ebs_block_device should have volume_size")
	})

	t.Run("convertSchemaToTypeDescription", func(t *testing.T) {
		result := converter.convertSchemaToTypeDescription("hashicorp/aws", "aws_instance", instanceSchema, "resource")

		assert.Equal(t, "hashicorp/aws", result.ProviderName)
		assert.Equal(t, "aws_instance", result.Type)
		assert.NotEmpty(t, result.Description)
		assert.NotEmpty(t, result.Properties, "Should have properties")

		// Check that nested blocks are properly represented
		ebsBlockProp, ok := result.Properties["ebs_block_device"]
		require.True(t, ok, "Should have ebs_block_device property")

		ebsBlock, ok := ebsBlockProp.(map[string]any)
		require.True(t, ok, "ebs_block_device should be a map")
		assert.Equal(t, true, ebsBlock["is_block"], "ebs_block_device should be marked as a block")

		// Check nested block properties
		ebsProps, ok := ebsBlock["properties"].(map[string]any)
		require.True(t, ok, "ebs_block_device should have properties")
		assert.Contains(t, ebsProps, "device_name", "Should have device_name in nested block")
		assert.Contains(t, ebsProps, "volume_size", "Should have volume_size in nested block")
	})

	t.Run("nested blocks contain correct metadata", func(t *testing.T) {
		result := converter.convertSchemaToTypeDescription("hashicorp/aws", "aws_instance", instanceSchema, "resource")

		// Check that nested block has required fields, usage, etc.
		ebsBlock := result.Properties["ebs_block_device"].(map[string]any)

		// Verify it has nesting information
		assert.Contains(t, ebsBlock, "nesting", "Should have nesting information")
		assert.Contains(t, ebsBlock, "type", "Should have type information")
		assert.Contains(t, ebsBlock, "min_items", "Should have min_items")
		assert.Contains(t, ebsBlock, "max_items", "Should have max_items")

		// Check nested properties have correct metadata
		ebsProps := ebsBlock["properties"].(map[string]any)
		for propName, prop := range ebsProps {
			propMap, ok := prop.(map[string]any)
			require.True(t, ok, "Property %s should be a map", propName)
			assert.Contains(t, propMap, "type", "Property %s should have type", propName)
			assert.Contains(t, propMap, "required", "Property %s should have required flag", propName)
			assert.Contains(t, propMap, "usage", "Property %s should have usage", propName)
		}
	})
}

// TestSchemaConverter_Integration_AWS_VPC tests conversion with simpler nested structure
func TestSchemaConverter_Integration_AWS_VPC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	converter := &SchemaConverter{}

	provider, err := connectToProvider(ctx, types.ProviderConfig{
		Name:    "hashicorp/aws",
		Version: "6.16.0",
	})
	if err != nil {
		t.Skipf("Could not connect to AWS provider: %v", err)
		return
	}
	defer provider.Close()

	schemaReq := &providerops.GetProviderSchemaRequest{}
	schemaResp, err := provider.GetProviderSchema(ctx, schemaReq)
	require.NoError(t, err)

	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())
	vpcSchema, found := resourceSchemas["aws_vpc"]
	require.True(t, found, "aws_vpc resource should exist")

	t.Run("convert vpc schema", func(t *testing.T) {
		result := converter.convertSchemaToTypeDescription("hashicorp/aws", "aws_vpc", vpcSchema, "resource")

		assert.Equal(t, "hashicorp/aws", result.ProviderName)
		assert.Equal(t, "aws_vpc", result.Type)

		// VPC has simpler structure - mostly attributes
		assert.Contains(t, result.Properties, "cidr_block")
		assert.Contains(t, result.Properties, "enable_dns_support")
		assert.Contains(t, result.Properties, "tags")
	})
}

// TestSchemaConverter_Integration_AWS_SecurityGroup tests deeply nested blocks
// Security groups have rules which can have nested configurations
func TestSchemaConverter_Integration_AWS_SecurityGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	converter := &SchemaConverter{}

	provider, err := connectToProvider(ctx, types.ProviderConfig{
		Name:    "hashicorp/aws",
		Version: "6.16.0",
	})
	if err != nil {
		t.Skipf("Could not connect to AWS provider: %v", err)
		return
	}
	defer provider.Close()

	schemaReq := &providerops.GetProviderSchemaRequest{}
	schemaResp, err := provider.GetProviderSchema(ctx, schemaReq)
	require.NoError(t, err)

	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())
	sgSchema, found := resourceSchemas["aws_security_group"]
	require.True(t, found, "aws_security_group resource should exist")

	t.Run("security group has nested blocks", func(t *testing.T) {
		result := converter.opentofuSchemaToObjectType(sgSchema)

		attrTypes := result.AttributeTypes()

		// Security groups have ingress and egress blocks
		assert.Contains(t, attrTypes, "ingress", "Should have ingress block")
		assert.Contains(t, attrTypes, "egress", "Should have egress block")

		// Check ingress is a collection of objects
		ingressType := attrTypes["ingress"]
		assert.True(t, ingressType.IsSetType() || ingressType.IsListType(), "ingress should be a collection")

		ingressElem := ingressType.ElementType()
		assert.True(t, ingressElem.IsObjectType(), "ingress elements should be objects")

		// Check ingress has expected fields
		ingressAttrs := ingressElem.AttributeTypes()
		assert.Contains(t, ingressAttrs, "from_port", "ingress should have from_port")
		assert.Contains(t, ingressAttrs, "to_port", "ingress should have to_port")
		assert.Contains(t, ingressAttrs, "protocol", "ingress should have protocol")
	})

	t.Run("convertSchemaToTypeDescription handles ingress/egress", func(t *testing.T) {
		result := converter.convertSchemaToTypeDescription("hashicorp/aws", "aws_security_group", sgSchema, "resource")

		// Check ingress property structure
		ingressProp, ok := result.Properties["ingress"]
		require.True(t, ok, "Should have ingress property")

		ingressMap, ok := ingressProp.(map[string]any)
		require.True(t, ok, "ingress should be a map")

		// In AWS provider v6.16.0+, ingress/egress changed from nested blocks to attributes
		// This test handles both cases for compatibility across provider versions
		if isBlock, hasIsBlock := ingressMap["is_block"]; hasIsBlock && isBlock == true {
			// It's a nested block - verify block structure
			assert.Contains(t, ingressMap, "properties", "ingress block should have properties")

			ingressProps, ok := ingressMap["properties"].(map[string]any)
			require.True(t, ok, "ingress properties should be a map")

			// Verify all properties have correct structure
			assert.Contains(t, ingressProps, "from_port")
			assert.Contains(t, ingressProps, "to_port")
			assert.Contains(t, ingressProps, "protocol")
		} else {
			// It's an attribute - verify it has type information
			assert.Contains(t, ingressMap, "type", "ingress attribute should have type")
			assert.Contains(t, ingressMap, "usage", "ingress attribute should have usage")
			t.Logf("Note: ingress is an attribute (type: %v) in this provider version, not a nested block", ingressMap["type"])
		}
	})
}

// TestSchemaConverter_Integration_AWS_CloudFrontDistribution tests very complex nested structures
// CloudFront distributions have multiple levels of nesting
func TestSchemaConverter_Integration_AWS_CloudFrontDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	converter := &SchemaConverter{}

	provider, err := connectToProvider(ctx, types.ProviderConfig{
		Name:    "hashicorp/aws",
		Version: "6.16.0",
	})
	if err != nil {
		t.Skipf("Could not connect to AWS provider: %v", err)
		return
	}
	defer provider.Close()

	schemaReq := &providerops.GetProviderSchemaRequest{}
	schemaResp, err := provider.GetProviderSchema(ctx, schemaReq)
	require.NoError(t, err)

	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())
	cfSchema, found := resourceSchemas["aws_cloudfront_distribution"]
	require.True(t, found, "aws_cloudfront_distribution resource should exist")

	t.Run("cloudfront has deeply nested blocks", func(t *testing.T) {
		result := converter.opentofuSchemaToObjectType(cfSchema)

		attrTypes := result.AttributeTypes()

		// CloudFront has complex nested structures
		assert.Contains(t, attrTypes, "origin", "Should have origin block")
		assert.Contains(t, attrTypes, "default_cache_behavior", "Should have default_cache_behavior block")

		// Check origin block structure
		originType := attrTypes["origin"]
		assert.True(t, originType.IsSetType() || originType.IsListType(), "origin should be a collection")

		originElem := originType.ElementType()
		assert.True(t, originElem.IsObjectType(), "origin elements should be objects")

		originAttrs := originElem.AttributeTypes()
		assert.Contains(t, originAttrs, "origin_id", "origin should have origin_id")
		assert.Contains(t, originAttrs, "domain_name", "origin should have domain_name")

		// Check for nested blocks within origin (e.g., custom_origin_config)
		if customOriginType, hasCustom := originAttrs["custom_origin_config"]; hasCustom {
			// This is a deeply nested block
			if customOriginType.IsObjectType() || customOriginType.IsListType() {
				t.Logf("Found nested block custom_origin_config within origin")

				var customOriginElem cty.Type
				if customOriginType.IsListType() {
					customOriginElem = customOriginType.ElementType()
				} else {
					customOriginElem = customOriginType
				}

				if customOriginElem.IsObjectType() {
					customAttrs := customOriginElem.AttributeTypes()
					assert.NotEmpty(t, customAttrs, "custom_origin_config should have attributes")
					t.Logf("custom_origin_config has %d attributes", len(customAttrs))
				}
			}
		}
	})

	t.Run("convertSchemaToTypeDescription handles deeply nested blocks", func(t *testing.T) {
		result := converter.convertSchemaToTypeDescription("hashicorp/aws", "aws_cloudfront_distribution", cfSchema, "resource")

		assert.Equal(t, "hashicorp/aws", result.ProviderName)
		assert.Equal(t, "aws_cloudfront_distribution", result.Type)

		// Check origin block
		originProp, ok := result.Properties["origin"]
		require.True(t, ok, "Should have origin property")

		originBlock, ok := originProp.(map[string]any)
		require.True(t, ok, "origin should be a map")
		assert.Equal(t, true, originBlock["is_block"], "origin should be marked as a block")

		// Check for nested blocks within origin
		if nestedBlocks, hasNested := originBlock["nested_blocks"].(map[string]any); hasNested {
			t.Logf("Origin has %d nested blocks", len(nestedBlocks))

			// Check each nested block is properly structured
			for nestedName, nestedBlock := range nestedBlocks {
				nestedMap, ok := nestedBlock.(map[string]any)
				require.True(t, ok, "Nested block %s should be a map", nestedName)
				assert.Equal(t, true, nestedMap["is_block"], "Nested block %s should be marked as a block", nestedName)

				if nestedProps, hasProps := nestedMap["properties"].(map[string]any); hasProps {
					assert.NotEmpty(t, nestedProps, "Nested block %s should have properties", nestedName)
					t.Logf("Nested block %s has %d properties", nestedName, len(nestedProps))
				}
			}
		}

		// Check default_cache_behavior block
		cacheBehaviorProp, ok := result.Properties["default_cache_behavior"]
		require.True(t, ok, "Should have default_cache_behavior property")

		cacheBehaviorBlock, ok := cacheBehaviorProp.(map[string]any)
		require.True(t, ok, "default_cache_behavior should be a map")
		assert.Equal(t, true, cacheBehaviorBlock["is_block"])

		// Cache behavior should have nested blocks too (like forwarded_values)
		if cbNestedBlocks, hasNested := cacheBehaviorBlock["nested_blocks"].(map[string]any); hasNested {
			t.Logf("default_cache_behavior has %d nested blocks", len(cbNestedBlocks))
			assert.NotEmpty(t, cbNestedBlocks, "default_cache_behavior should have nested blocks")
		}
	})

	t.Run("verify three levels of nesting", func(t *testing.T) {
		result := converter.convertSchemaToTypeDescription("hashicorp/aws", "aws_cloudfront_distribution", cfSchema, "resource")

		// Level 1: origin block
		originBlock, ok := result.Properties["origin"].(map[string]any)
		require.True(t, ok, "Should have origin block")

		// Level 2: nested blocks within origin (e.g., custom_origin_config)
		if nestedBlocks, hasNested := originBlock["nested_blocks"].(map[string]any); hasNested && len(nestedBlocks) > 0 {
			// Pick the first nested block
			for nestedName, nestedBlock := range nestedBlocks {
				nestedMap := nestedBlock.(map[string]any)

				// Level 3: Check if this nested block has its own nested blocks
				if deeplyNested, hasDeep := nestedMap["nested_blocks"].(map[string]any); hasDeep && len(deeplyNested) > 0 {
					t.Logf("Found 3 levels of nesting! origin -> %s -> %v", nestedName, deeplyNested)
					assert.NotEmpty(t, deeplyNested, "Should have third level of nesting")
				} else {
					// At least verify we have 2 levels
					t.Logf("Found 2 levels of nesting: origin -> %s", nestedName)
				}
				break // Just check the first one
			}
		}
	})
}

// TestSchemaConverter_Integration_AllNestingModes tests different nesting modes
func TestSchemaConverter_Integration_AllNestingModes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	converter := &SchemaConverter{}

	provider, err := connectToProvider(ctx, types.ProviderConfig{
		Name:    "hashicorp/aws",
		Version: "6.16.0",
	})
	if err != nil {
		t.Skipf("Could not connect to AWS provider: %v", err)
		return
	}
	defer provider.Close()

	schemaReq := &providerops.GetProviderSchemaRequest{}
	schemaResp, err := provider.GetProviderSchema(ctx, schemaReq)
	require.NoError(t, err)

	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())

	// Test various resources to ensure we handle all nesting modes
	testResources := []string{
		"aws_instance",                // Has NestingList and NestingSet blocks
		"aws_security_group",          // Has NestingSet blocks
		"aws_cloudfront_distribution", // Has complex nesting
		"aws_vpc",                     // Simpler structure
	}

	for _, resourceType := range testResources {
		t.Run(resourceType, func(t *testing.T) {
			schema, found := resourceSchemas[resourceType]
			if !found {
				t.Skipf("Resource %s not found in provider schema", resourceType)
				return
			}

			// Test opentofuSchemaToObjectType
			ctyType := converter.opentofuSchemaToObjectType(schema)
			assert.True(t, ctyType.IsObjectType() || ctyType.Equals(cty.DynamicPseudoType),
				"Resource %s should produce an object type or dynamic type", resourceType)

			if ctyType.IsObjectType() {
				attrTypes := ctyType.AttributeTypes()
				assert.NotEmpty(t, attrTypes, "Resource %s should have attributes", resourceType)
			}

			// Test convertSchemaToTypeDescription
			typeDesc := converter.convertSchemaToTypeDescription("hashicorp/aws", resourceType, schema, "resource")
			assert.Equal(t, "hashicorp/aws", typeDesc.ProviderName)
			assert.Equal(t, resourceType, typeDesc.Type)
			assert.NotEmpty(t, typeDesc.Properties, "Resource %s should have properties", resourceType)

			// Verify that any nested blocks are properly marked
			for propName, prop := range typeDesc.Properties {
				if propMap, ok := prop.(map[string]any); ok {
					if isBlock, hasIsBlock := propMap["is_block"]; hasIsBlock && isBlock == true {
						t.Logf("Resource %s has nested block: %s", resourceType, propName)

						// Verify block has required metadata
						assert.Contains(t, propMap, "nesting", "Block %s should have nesting info", propName)
						assert.Contains(t, propMap, "type", "Block %s should have type info", propName)
						assert.Contains(t, propMap, "min_items", "Block %s should have min_items", propName)
						assert.Contains(t, propMap, "max_items", "Block %s should have max_items", propName)
					}
				}
			}
		})
	}
}

// connectToProvider is a helper function to connect to a provider for testing
// It downloads the provider and starts it using the GRPC plugin interface
func connectToProvider(ctx context.Context, provider types.ProviderConfig) (tofuprovider.GRPCPluginProvider, error) {
	// Create temporary directory for provider binaries
	tmpDir, err := os.MkdirTemp("", "provider-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create registry client
	registryClient := registry.NewOpenTofuClient()

	// Get provider download info
	downloadInfo, err := registryClient.GetProviderDownload(ctx, provider)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to get provider download info: %w", err)
	}

	// Download and extract provider binary
	binary, err := downloadAndExtractProvider(ctx, registryClient, provider, tmpDir, downloadInfo.DownloadURL)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to download provider: %w", err)
	}

	// Start the provider
	grpcProvider, err := tofuprovider.StartGRPCPlugin(ctx, binary)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to start provider: %w", err)
	}

	return grpcProvider, nil
}

// downloadAndExtractProvider downloads and extracts a provider binary
func downloadAndExtractProvider(ctx context.Context, registryClient types.RegistryClient, provider types.ProviderConfig, tmpDir, downloadURL string) (string, error) {
	// Create provider directory
	providerDir := filepath.Join(tmpDir, strings.ReplaceAll(provider.Name, "/", "_")+"_"+provider.Version)
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create provider directory: %w", err)
	}

	// Download zip file
	zipPath := filepath.Join(providerDir, "provider.zip")
	resp, err := registryClient.Download(ctx, downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download provider: %w", err)
	}
	defer resp.Close()

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	if _, err := io.Copy(zipFile, resp); err != nil {
		return "", fmt.Errorf("failed to save zip file: %w", err)
	}
	zipFile.Close()

	// Extract zip file
	if err := extractZip(zipPath, providerDir); err != nil {
		return "", fmt.Errorf("failed to extract provider: %w", err)
	}

	// Find provider binary
	binaryPath, err := findProviderBinary(providerDir)
	if err != nil {
		return "", fmt.Errorf("failed to find provider binary: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return binaryPath, nil
}

// findProviderBinary finds the provider binary in a directory
func findProviderBinary(dir string) (string, error) {
	var binaryPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.Contains(info.Name(), "terraform-provider-") {
			binaryPath = path
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if binaryPath == "" {
		return "", errors.New("binary not found")
	}
	return binaryPath, nil
}

// extractZip extracts a zip file to a destination directory
func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			rc.Close()
			continue
		}

		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(file, rc)
		file.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
