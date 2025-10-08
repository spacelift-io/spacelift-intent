// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/spacelift-io/spacelift-intent/types"
)

const (
	registryURL      = "https://registry.opentofu.org"
	apiURL           = "https://api.opentofu.org"
	searchTemplate   = "/registry/docs/search?q=%s"
	downloadTemplate = "/v1/providers/%s/%s/%s/download/%s/%s"
	versionsTemplate = "/v1/providers/%s/%s/versions"
	userAgent        = "spacelift-intent"
)

// openTofuClient implements Client for OpenTofu registry
type openTofuClient struct {
	client *http.Client

	searchURLTemplate   string
	downloadURLTemplate string
	versionsURLTemplate string
}

// NewOpenTofuClient creates a new OpenTofu registry client.
// The client can be configured using environment variables:
//   - OPENTOFU_REGISTRY_URL: Override the default registry URL (default: https://registry.opentofu.org)
//   - OPENTOFU_API_URL: Override the default API URL (default: https://api.opentofu.org)
func NewOpenTofuClient() types.RegistryClient {

	regURL := registryURL
	apiBaseURL := apiURL

	if envRegistryURL := os.Getenv("OPENTOFU_REGISTRY_URL"); envRegistryURL != "" {
		regURL = envRegistryURL
	}

	if envAPIURL := os.Getenv("OPENTOFU_API_URL"); envAPIURL != "" {
		apiBaseURL = envAPIURL
	}

	return &openTofuClient{
		client:              &http.Client{},
		searchURLTemplate:   apiBaseURL + searchTemplate,
		downloadURLTemplate: regURL + downloadTemplate,
		versionsURLTemplate: regURL + versionsTemplate,
	}
}

// SearchProviders searches for providers in the registry
func (c *openTofuClient) SearchProviders(ctx context.Context, query string) ([]types.ProviderSearchResult, error) {
	searchURL := fmt.Sprintf(c.searchURLTemplate, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search providers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	var results []types.ProviderSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Filter for providers only
	var providers []types.ProviderSearchResult
	for _, result := range results {
		if result.Type == "provider" {
			providers = append(providers, result)
		}
	}

	return providers, nil
}

// FindProvider finds the most popular provider for a given query
func (c *openTofuClient) FindProvider(ctx context.Context, query string) (*types.ProviderSearchResult, error) {
	// TODO: Implement more sophisticated provider ranking beyond just popularity
	// TODO: Add caching for provider search results to improve performance
	// TODO: Consider fuzzy matching for provider names
	results, err := c.SearchProviders(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search providers: %w", err)
	}

	// If no results, return nil
	if len(results) == 0 {
		return nil, nil
	}

	// Find the most popular provider
	var maxPopularity float64 = -1
	bestProvider := &types.ProviderSearchResult{}
	for _, result := range results {
		if result.Popularity > maxPopularity {
			maxPopularity = result.Popularity
			bestProvider = &result
		}
	}

	return bestProvider, nil
}

func (c *openTofuClient) GetProviderVersions(ctx context.Context, providerName string) ([]types.ProviderVersionInfo, error) {
	// Parse provider name
	namespace, providerType, err := parseProviderName(providerName)
	if err != nil {
		return nil, err
	}

	// Get available versions
	versions, err := c.getProviderVersions(ctx, namespace, providerType)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

// GetProviderDownload gets download information for a provider
func (c *openTofuClient) GetProviderDownload(ctx context.Context, providerName string, version *string) (*types.DownloadInfo, error) {
	// Parse provider name
	namespace, providerType, err := parseProviderName(providerName)
	if err != nil {
		return nil, err
	}

	// Get available versions
	versions, err := c.getProviderVersions(ctx, namespace, providerType)
	if err != nil {
		return nil, err
	}

	// Find compatible version
	selectedVersion, err := selectCompatibleVersion(versions, version)
	if err != nil {
		return nil, err
	}

	// Get download URL
	downloadURL := fmt.Sprintf(c.downloadURLTemplate, namespace, providerType, selectedVersion, runtime.GOOS, runtime.GOARCH)

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get download info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download not available for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	var download downloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&download); err != nil {
		return nil, fmt.Errorf("failed to decode download response: %w", err)
	}

	return &types.DownloadInfo{
		DownloadURL: download.DownloadURL,
		Shasum:      download.Shasum,
		Version:     selectedVersion,
	}, nil
}

// Download downloads a file from the given URL
func (c *openTofuClient) Download(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// getProviderVersions gets available versions for a provider
func (c *openTofuClient) getProviderVersions(ctx context.Context, namespace, providerType string) ([]types.ProviderVersionInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(c.versionsURLTemplate, namespace, providerType), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create versions request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("provider not found in registry")
	}

	var versions versionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to decode versions response: %w", err)
	}

	if len(versions.Versions) == 0 {
		return nil, fmt.Errorf("no versions available")
	}

	return versions.Versions, nil
}

// selectCompatibleVersion selects the latest version that supports protocol 5
// TODO: support multiple protocols in the future
func selectCompatibleVersion(versions []types.ProviderVersionInfo, version *string) (string, error) {
	for _, v := range versions {
		for _, protocol := range v.Protocols {
			if protocol == "5.0" {
				return v.Version, nil
			}
		}
		if version != nil && *version == v.Version {
			return v.Version, nil
		}
	}
	return "", fmt.Errorf("no compatible version found")
}

// parseProviderName parses a provider name into namespace and type
func parseProviderName(providerName string) (namespace, providerType string, err error) {
	if len(providerName) == 0 {
		return "", "", fmt.Errorf("empty provider name")
	}

	parts := strings.Split(providerName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid provider name format, expected 'namespace/type'")
	}

	return parts[0], parts[1], nil
}

// versionInfo represents provider version information from registry
type versionInfo struct {
	Version   string     `json:"version"`
	Protocols []string   `json:"protocols"`
	Platforms []platform `json:"platforms"`
}

// platform represents a supported platform
type platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// versionsResponse represents the registry response for available versions
type versionsResponse struct {
	Versions []types.ProviderVersionInfo `json:"versions"`
}

// downloadResponse represents the registry response for download information
type downloadResponse struct {
	DownloadURL string `json:"download_url"`
	Shasum      string `json:"shasum"`
}
