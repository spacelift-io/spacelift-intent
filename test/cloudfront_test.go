package test

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/provider"
	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/storage"
	"github.com/spacelift-io/spacelift-intent/tools"
)

// loadAWSCredentials loads AWS credentials from .env.aws file
func loadAWSCredentials(t *testing.T) map[string]string {
	file, err := os.Open("../.env.aws")
	if err != nil {
		t.Skip("Skipping test: .env.aws file not found")
		return nil
	}
	defer file.Close()

	credentials := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove surrounding quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}

			credentials[key] = value
		}
	}

	require.NoError(t, scanner.Err(), "Failed to read .env.aws file")

	// Verify required AWS credentials are present
	requiredKeys := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION"}
	for _, key := range requiredKeys {
		require.NotEmpty(t, credentials[key], "Missing required AWS credential: %s", key)
	}

	return credentials
}

// NewTestHelperWithTimeout creates a test helper with custom timeout for long-running operations
func NewTestHelperWithTimeout(t *testing.T, timeout time.Duration, optionalDirs ...string) *TestHelper {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Create temporary or custom directories
	var tempDir string
	if len(optionalDirs) > 0 && optionalDirs[0] != "" {
		tempDir = optionalDirs[0]
	} else {
		if testDir, ok := t.Context().Value("testDir").(string); ok && testDir != "" {
			tempDir = testDir
		} else {
			tempDir = t.TempDir()
		}
	}
	dbDir := filepath.Join(tempDir, "db")
	err := os.MkdirAll(dbDir, 0755)
	require.NoError(t, err, "Failed to create database directory")

	// Initialize storage
	stor, err := storage.NewSQLiteStorage(filepath.Join(dbDir, "state.db"))
	require.NoError(t, err, "Failed to initialize storage")

	// Initialize registry client
	registryClient := registry.NewOpenTofuClient()

	// Initialize provider manager
	providerManager := provider.NewAdaptiveManager(tempDir, registryClient)

	// Create tool handlers
	toolHandlers := tools.New(registryClient, providerManager, stor)

	// Convert tools to server tools
	mcpTools := toolHandlers.Tools()
	serverTools := make([]server.ServerTool, 0, len(mcpTools))
	for _, tool := range mcpTools {
		serverTools = append(serverTools, server.ServerTool{
			Tool:    tool.Tool,
			Handler: tool.Handler,
		})
	}

	// Create test server
	testServer := mcptest.NewUnstartedServer(t)
	testServer.AddTools(serverTools...)

	// Start the server
	err = testServer.Start(ctx)
	require.NoError(t, err, "Failed to start test server")

	return &TestHelper{
		t:       t,
		ctx:     ctx,
		cancel:  cancel,
		tempDir: tempDir,
		dbDir:   dbDir,
		server:  testServer,
		client:  testServer.Client(),
	}
}

// TestCloudFrontDistributionCreate tests creating a CloudFront distribution using lifecycle-resources-create
func TestCloudFrontDistributionCreate(t *testing.T) {
	// Load AWS credentials from .env.aws
	credentials := loadAWSCredentials(t)
	if credentials == nil {
		return // Test was skipped
	}

	// Set environment variables for the test
	for key, value := range credentials {
		t.Setenv(key, value)
	}

	// Use extended timeout for CloudFront operations (10 minutes)
	th := NewTestHelperWithTimeout(t, 10*time.Minute)
	defer th.Cleanup()

	const resourceID = "test-cloudfront-distribution"
	const provider = "hashicorp/aws"
	const resourceType = "aws_cloudfront_distribution"

	// CloudFront distribution configuration
	cloudFrontConfig := map[string]any{
		"origin": []map[string]any{
			{
				"domain_name": "example.com",
				"origin_id":   "example",
				"custom_origin_config": []map[string]any{
					{
						"http_port":              80,
						"https_port":             443,
						"origin_protocol_policy": "https-only",
						"origin_ssl_protocols":   []string{"TLSv1.2"},
					},
				},
			},
		},
		"enabled": true,
		"default_cache_behavior": []map[string]any{
			{
				"target_origin_id":       "example",
				"allowed_methods":        []string{"GET", "HEAD"},
				"cached_methods":         []string{"GET", "HEAD"},
				"cache_policy_id":        "658327ea-f89d-4fab-a63d-7e88639e58f6",
				"viewer_protocol_policy": "redirect-to-https",
			},
		},
		"restrictions": []map[string]any{
			{
				"geo_restriction": []map[string]any{
					{
						"restriction_type": "none",
					},
				},
			},
		},
		"viewer_certificate": []map[string]any{
			{
				"cloudfront_default_certificate": true,
			},
		},
		"comment": "Example CloudFront distribution",
		"tags": map[string]string{
			"Environment": "production",
			"Name":        "example-distribution",
		},
	}

	t.Run("CreateCloudFrontDistribution", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-create", map[string]any{
			"resource_id":   resourceID,
			"provider":      provider,
			"resource_type": resourceType,
			"config":        cloudFrontConfig,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-create")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Create result should contain resource ID")
		require.Contains(t, content, "created", "Should show created status")

		t.Logf("CloudFront distribution created successfully: %s", content)
	})

	t.Run("RefreshCloudFrontDistribution", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-refresh", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-refresh")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Refresh result should contain resource ID")

		t.Logf("CloudFront distribution refreshed: %s", content)
	})

	t.Run("GetCloudFrontDistributionState", func(t *testing.T) {
		result, err := th.CallTool("state-get", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "state-get")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "State should contain resource ID")
		require.Contains(t, content, "aws_cloudfront_distribution", "State should contain resource type")

		t.Logf("CloudFront distribution state: %s", content)
	})

	t.Run("DeleteCloudFrontDistribution", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-delete")

		content := th.GetTextContent(result)
		require.Contains(t, content, resourceID, "Delete result should contain resource ID")
		require.Contains(t, content, "deleted", "Should show deleted status")

		t.Logf("CloudFront distribution deleted: %s", content)
	})
}
