package standalone

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"spacelift-intent-mcp/instructions"
	"spacelift-intent-mcp/internal/filesystem"
	"spacelift-intent-mcp/provider"
	"spacelift-intent-mcp/registry"
	"spacelift-intent-mcp/storage"
	"spacelift-intent-mcp/tools"
	"spacelift-intent-mcp/types"
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
	Port       int
	ServerType string
	TmpDir     string
	DBDir      string
}

// NewServer creates a new standalone server instance
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Ensure directories exist
	if err := filesystem.EnsureDirs(config.TmpDir, config.DBDir); err != nil {
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
	providerManager := provider.NewManager(config.TmpDir, registryClient)
	toolHandlers := tools.New(registryClient, providerManager, stateStorage)

	// Create server
	s := &Server{
		toolHandlers:    toolHandlers,
		storage:         stateStorage,
		providerManager: providerManager,
		config:          config,
	}

	s.mcp = server.NewMCPServer(
		"spacelift-intent-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithLogging(),
		server.WithInstructions(instructions.Get()),
	)

	// Register tools
	toolHandlers.RegisterTools(s)

	return s, nil
}

// Start starts the server with the given configuration
func (s *Server) Start(ctx context.Context) error {
	log.Printf("Starting standalone server in %s mode", s.config.ServerType)

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

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
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

// GetMode returns the current operational mode
func (s *Server) GetMode() string {
	return "standalone"
}

// Run starts the server in standalone mode (legacy compatibility method)
func (s *Server) Run(port int, serverType string) error {
	// Update config with provided parameters
	s.config.Port = port
	s.config.ServerType = serverType

	// Start with background context
	return s.Start(context.Background())
}

// AddTool registers an MCP tool handler
func (s *Server) AddTool(tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	s.mcp.AddTool(tool, handler)
}
