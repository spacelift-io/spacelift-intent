// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/spacelift-io/spacelift-intent/allowlist"
	"github.com/spacelift-io/spacelift-intent/instructions"
	"github.com/spacelift-io/spacelift-intent/provider"
	"github.com/spacelift-io/spacelift-intent/registry"
	"github.com/spacelift-io/spacelift-intent/tools"
	"github.com/spacelift-io/spacelift-intent/types"
)

// Server implements the StandaloneServer interface
// This wraps the existing monolithic functionality
type Server struct {
	mcp             *mcp.Server
	toolHandlers    *tools.ToolHandlers
	storage         types.Storage
	providerManager types.ProviderManager
	config          *Config
}

// Config holds configuration for standalone server
type Config struct {
	TmpDir    string
	DBDir     string
	Storage   types.Storage
	Allowlist *allowlist.Allowlist
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

	// Create services
	registryClient := registry.NewAllowlistedClient(registry.NewOpenTofuClient(), config.Allowlist)
	providerManager := provider.NewOpenTofuAdapter(config.TmpDir, registryClient, config.Allowlist)
	toolHandlers := tools.New(registryClient, providerManager, config.Storage)

	// Create server
	s := &Server{
		toolHandlers:    toolHandlers,
		storage:         config.Storage,
		providerManager: providerManager,
		config:          config,
	}

	s.mcp = mcp.NewServer(&mcp.Implementation{
		Name:    "spacelift-intent",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Instructions: instructions.GetWithAllowlist(config.Allowlist),
	})

	for _, tool := range toolHandlers.Tools() {
		s.mcp.AddTool(&tool.Tool, tool.Handler)
	}

	return s, nil
}

// start starts the server with the given configuration
func (s *Server) start(ctx context.Context) error {
	log.Printf("Starting standalone server in stdio mode")
	log.Printf("Starting MCP stdio server")

	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}

// stop gracefully shuts down the server
func (s *Server) stop(ctx context.Context) {
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
}

// AddTool registers an MCP tool handler
func (s *Server) AddTool(tool *mcp.Tool, handler mcp.ToolHandler) {
	s.mcp.AddTool(tool, handler)
}
