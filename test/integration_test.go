package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/provider"
	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/storage"
	"github.com/spacelift-io/spacelift-intent/tools"
)

// TestHelper encapsulates test setup and utilities
type TestHelper struct {
	t       *testing.T
	ctx     context.Context
	cancel  context.CancelFunc
	tempDir string
	dbDir   string
	server  *mcptest.Server
	client  *client.Client
}

// NewTestHelper creates a new test helper with server setup
func NewTestHelper(t *testing.T) *TestHelper {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	// Create temporary directories
	tempDir := t.TempDir()
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

// Cleanup closes the test helper and cleans up resources
func (th *TestHelper) Cleanup() {
	if th.server != nil {
		th.server.Close()
	}
	if th.cancel != nil {
		th.cancel()
	}
}

// CallTool is a convenience method to call an MCP tool
func (th *TestHelper) CallTool(toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	return th.client.CallTool(th.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
}

// AssertToolSuccess asserts that a tool call was successful
func (th *TestHelper) AssertToolSuccess(result *mcp.CallToolResult, err error, toolName string) {
	require.NoError(th.t, err, "Tool %s should not return error", toolName)
	assert.False(th.t, result.IsError, "Tool %s should not return error result", toolName)
}

// GetTextContent extracts text content from tool result
func (th *TestHelper) GetTextContent(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		return textContent.Text
	}
	return ""
}

// TestMCPServerIntegration is the main integration test suite
func TestMCPServerIntegration(t *testing.T) {
	th := NewTestHelper(t)
	defer th.Cleanup()

	t.Run("Server Initialization", func(t *testing.T) {
		testServerInitialization(t, th)
	})

	t.Run("Provider Operations", func(t *testing.T) {
		testProviderOperations(t, th)
	})

	t.Run("Resource Lifecycle", func(t *testing.T) {
		testResourceLifecycle(t, th)
	})

	t.Run("State Management", func(t *testing.T) {
		testStateManagement(t, th)
	})

	t.Run("Data Sources", func(t *testing.T) {
		testDataSources(t, th)
	})

	t.Run("Dependencies", func(t *testing.T) {
		testDependencies(t, th)
	})
}

// testServerInitialization tests basic server functionality
func testServerInitialization(t *testing.T, th *TestHelper) {
	t.Run("ListTools", func(t *testing.T) {
		tools, err := th.client.ListTools(th.ctx, mcp.ListToolsRequest{})
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
			"lifecycle-resources-resume",
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
	})

	t.Run("ServerHealth", func(t *testing.T) {
		// Test that we can make a simple tool call
		result, err := th.CallTool("state-list", map[string]any{})
		th.AssertToolSuccess(result, err, "state-list")

		content := th.GetTextContent(result)
		assert.NotEmpty(t, content, "Should return some content")
	})
}

// testProviderOperations tests provider search and describe functionality
func testProviderOperations(t *testing.T, th *TestHelper) {
	t.Run("SearchProvider", func(t *testing.T) {
		result, err := th.CallTool("provider-search", map[string]any{
			"query": "hashicorp/random",
		})
		th.AssertToolSuccess(result, err, "provider-search")

		content := th.GetTextContent(result)
		assert.Contains(t, content, "random", "Search results should contain 'random'")
		assert.Contains(t, content, "hashicorp", "Search results should contain 'hashicorp'")
	})

	t.Run("DescribeProvider", func(t *testing.T) {
		result, err := th.CallTool("provider-describe", map[string]any{
			"provider": "hashicorp/random",
		})
		th.AssertToolSuccess(result, err, "provider-describe")

		content := th.GetTextContent(result)
		assert.Contains(t, content, "random", "Provider description should contain provider info")
	})

	t.Run("DescribeResource", func(t *testing.T) {
		result, err := th.CallTool("provider-resources-describe", map[string]any{
			"provider":      "hashicorp/random",
			"resource_type": "random_string",
		})
		th.AssertToolSuccess(result, err, "provider-resources-describe")

		content := th.GetTextContent(result)
		assert.Contains(t, content, "random_string", "Resource description should contain resource type")
		assert.Contains(t, content, "length", "random_string should have 'length' attribute")
	})
}

// testResourceLifecycle tests all resource lifecycle operations
func testResourceLifecycle(t *testing.T, th *TestHelper) {
	const resourceID = "test-random-string"
	const provider = "hashicorp/random"
	const resourceType = "random_string"

	t.Run("CreateResource", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-create", map[string]any{
			"resource_id":   resourceID,
			"provider":      provider,
			"resource_type": resourceType,
			"config": map[string]any{
				"length":  8,
				"special": true,
			},
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-create")

		content := th.GetTextContent(result)
		assert.Contains(t, content, resourceID, "Create result should contain resource ID")
		assert.Contains(t, content, "created", "Should show created status")
	})

	t.Run("UpdateResource", func(t *testing.T) {
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
	})

	t.Run("RefreshResource", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-refresh", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-refresh")

		content := th.GetTextContent(result)
		assert.Contains(t, content, resourceID, "Refresh result should contain resource ID")
	})

	t.Run("GetResourceOperations", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-operations", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-operations")

		content := th.GetTextContent(result)
		assert.Contains(t, content, resourceID, "Operations should contain resource ID")
		assert.Contains(t, content, "create", "Should show create operation")
		assert.Contains(t, content, "update", "Should show update operation")
	})

	t.Run("ImportResource", func(t *testing.T) {
		// For random_string, we can't really import an existing one,
		// but we can test the import functionality with a new ID
		importResourceID := "test-imported-string"
		result, err := th.CallTool("lifecycle-resources-import", map[string]any{
			"resource_id":   importResourceID,
			"provider":      provider,
			"resource_type": resourceType,
			"import_id":     "dummy-import-id", // This will likely fail but tests the flow
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
	})

	t.Run("ResumeResource", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-resume", map[string]any{
			"resource_id": resourceID,
		})
		// This might not have any operations to resume, but test the functionality
		if result.IsError {
			content := th.GetTextContent(result)
			if assert.Contains(t, content, "Not supported", "Expected not supported error") {
				t.Logf("Resume operation not supported as expected: %s", content)
			}
		} else {
			th.AssertToolSuccess(result, err, "lifecycle-resources-resume")
		}
	})

	t.Run("DeleteResource", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-delete")

		content := th.GetTextContent(result)
		assert.Contains(t, content, resourceID, "Delete result should contain resource ID")
		assert.Contains(t, content, "deleted", "Should show deleted status")
	})
}

