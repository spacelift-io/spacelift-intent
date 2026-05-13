// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"fmt"
	"io"

	"github.com/spacelift-io/spacelift-intent/allowlist"
	"github.com/spacelift-io/spacelift-intent/types"
)

// NewAllowlistedClient wraps a RegistryClient so that search results are
// filtered against the allowlist. This is the UX layer — disallowed providers
// never surface to the model. The hard gate lives in the provider manager;
// this decorator does not substitute for that.
//
// When the allowlist is disabled, the inner client is returned unchanged so
// there is zero overhead in the default configuration.
func NewAllowlistedClient(inner types.RegistryClient, al *allowlist.Allowlist) types.RegistryClient {
	if !al.Enabled() {
		return inner
	}
	return &allowlistedClient{inner: inner, allowlist: al}
}

type allowlistedClient struct {
	inner     types.RegistryClient
	allowlist *allowlist.Allowlist
}

func (c *allowlistedClient) SearchProviders(ctx context.Context, query string) ([]types.ProviderSearchResult, error) {
	results, err := c.inner.SearchProviders(ctx, query)
	if err != nil {
		return nil, err
	}
	filtered := make([]types.ProviderSearchResult, 0, len(results))
	for _, r := range results {
		if c.allowlist.AllowsName(r.Addr) == nil {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

func (c *allowlistedClient) FindProvider(ctx context.Context, query string) (*types.ProviderSearchResult, error) {
	results, err := c.inner.SearchProviders(ctx, query)
	if err != nil {
		return nil, err
	}
	var best *types.ProviderSearchResult
	for i := range results {
		if c.allowlist.AllowsName(results[i].Addr) != nil {
			continue
		}
		if best == nil || results[i].Popularity > best.Popularity {
			best = &results[i]
		}
	}
	return best, nil
}

func (c *allowlistedClient) GetProviderVersions(ctx context.Context, provider types.ProviderConfig) ([]types.ProviderVersionInfo, error) {
	if err := c.allowlist.AllowsName(provider.Name); err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}
	return c.inner.GetProviderVersions(ctx, provider)
}

func (c *allowlistedClient) GetProviderDownload(ctx context.Context, provider types.ProviderConfig) (*types.DownloadInfo, error) {
	if err := c.allowlist.Allows(&provider); err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}
	return c.inner.GetProviderDownload(ctx, provider)
}

func (c *allowlistedClient) Download(ctx context.Context, url string) (io.ReadCloser, error) {
	return c.inner.Download(ctx, url)
}
