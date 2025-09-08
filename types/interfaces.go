package types

import (
	"context"
	"io"
)

// ProviderManager interface defines provider management operations
type ProviderManager interface {
	// Provider lifecycle
	LoadProvider(ctx context.Context, providerName string) error
	GetProviderVersion(ctx context.Context, providerName string) (string, error)
	Cleanup(ctx context.Context)

	// Resource operations
	PlanResource(ctx context.Context, providerName, resourceType string, currentState *map[string]any, newConfig map[string]any) (map[string]any, error)
	CreateResource(ctx context.Context, providerName, resourceType string, config map[string]any) (map[string]any, error)
	UpdateResource(ctx context.Context, providerName, resourceType string, currentState, newConfig map[string]any) (map[string]any, error)
	DeleteResource(ctx context.Context, providerName, resourceType string, state map[string]any) error
	RefreshResource(ctx context.Context, providerName, resourceType string, currentState map[string]any) (map[string]any, error)
	ImportResource(ctx context.Context, providerName, resourceType, resourceID string) (map[string]any, error)
	ListResources(ctx context.Context, providerName string) ([]string, error)
	DescribeResource(ctx context.Context, providerName, resourceType string) (*TypeDescription, error)

	// Data source operations
	DescribeDataSource(ctx context.Context, providerName, dataSourceType string) (*TypeDescription, error)
	ReadDataSource(ctx context.Context, providerName, dataSourceType string, config map[string]any) (map[string]any, error)
	ListDataSources(ctx context.Context, providerName string) ([]string, error)
}

// RegistryClient interface defines operations for interacting with provider registry
type RegistryClient interface {
	Download(ctx context.Context, url string) (io.ReadCloser, error)
	GetProviderDownload(ctx context.Context, providerName string) (*DownloadInfo, error)
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

	// Policy operations
	SavePolicy(ctx context.Context, policy Policy) error
	GetPolicy(ctx context.Context, id string) (*Policy, error)
	ListPolicies(ctx context.Context) ([]Policy, error)
	DeletePolicy(ctx context.Context, id string) error

	SaveResourceOperation(ctx context.Context, operation ResourceOperation) error
	ListResourceOperations(ctx context.Context, args ResourceOperationsArgs) ([]ResourceOperation, error)
	GetResourceOperation(ctx context.Context, resourceID string) (*ResourceOperation, error)

	Close() error
}

// PolicyEngine interface defines policy evaluation operations
type PolicyEngine interface {
	EvaluatePolicies(ctx context.Context, policies []Policy, input PolicyInput) (PolicyResult, error)
	ValidatePolicy(ctx context.Context, regoCode string) error
}

// PolicyEvaluator interface defines policy coordination for resource operations
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, input PolicyInput) (*PolicyResult, error)
}
