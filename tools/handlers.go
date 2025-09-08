package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"spacelift-intent-mcp/policy"
	datasourceLifecycle "spacelift-intent-mcp/tools/lifecycle/datasources"
	resourceLifecycle "spacelift-intent-mcp/tools/lifecycle/resources"
	"spacelift-intent-mcp/tools/lifecycle/resources/dependencies"
	policyTools "spacelift-intent-mcp/tools/policy"
	projectTools "spacelift-intent-mcp/tools/project"
	"spacelift-intent-mcp/tools/provider"
	datasourceSchema "spacelift-intent-mcp/tools/provider/datasources"
	resourceSchema "spacelift-intent-mcp/tools/provider/resources"
	"spacelift-intent-mcp/tools/state"
	"spacelift-intent-mcp/types"
)

// Server interface defines the MCP server
type Server interface {
	AddTool(tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error))
}

// ToolHandlers contains all MCP tool handlers
type ToolHandlers struct {
	registryClient  types.RegistryClient
	providerManager types.ProviderManager
	storage         types.Storage
	policyEngine    types.PolicyEngine
	policyEvaluator types.PolicyEvaluator
}

// New creates new tool handlers
func New(registryClient types.RegistryClient, providerManager types.ProviderManager, storage types.Storage, policyEngine types.PolicyEngine) *ToolHandlers {
	return &ToolHandlers{
		registryClient:  registryClient,
		providerManager: providerManager,
		storage:         storage,
		policyEngine:    policyEngine,
		policyEvaluator: policy.NewEvaluator(storage, policyEngine),
	}
}

// RegisterTools registers all MCP tools with the server
func (th *ToolHandlers) RegisterTools(s Server) {
	// Register search providers tool
	tool, handler := provider.Search(th.registryClient)
	s.AddTool(tool, handler)

	// Register describe provider tool
	tool, handler = provider.Describe(th.providerManager)
	s.AddTool(tool, handler)

	// Register describe resource tool
	tool, handler = resourceSchema.Describe(th.providerManager)
	s.AddTool(tool, handler)

	// Register create resource tool
	tool, handler = resourceLifecycle.Create(th.storage, th.providerManager, th.policyEvaluator)
	s.AddTool(tool, handler)

	// Register update resource tool
	tool, handler = resourceLifecycle.Update(th.storage, th.providerManager, th.policyEvaluator)
	s.AddTool(tool, handler)

	// Register operations resource tool
	tool, handler = resourceLifecycle.Operations(th.storage)
	s.AddTool(tool, handler)

	// Register describe data source tool
	tool, handler = datasourceSchema.Describe(th.providerManager)
	s.AddTool(tool, handler)

	// Register read data source tool
	tool, handler = datasourceLifecycle.Read(th.providerManager)
	s.AddTool(tool, handler)

	// Register get state tool
	tool, handler = state.Get(th.storage)
	s.AddTool(tool, handler)

	// Register list states tool
	tool, handler = state.List(th.storage)
	s.AddTool(tool, handler)

	// Register delete resource tool
	tool, handler = resourceLifecycle.Delete(th.storage, th.providerManager, th.policyEvaluator)
	s.AddTool(tool, handler)

	// Register refresh resource tool
	tool, handler = resourceLifecycle.Refresh(th.storage, th.providerManager, th.policyEvaluator)
	s.AddTool(tool, handler)

	// Register import resource tool
	tool, handler = resourceLifecycle.Import(th.storage, th.providerManager, th.policyEvaluator)
	s.AddTool(tool, handler)

	// Register resume resource tool
	tool, handler = resourceLifecycle.Resume(th.storage, th.providerManager, th.policyEvaluator)
	s.AddTool(tool, handler)

	// Register eject resource tool
	tool, handler = state.Eject(th.storage)
	s.AddTool(tool, handler)

	// Register dependency management tools
	tool, handler = dependencies.Add(th.storage)
	s.AddTool(tool, handler)

	tool, handler = dependencies.Remove(th.storage)
	s.AddTool(tool, handler)

	tool, handler = dependencies.Get(th.storage)
	s.AddTool(tool, handler)

	tool, handler = state.Timeline(th.storage)
	s.AddTool(tool, handler)

	// Register policy management tools
	tool, handler = policyTools.Create(th.storage, th.policyEngine)
	s.AddTool(tool, handler)

	tool, handler = policyTools.List(th.storage)
	s.AddTool(tool, handler)

	tool, handler = policyTools.Get(th.storage)
	s.AddTool(tool, handler)

	tool, handler = policyTools.Update(th.storage, th.policyEngine)
	s.AddTool(tool, handler)

	tool, handler = policyTools.Delete(th.storage)
	s.AddTool(tool, handler)

	tool, handler = policyTools.DryRun(th.storage, th.policyEngine)
	s.AddTool(tool, handler)

	tool, handler = policyTools.InputSchema()
	s.AddTool(tool, handler)

	// Register intent project management tools
	tool, handler = projectTools.Create()
	s.AddTool(tool, handler)

	tool, handler = projectTools.Delete()
	s.AddTool(tool, handler)

	tool, handler = projectTools.Describe()
	s.AddTool(tool, handler)

	tool, handler = projectTools.List()
	s.AddTool(tool, handler)

	tool, handler = projectTools.Current()
	s.AddTool(tool, handler)

	tool, handler = projectTools.Use()
	s.AddTool(tool, handler)
}
