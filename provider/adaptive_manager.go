// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"log"

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
		adapter = legacy.NewManager(tmpDir, registry)
		log.Printf("[INFO] Using legacy provider implementation for %s", tmpDir)
	} else {
		adapter = NewOpenTofuAdapter(tmpDir, registry)
		log.Printf("[INFO] Using opentofu provider implementation for %s", tmpDir)
	}

	return &AdaptiveManager{
		adapter: adapter,
	}
}

func (m *AdaptiveManager) LoadProvider(ctx context.Context, provider *types.ProviderConfig) error {
	return m.adapter.LoadProvider(ctx, provider)
}

func (m *AdaptiveManager) DescribeProvider(ctx context.Context, providerConfig *types.ProviderConfig) (*types.ProviderSchema, *string, error) {
	return m.adapter.DescribeProvider(ctx, providerConfig)
}

func (m *AdaptiveManager) GetProviderVersion(ctx context.Context, providerName string) (string, error) {
	return m.adapter.GetProviderVersion(ctx, providerName)
}

func (m *AdaptiveManager) GetProviderVersions(ctx context.Context, providerName string) ([]types.ProviderVersionInfo, error) {
	return m.adapter.GetProviderVersions(ctx, providerName)
}

func (m *AdaptiveManager) ListResources(ctx context.Context, provider *types.ProviderConfig) ([]string, error) {
	return m.adapter.ListResources(ctx, provider)
}

func (m *AdaptiveManager) DescribeResource(ctx context.Context, provider *types.ProviderConfig, resourceType string) (*types.TypeDescription, error) {
	return m.adapter.DescribeResource(ctx, provider, resourceType)
}

func (m *AdaptiveManager) ListDataSources(ctx context.Context, provider *types.ProviderConfig) ([]string, error) {
	return m.adapter.ListDataSources(ctx, provider)
}

func (m *AdaptiveManager) DescribeDataSource(ctx context.Context, provider *types.ProviderConfig, dataSourceType string) (*types.TypeDescription, error) {
	return m.adapter.DescribeDataSource(ctx, provider, dataSourceType)
}

func (m *AdaptiveManager) PlanResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, currentState *map[string]any, newConfig map[string]any) (map[string]any, error) {
	return m.adapter.PlanResource(ctx, provider, resourceType, currentState, newConfig)
}

func (m *AdaptiveManager) CreateResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, config map[string]any) (map[string]any, error) {
	return m.adapter.CreateResource(ctx, provider, resourceType, config)
}

func (m *AdaptiveManager) UpdateResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, currentState, newConfig map[string]any) (map[string]any, error) {
	return m.adapter.UpdateResource(ctx, provider, resourceType, currentState, newConfig)
}

func (m *AdaptiveManager) DeleteResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, currentState map[string]any) error {
	return m.adapter.DeleteResource(ctx, provider, resourceType, currentState)
}

func (m *AdaptiveManager) ReadDataSource(ctx context.Context, provider *types.ProviderConfig, dataSourceType string, config map[string]any) (map[string]any, error) {
	return m.adapter.ReadDataSource(ctx, provider, dataSourceType, config)
}

func (m *AdaptiveManager) ImportResource(ctx context.Context, provider *types.ProviderConfig, resourceType, resourceID string) (map[string]any, error) {
	return m.adapter.ImportResource(ctx, provider, resourceType, resourceID)
}

func (m *AdaptiveManager) RefreshResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, currentState map[string]any) (map[string]any, error) {
	return m.adapter.RefreshResource(ctx, provider, resourceType, currentState)
}

func (m *AdaptiveManager) Cleanup(ctx context.Context) {
	m.adapter.Cleanup(ctx)
}
