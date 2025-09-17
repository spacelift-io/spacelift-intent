package provider

import (
	"context"
	"fmt"

	"github.com/spacelift-io/spacelift-intent/provider/legacy"
	"github.com/spacelift-io/spacelift-intent/types"
)

// AdaptiveManager wraps both provider implementations and switches based on feature flags
type AdaptiveManager struct {
	adapter types.ProviderManager
}

// NewAdaptiveManager creates a manager that can switch between implementations
func NewAdaptiveManager(tmpDir string, registry types.RegistryClient) types.ProviderManager {
	var adapter types.ProviderManager

	// Use build tag to determine which implementation to use
	if useLegacyProvider {
		fmt.Println("Using legacy provider")
		adapter = legacy.NewManager(tmpDir, registry)
	} else {
		fmt.Println("Using new provider")
		adapter = NewOpenTofuAdapter(tmpDir, registry)
	}

	return &AdaptiveManager{
		adapter: adapter,
	}
}

func (m *AdaptiveManager) LoadProvider(ctx context.Context, providerName string) error {
	return m.adapter.LoadProvider(ctx, providerName)
}

func (m *AdaptiveManager) GetProviderVersion(ctx context.Context, providerName string) (string, error) {
	return m.adapter.GetProviderVersion(ctx, providerName)
}

func (m *AdaptiveManager) ListResources(ctx context.Context, providerName string) ([]string, error) {
	return m.adapter.ListResources(ctx, providerName)
}

func (m *AdaptiveManager) DescribeResource(ctx context.Context, providerName, resourceType string) (*types.TypeDescription, error) {
	return m.adapter.DescribeResource(ctx, providerName, resourceType)
}

func (m *AdaptiveManager) ListDataSources(ctx context.Context, providerName string) ([]string, error) {
	return m.adapter.ListDataSources(ctx, providerName)
}

func (m *AdaptiveManager) DescribeDataSource(ctx context.Context, providerName, dataSourceType string) (*types.TypeDescription, error) {
	return m.adapter.DescribeDataSource(ctx, providerName, dataSourceType)
}

func (m *AdaptiveManager) PlanResource(ctx context.Context, providerName, resourceType string, currentState *map[string]any, newConfig map[string]any) (map[string]any, error) {
	return m.adapter.PlanResource(ctx, providerName, resourceType, currentState, newConfig)
}

func (m *AdaptiveManager) CreateResource(ctx context.Context, providerName, resourceType string, config map[string]any) (map[string]any, error) {
	return m.adapter.CreateResource(ctx, providerName, resourceType, config)
}

func (m *AdaptiveManager) UpdateResource(ctx context.Context, providerName, resourceType string, currentState, newConfig map[string]any) (map[string]any, error) {
	return m.adapter.UpdateResource(ctx, providerName, resourceType, currentState, newConfig)
}

func (m *AdaptiveManager) DeleteResource(ctx context.Context, providerName, resourceType string, currentState map[string]any) error {
	return m.adapter.DeleteResource(ctx, providerName, resourceType, currentState)
}

func (m *AdaptiveManager) ReadDataSource(ctx context.Context, providerName, dataSourceType string, config map[string]any) (map[string]any, error) {
	return m.adapter.ReadDataSource(ctx, providerName, dataSourceType, config)
}

func (m *AdaptiveManager) ImportResource(ctx context.Context, providerName, resourceType, resourceID string) (map[string]any, error) {
	return m.adapter.ImportResource(ctx, providerName, resourceType, resourceID)
}

func (m *AdaptiveManager) RefreshResource(ctx context.Context, providerName, resourceType string, currentState map[string]any) (map[string]any, error) {
	return m.adapter.RefreshResource(ctx, providerName, resourceType, currentState)
}

func (m *AdaptiveManager) Cleanup(ctx context.Context) {
	m.adapter.Cleanup(ctx)
}