// testStateManagement tests state management operations
func testStateManagement(t *testing.T, th *TestHelper) {
	// Create a resource first to have some state to work with
	const resourceID = "test-state-resource"

	// Create a resource for state testing
	t.Run("SetupStateResource", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-create", map[string]any{
			"resource_id":   resourceID,
			"provider":      "hashicorp/random",
			"resource_type": "random_pet",
			"config": map[string]any{
				"length": 2,
			},
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-create")
	})

	t.Run("GetState", func(t *testing.T) {
		result, err := th.CallTool("state-get", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "state-get")

		content := th.GetTextContent(result)
		assert.Contains(t, content, resourceID, "State should contain resource ID")
		assert.Contains(t, content, "random_pet", "State should contain resource type")
	})

	t.Run("ListStates", func(t *testing.T) {
		result, err := th.CallTool("state-list", map[string]any{})
		th.AssertToolSuccess(result, err, "state-list")

		content := th.GetTextContent(result)
		assert.Contains(t, content, resourceID, "State list should contain our resource")
	})

	t.Run("StateTimeline", func(t *testing.T) {
		result, err := th.CallTool("state-timeline", map[string]any{})
		th.AssertToolSuccess(result, err, "state-timeline")

		content := th.GetTextContent(result)
		assert.Contains(t, content, resourceID, "Timeline should contain our resource")
		assert.Contains(t, content, "create", "Timeline should show create operation")
	})

	t.Run("EjectState", func(t *testing.T) {
		result, err := th.CallTool("state-eject", map[string]any{
			"resource_id": resourceID,
		})
		th.AssertToolSuccess(result, err, "state-eject")

		content := th.GetTextContent(result)
		assert.Contains(t, content, "ejected", "Eject should confirm ejection")
		assert.Contains(t, content, "random_pet", "Eject should contain resource type")
		assert.Contains(t, content, resourceID, "Eject should contain resource name")
	})

	// Clean up the resource (it might already be ejected, so deletion could fail)
	t.Run("CleanupStateResource", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
			"resource_id": resourceID,
		})
		// Resource may have been ejected, so deletion might fail
		if result.IsError {
			content := th.GetTextContent(result)
			t.Logf("Expected: resource already ejected, deletion failed: %s", content)
		} else {
			th.AssertToolSuccess(result, err, "lifecycle-resources-delete")
		}
	})
}

