// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package testhelper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spacelift-io/spacelift-intent/provider"
	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/storage"
	"github.com/spacelift-io/spacelift-intent/tools"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // Import SQLite driver for database/sql.
)

// TestHelper encapsulates test setup and utilities
type TestHelper struct {
	t       *testing.T
	Ctx     context.Context
	cancel  context.CancelFunc
	tempDir string
	dbDir   string
	server  *mcptest.Server
	Client  *client.Client
}

// NewTestHelper creates a new test helper with server setup
func NewTestHelper(t *testing.T, optionalDirs ...string) *TestHelper {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	// Create temporary or custom directories
	var tempDir string
	if len(optionalDirs) > 0 && optionalDirs[0] != "" {
		tempDir = optionalDirs[0]
	} else {
		tempDir = t.TempDir()
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
		Ctx:     ctx,
		cancel:  cancel,
		tempDir: tempDir,
		dbDir:   dbDir,
		server:  testServer,
		Client:  testServer.Client(),
	}
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
		Ctx:     ctx,
		cancel:  cancel,
		tempDir: tempDir,
		dbDir:   dbDir,
		server:  testServer,
		Client:  testServer.Client(),
	}
}

// Cleanup closes the test helper and cleans up resources
func (th *TestHelper) Cleanup() {
	// Cancel context first to gracefully stop operations
	if th.cancel != nil {
		th.cancel()
	}
	// Then close the server after operations have stopped
	if th.server != nil {
		th.server.Close()
	}
}

// CallTool is a convenience method to call an MCP tool
func (th *TestHelper) CallTool(toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	return th.Client.CallTool(th.Ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
}

// AssertToolSuccess asserts that a tool call was successful
func (th *TestHelper) AssertToolSuccess(result *mcp.CallToolResult, err error, toolName string) {
	require.NoError(th.t, err, "Tool %s should not return error", toolName)

	if result.IsError {
		// Get the full error content for debugging
		errorContent := th.GetTextContent(result)
		require.False(th.t, result.IsError, "Tool %s failed with error: %s", toolName, errorContent)
	}
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

// GenerateUniqueResourceID generates a unique resource ID for testing
func GenerateUniqueResourceID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// CleanupResource attempts to delete a resource if it exists
func (th *TestHelper) CleanupResource(resourceID string) {
	result, err := th.CallTool("lifecycle-resources-delete", map[string]any{
		"resource_id": resourceID,
	})
	if err != nil {
		th.t.Logf("Warning: Failed to cleanup resource %s: %v", resourceID, err)
	} else if result.IsError {
		th.t.Logf("Warning: Failed to cleanup resource %s: %s", resourceID, th.GetTextContent(result))
	} else {
		th.t.Logf("Successfully cleaned up resource %s", resourceID)
	}
}

// CreateTestResource creates a resource and returns its ID for testing
func (th *TestHelper) CreateTestResource(providerName, providerVersion, resourceType string, config map[string]any) string {
	resourceID := GenerateUniqueResourceID("test-resource")

	result, err := th.CallTool("lifecycle-resources-create", map[string]any{
		"resource_id":      resourceID,
		"provider":         providerName,
		"provider_version": providerVersion,
		"resource_type":    resourceType,
		"config":           config,
	})

	th.AssertToolSuccess(result, err, "lifecycle-resources-create")
	return resourceID
}

// EnsureResourceExists creates a resource if it doesn't exist in state
func (th *TestHelper) EnsureResourceExists(resourceID, provider, resourceType string, config map[string]any) {
	result, _ := th.CallTool("state-get", map[string]any{
		"resource_id": resourceID,
	})

	if result.IsError {
		content := th.GetTextContent(result)
		if contains(content, "No state found") {
			th.t.Logf("Resource not found in state, creating it...")
			createResult, createErr := th.CallTool("lifecycle-resources-create", map[string]any{
				"resource_id":   resourceID,
				"provider":      provider,
				"resource_type": resourceType,
				"config":        config,
			})
			th.AssertToolSuccess(createResult, createErr, "lifecycle-resources-create")
		}
	}
}

// contains is a simple helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

// indexOf finds the first occurrence of substr in s
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
