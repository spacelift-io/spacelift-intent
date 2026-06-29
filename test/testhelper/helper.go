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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // Import SQLite driver for database/sql.

	"github.com/spacelift-io/spacelift-intent/provider"
	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/storage"
	"github.com/spacelift-io/spacelift-intent/tools"
	"github.com/spacelift-io/spacelift-intent/types"
)

// TestHelper encapsulates test setup and utilities
type TestHelper struct {
	t       *testing.T
	Ctx     context.Context
	cancel  context.CancelFunc
	tempDir string
	dbDir   string
	server  *mcp.Server
	session *mcp.ClientSession
	Storage types.Storage
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
	store, err := storage.NewSQLiteStorage(filepath.Join(dbDir, "state.db"))
	require.NoError(t, err, "Failed to initialize storage")

	require.NoError(t, store.Migrate())

	// Initialize registry client
	registryClient := registry.NewOpenTofuClient()

	// Initialize provider manager
	providerManager := provider.NewOpenTofuAdapter(tempDir, registryClient, nil)

	// Create tool handlers
	toolHandlers := tools.New(registryClient, providerManager, store)

	// Create test server
	server := mcp.NewServer(&mcp.Implementation{Name: t.Name(), Version: "1.0.0"}, nil)
	for _, tool := range toolHandlers.Tools() {
		server.AddTool(&tool.Tool, tool.Handler)
	}

	// Create in-memory transports for testing
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Start server in background
	go func() {
		if _, err := server.Connect(ctx, serverTransport, nil); err != nil && ctx.Err() == nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Create and connect client
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err, "Failed to connect test client")

	return &TestHelper{
		t:       t,
		Ctx:     ctx,
		cancel:  cancel,
		tempDir: tempDir,
		dbDir:   dbDir,
		server:  server,
		session: session,
		Storage: store,
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
	providerManager := provider.NewOpenTofuAdapter(tempDir, registryClient, nil)

	// Create tool handlers
	toolHandlers := tools.New(registryClient, providerManager, stor)

	// Create test server
	server := mcp.NewServer(&mcp.Implementation{Name: t.Name(), Version: "1.0.0"}, nil)
	for _, tool := range toolHandlers.Tools() {
		server.AddTool(&tool.Tool, tool.Handler)
	}

	// Create in-memory transports for testing
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Start server in background
	go func() {
		if _, err := server.Connect(ctx, serverTransport, nil); err != nil && ctx.Err() == nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Create and connect client
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err, "Failed to connect test client")

	return &TestHelper{
		t:       t,
		Ctx:     ctx,
		cancel:  cancel,
		tempDir: tempDir,
		dbDir:   dbDir,
		server:  server,
		session: session,
		Storage: stor,
	}
}

// Cleanup closes the test helper and cleans up resources
func (th *TestHelper) Cleanup() {
	// Close client session first
	if th.session != nil {
		th.session.Close()
	}
	// Cancel context to gracefully stop server
	if th.cancel != nil {
		th.cancel()
	}
}

// CallTool is a convenience method to call an MCP tool
func (th *TestHelper) CallTool(toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	return th.session.CallTool(th.Ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
}

// ListTools returns the list of tools available on the server
func (th *TestHelper) ListTools() (*mcp.ListToolsResult, error) {
	return th.session.ListTools(th.Ctx, &mcp.ListToolsParams{})
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
	if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
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
