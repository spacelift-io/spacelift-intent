package tools

import (
	"spacelift-intent-mcp/tools/internal"
	datasourceLifecycle "spacelift-intent-mcp/tools/lifecycle/datasources"
	resourceLifecycle "spacelift-intent-mcp/tools/lifecycle/resources"
	"spacelift-intent-mcp/tools/lifecycle/resources/dependencies"
	projectTools "spacelift-intent-mcp/tools/project"
	"spacelift-intent-mcp/tools/provider"
	datasourceSchema "spacelift-intent-mcp/tools/provider/datasources"
	resourceSchema "spacelift-intent-mcp/tools/provider/resources"
	"spacelift-intent-mcp/tools/state"
	"spacelift-intent-mcp/types"
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

	// Register resume resource tool
	tools = append(tools, resourceLifecycle.Resume(th.storage, th.providerManager))

	// Register eject resource tool
	tools = append(tools, state.Eject(th.storage))

	// Register dependency management tools
	tools = append(tools, dependencies.Add(th.storage))

	tools = append(tools, dependencies.Remove(th.storage))

	tools = append(tools, dependencies.Get(th.storage))

	tools = append(tools, state.Timeline(th.storage))

	// Register intent project management tools
	tools = append(tools, projectTools.Create())

	tools = append(tools, projectTools.Delete())

	tools = append(tools, projectTools.Describe())

	tools = append(tools, projectTools.List())

	tools = append(tools, projectTools.Current())

	tools = append(tools, projectTools.Use())

	return tools
}
