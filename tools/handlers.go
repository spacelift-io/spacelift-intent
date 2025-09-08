// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"github.com/spacelift-io/spacelift-intent/tools/internal"
	datasourceLifecycle "github.com/spacelift-io/spacelift-intent/tools/lifecycle/datasources"
	resourceLifecycle "github.com/spacelift-io/spacelift-intent/tools/lifecycle/resources"
	"github.com/spacelift-io/spacelift-intent/tools/lifecycle/resources/dependencies"
	"github.com/spacelift-io/spacelift-intent/tools/provider"
	datasourceSchema "github.com/spacelift-io/spacelift-intent/tools/provider/datasources"
	resourceSchema "github.com/spacelift-io/spacelift-intent/tools/provider/resources"
	"github.com/spacelift-io/spacelift-intent/tools/state"
	"github.com/spacelift-io/spacelift-intent/types"
)

// ToolHandlers contains all MCP tool handlers
type ToolHandlers struct {
	registryClient  types.RegistryClient
	providerManager types.ProviderManager
	storage         types.Storage
}

// New creates new tool handlers
func New(registryClient types.RegistryClient, providerManager types.ProviderManager, storage types.Storage) *ToolHandlers {
	return &ToolHandlers{
		registryClient:  registryClient,
		providerManager: providerManager,
		storage:         storage,
	}
}

// RegisterTools registers all MCP tools with the server
func (th *ToolHandlers) Tools() []internal.Tool {
	tools := []internal.Tool{}
	// Register search providers tool
	tools = append(tools, provider.Search(th.registryClient))

	// Register describe provider tool
	tools = append(tools, provider.Describe(th.providerManager))

	// Register describe resource tool
	tools = append(tools, resourceSchema.Describe(th.providerManager))

	// Register create resource tool
	tools = append(tools, resourceLifecycle.Create(th.storage, th.providerManager))

	// Register update resource tool
	tools = append(tools, resourceLifecycle.Update(th.storage, th.providerManager))

	// Register operations resource tool
	tools = append(tools, resourceLifecycle.Operations(th.storage))

	// Register describe data source tool
	tools = append(tools, datasourceSchema.Describe(th.providerManager))

	// Register read data source tool
	tools = append(tools, datasourceLifecycle.Read(th.providerManager))

	// Register get state tool
	tools = append(tools, state.Get(th.storage))

	// Register list states tool
	tools = append(tools, state.List(th.storage))

	// Register delete resource tool
	tools = append(tools, resourceLifecycle.Delete(th.storage, th.providerManager))

	// Register refresh resource tool
	tools = append(tools, resourceLifecycle.Refresh(th.storage, th.providerManager))

	// Register import resource tool
	tools = append(tools, resourceLifecycle.Import(th.storage, th.providerManager))

	// Register eject resource tool
	tools = append(tools, state.Eject(th.storage))

	// Register dependency management tools
	tools = append(tools, dependencies.Add(th.storage))

	tools = append(tools, dependencies.Remove(th.storage))

	tools = append(tools, dependencies.Get(th.storage))

	tools = append(tools, state.Timeline(th.storage))

	return tools
}