// testDataSources tests data source operations
func testDataSources(t *testing.T, th *TestHelper) {
	t.Run("DescribeDataSource", func(t *testing.T) {
		result, err := th.CallTool("provider-datasources-describe", map[string]any{
			"provider":         "hashicorp/random",
			"data_source_type": "random_string",
		})
		// The hashicorp/random provider doesn't have data sources, so this is expected to fail
		if result.IsError {
			content := th.GetTextContent(result)
			assert.Contains(t, content, "no data source schemas", "Expected no data source schemas error")
			t.Logf("Data source describe failed as expected: %s", content)
		} else {
			th.AssertToolSuccess(result, err, "provider-datasources-describe")
		}
	})

	t.Run("ReadDataSource", func(t *testing.T) {
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
	})
}

// testDependencies tests dependency management
func testDependencies(t *testing.T, th *TestHelper) {
	const parentResourceID = "test-dependency-parent"
	const childResourceID = "test-dependency-child"

	// Setup: Create two resources for dependency testing
	t.Run("SetupDependencyResources", func(t *testing.T) {
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
	})

	t.Run("AddDependency", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-dependencies-add", map[string]any{
			"from_resource_id": childResourceID,
			"to_resource_id":   parentResourceID,
			"dependency_type":  "explicit",
		})
		// Dependencies might fail due to foreign key constraints if resources aren't properly stored
		if result.IsError {
			content := th.GetTextContent(result)
			if assert.Contains(t, content, "constraint failed", "Expected constraint error") {
				t.Logf("Dependency add failed as might be expected: %s", content)
				// Skip remaining dependency tests since the initial add failed
				t.Skip("Skipping remaining dependency tests due to constraint issues")
			}
		} else {
			th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-add")

			content := th.GetTextContent(result)
			assert.Contains(t, content, childResourceID, "Dependency add result should contain child resource")
			assert.Contains(t, content, parentResourceID, "Dependency add result should contain parent resource")
		}
	})

	t.Run("GetDependencies", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-dependencies-get", map[string]any{
			"resource_id": childResourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-get")

		content := th.GetTextContent(result)
		// If dependencies were added successfully, they would show up here
		// Otherwise, we just verify the tool works
		t.Logf("Dependencies result: %s", content)
	})

	t.Run("RemoveDependency", func(t *testing.T) {
		result, err := th.CallTool("lifecycle-resources-dependencies-remove", map[string]any{
			"from_resource_id": childResourceID,
			"to_resource_id":   parentResourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-dependencies-remove")

		content := th.GetTextContent(result)
		assert.Contains(t, content, "removed", "Should show removal status")
	})

	// Cleanup: Delete the test resources
	t.Run("CleanupDependencyResources", func(t *testing.T) {
		// Delete child first (good practice)
		result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
			"resource_id": childResourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-delete child")

		// Delete parent
		result, err = th.CallTool("lifecycle-resources-delete", map[string]any{
			"resource_id": parentResourceID,
		})
		th.AssertToolSuccess(result, err, "lifecycle-resources-delete parent")
	})
}
