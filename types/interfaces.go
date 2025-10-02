package types

import (
	"context"
	"io"
)

// ProviderManager interface defines provider management operations
type ProviderManager interface {
	// Provider lifecycle
	LoadProvider(ctx context.Context, provider *ProviderConfig) error
	Cleanup(ctx context.Context)

	// Resource operations
	PlanResource(ctx context.Context, provider *ProviderConfig, resourceType string, currentState *map[string]any, newConfig map[string]any) (map[string]any, error)
	CreateResource(ctx context.Context, provider *ProviderConfig, resourceType string, config map[string]any) (map[string]any, error)
	UpdateResource(ctx context.Context, provider *ProviderConfig, resourceType string, currentState, newConfig map[string]any) (map[string]any, error)
	DeleteResource(ctx context.Context, provider *ProviderConfig, resourceType string, state map[string]any) error
	RefreshResource(ctx context.Context, provider *ProviderConfig, resourceType string, currentState map[string]any) (map[string]any, error)
	ImportResource(ctx context.Context, provider *ProviderConfig, resourceType, importID string) (map[string]any, error)

	// Data source operations
	ReadDataSource(ctx context.Context, provider *ProviderConfig, dataSourceType string, config map[string]any) (map[string]any, error)

	// Configuration
	GetProviderVersion(ctx context.Context, providerName string) (string, error)
	GetProviderVersions(ctx context.Context, providerName string) ([]ProviderVersionInfo, error)
	DescribeProvider(ctx context.Context, providerConfig *ProviderConfig) (*ProviderSchema, *string, error)
	DescribeResource(ctx context.Context, provider *ProviderConfig, resourceType string) (*TypeDescription, error)
	DescribeDataSource(ctx context.Context, provider *ProviderConfig, dataSourceType string) (*TypeDescription, error)

	// deprecated
	ListResources(ctx context.Context, provider *ProviderConfig) ([]string, error)
	ListDataSources(ctx context.Context, provider *ProviderConfig) ([]string, error)
}

// RegistryClient interface defines operations for interacting with provider registry
type RegistryClient interface {
	Download(ctx context.Context, url string) (io.ReadCloser, error)
	GetProviderDownload(ctx context.Context, providerName string, version *string) (*DownloadInfo, error)
	GetProviderVersions(ctx context.Context, providerName string) ([]ProviderVersionInfo, error)
	SearchProviders(ctx context.Context, query string) ([]ProviderSearchResult, error)
	FindProvider(ctx context.Context, query string) (*ProviderSearchResult, error)
}

// Storage interface defines all storage operations
type Storage interface {
	// State operations
	ListStates(ctx context.Context) ([]StateRecord, error)
	SaveState(ctx context.Context, record StateRecord) error
	GetState(ctx context.Context, id string) (*StateRecord, error)
	DeleteState(ctx context.Context, id string) error
	UpdateState(ctx context.Context, record StateRecord) error

	// Dependency operations
	AddDependency(ctx context.Context, edge DependencyEdge) error
	RemoveDependency(ctx context.Context, fromID, toID string) error
	GetDependencies(ctx context.Context, resourceID string) ([]DependencyEdge, error)
	GetDependents(ctx context.Context, resourceID string) ([]DependencyEdge, error)

	// Timeline operations
	GetTimeline(ctx context.Context, query TimelineQuery) (*TimelineResponse, error)

	SaveResourceOperation(ctx context.Context, operation ResourceOperation) error
	ListResourceOperations(ctx context.Context, args ResourceOperationsArgs) ([]ResourceOperation, error)
	GetResourceOperation(ctx context.Context, resourceID string) (*ResourceOperation, error)

	Close() error
}
