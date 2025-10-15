// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"strings"
)

type contextKey string

const (
	OperationContextKey contextKey = "operation"
	ChangedByContextKey contextKey = "changed_by"
)

// DownloadInfo contains provider download information
type DownloadInfo struct {
	DownloadURL string
	Shasum      string
	Version     string
}

// TypeDescription contains provider type information
type TypeDescription struct {
	ProviderName string         `json:"provider"`
	Type         string         `json:"type"`
	Description  string         `json:"description"`
	Properties   map[string]any `json:"properties"`
	Required     []string       `json:"required"`
}

// ProviderSearchResult represents a provider search result
type ProviderSearchResult struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Addr        string  `json:"addr"`
	Version     string  `json:"version"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Popularity  float64 `json:"popularity"`
}

// ProviderSearchToolResult represents the result returned by the provider-search tool
type ProviderSearchToolResult struct {
	Query    string               `json:"query"`
	Provider ProviderSearchResult `json:"provider"`
}

// StateRecord represents a stored resource state
type StateRecord struct {
	ResourceID      string         `json:"resource_id"`
	Provider        string         `json:"provider"`
	ProviderVersion string         `json:"provider_version"`
	ResourceType    string         `json:"resource_type"`
	State           map[string]any `json:"state"`
	CreatedAt       string         `json:"created_at"`
}

func (r StateRecord) GetProvider() *ProviderConfig {
	return &ProviderConfig{
		Name:    r.Provider,
		Version: r.ProviderVersion,
	}
}

// FieldMapping represents which fields impact which in a dependency
type FieldMapping struct {
	SourceField string `json:"source_field"` // Field in the dependent resource
	TargetField string `json:"target_field"` // Field in the dependency target
	Description string `json:"description"`  // How this field dependency works
}

// DependencyEdge represents a dependency relationship between resources
type DependencyEdge struct {
	FromResourceID string         `json:"from_resource_id"`
	ToResourceID   string         `json:"to_resource_id"`
	DependencyType string         `json:"dependency_type"` // "explicit", "implicit", "data_source"
	Explanation    string         `json:"explanation"`     // Why this dependency exists
	FieldMappings  []FieldMapping `json:"field_mappings"`  // Which fields impact which
	CreatedAt      string         `json:"created_at"`
}

// TimelineEvent represents a single event in the system timeline
type TimelineEvent struct {
	ID         string `json:"id"`
	ResourceID string `json:"resource_id,omitempty"` // Empty for global events
	Operation  string `json:"operation"`             // "create", "update", "delete", "import", "eject", "refresh"
	ChangedBy  string `json:"changed_by"`            // Who/what triggered the change
	CreatedAt  string `json:"created_at"`
}

// TimelineQuery represents query parameters for timeline requests
type TimelineQuery struct {
	ResourceID string `json:"resource_id,omitempty"` // If empty, get global timeline
	FromTime   string `json:"from_time,omitempty"`   // RFC3339 timestamp
	ToTime     string `json:"to_time,omitempty"`     // RFC3339 timestamp
	Limit      int    `json:"limit,omitempty"`       // Max events to return (default: 50)
	Offset     int    `json:"offset,omitempty"`      // Skip N events (for pagination)
}

// TimelineResponse represents a paginated timeline response
type TimelineResponse struct {
	Events     []TimelineEvent `json:"events"`
	TotalCount int             `json:"total_count"` // Total events matching query
	HasMore    bool            `json:"has_more"`    // True if there are more events beyond this page
}

// ResourceOperation represents a single operation on a resource
type ResourceOperation struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`

	ResourceOperationInput  // Input data for the resource operation
	ResourceOperationResult // Result data for the resource operation
}

// ResourceOperationInput represents the input data for a resource operation
type ResourceOperationInput struct {
	ResourceID      string         `json:"resource_id"`
	ResourceType    string         `json:"resource_type"`
	Provider        string         `json:"provider"`
	ProviderVersion string         `json:"provider_version"`
	Operation       string         `json:"operation"` // "create", "update", "delete", "import", "refresh"
	CurrentState    map[string]any `json:"current_state,omitempty"`
	ProposedState   map[string]any `json:"proposed_state,omitempty"`
}

type ResourceOperationResult struct {
	Failed *string `json:"failed,omitempty"` // Error message if operation failed
}

type ResourceOperationsArgs struct {
	ResourceID      *string `json:"resource_id"`
	ResourceType    *string `json:"resource_type"`
	Provider        *string `json:"provider"`
	ProviderVersion *string `json:"provider_version"`
	Limit           *int    `json:"limit"`
	Offset          int     `json:"offset"`
}

// ProviderSchema represents the schema information for a provider
type ProviderSchema struct {
	Provider    *TypeDescription
	Resources   map[string]*TypeDescription
	DataSources map[string]*TypeDescription
	Version     string
}

type ProviderConfig struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Config  map[string]any `json:"config,omitempty"`
}

// FullName returns a unique cache key for this provider configuration
// Both namespaced Name and Version must be non-empty strings
func (p *ProviderConfig) FullName() (string, error) {
	if p.Name == "" {
		return "", fmt.Errorf("empty provider name")
	}

	if p.Version == "" {
		return "", fmt.Errorf("provider %s is missing version", p.Name)
	}

	return p.Name + "@" + p.Version, nil
}

// NamespacedName parses and validates that the provider name is properly namespaced
// Expected format: "namespace/type" (e.g., "hashicorp/aws")
// Returns namespace and type, or an error if the format is invalid
func (p *ProviderConfig) NamespacedName() (string, string, error) {
	if p.Name == "" {
		return "", "", fmt.Errorf("empty provider name")
	}

	parts := strings.Split(p.Name, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid provider name format '%s', expected 'namespace/type'", p.Name)
	}

	return parts[0], parts[1], nil
}

// Parse parses and validates the complete provider configuration
// Returns namespace, type, and version, or an error if any component is invalid
func (p *ProviderConfig) Parse() (string, string, string, error) {
	namespace, name, err := p.NamespacedName()
	if err != nil {
		return "", "", "", err
	}

	if p.Version == "" {
		return "", "", "", fmt.Errorf("empty provider version")
	}

	return namespace, name, p.Version, nil
}

// ProviderVersionInfo represents provider version information from registry
type ProviderVersionInfo struct {
	Version   string             `json:"version"`
	Protocols []string           `json:"protocols"`
	Platforms []ProviderPlatform `json:"platforms"`
}

// ProviderPlatform represents a supported platform
type ProviderPlatform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}
