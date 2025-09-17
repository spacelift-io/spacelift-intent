package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/spacelift-io/spacelift-intent/instructions"
	"github.com/spacelift-io/spacelift-intent/provider"
	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/storage"
	"github.com/spacelift-io/spacelift-intent/tools"
	"github.com/spacelift-io/spacelift-intent/types"
)

// Server implements the StandaloneServer interface
// This wraps the existing monolithic functionality
type Server struct {
	mcp             *server.MCPServer
	toolHandlers    *tools.ToolHandlers
	storage         types.Storage
	providerManager types.ProviderManager
	config          *Config
}

// Config holds configuration for standalone server
type Config struct {
	TmpDir string
	DBDir  string
}

// newServer creates a new standalone server instance
func newServer(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Ensure directories exist
	if err := ensureDirs(config.TmpDir, config.DBDir); err != nil {
		return nil, err
	}

	// Create state storage
	dbPath := filepath.Join(config.DBDir, "state.db")
	stateStorage, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create state storage: %w", err)
	}

	// Create services
	registryClient := registry.NewOpenTofuClient()
	providerManager := provider.NewAdaptiveManager(config.TmpDir, registryClient)
	toolHandlers := tools.New(registryClient, providerManager, stateStorage)

	// Create server
	s := &Server{
		toolHandlers:    toolHandlers,
		storage:         stateStorage,
		providerManager: providerManager,
		config:          config,
	}

	s.mcp = server.NewMCPServer(
		"spacelift-intent",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithLogging(),
		server.WithInstructions(instructions.Get()),
	)

	for _, tool := range toolHandlers.Tools() {
		s.mcp.AddTool(tool.Tool, tool.Handler)
	}

	return s, nil
}

// start starts the server with the given configuration
func (s *Server) start(ctx context.Context) error {
	log.Printf("Starting standalone server in stdio mode")

	log.Printf("Starting MCP stdio server")

	// Start server in a goroutine so we can handle context cancellation
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ServeStdio(s.mcp)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down stdio server")
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// stop gracefully shuts down the server
func (s *Server) stop(ctx context.Context) error {
	log.Println("Stopping standalone server")

	if s.providerManager != nil {
		s.providerManager.Cleanup(ctx)
	}

	if s.storage != nil {
		if err := s.storage.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}

	log.Println("Standalone server stopped")
	return nil
}

// AddTool registers an MCP tool handler
func (s *Server) AddTool(tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	s.mcp.AddTool(tool, handler)
}
