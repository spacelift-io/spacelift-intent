// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/allowlist"
	"github.com/spacelift-io/spacelift-intent/types"
)

type fakeRegistry struct {
	searchResults []types.ProviderSearchResult
	versions      []types.ProviderVersionInfo
	download      *types.DownloadInfo

	searchCalls   int
	downloadCalls int
	versionsCalls int
}

func (f *fakeRegistry) SearchProviders(_ context.Context, _ string) ([]types.ProviderSearchResult, error) {
	f.searchCalls++
	return f.searchResults, nil
}

func (f *fakeRegistry) FindProvider(_ context.Context, _ string) (*types.ProviderSearchResult, error) {
	if len(f.searchResults) == 0 {
		return nil, nil
	}
	return &f.searchResults[0], nil
}

func (f *fakeRegistry) GetProviderVersions(_ context.Context, _ types.ProviderConfig) ([]types.ProviderVersionInfo, error) {
	f.versionsCalls++
	return f.versions, nil
}

func (f *fakeRegistry) GetProviderDownload(_ context.Context, _ types.ProviderConfig) (*types.DownloadInfo, error) {
	f.downloadCalls++
	return f.download, nil
}

func (f *fakeRegistry) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, nil
}

func TestAllowlistedClient_PassthroughWhenDisabled(t *testing.T) {
	inner := &fakeRegistry{}
	wrapped := NewAllowlistedClient(inner, allowlist.Disabled())
	assert.Same(t, inner, wrapped, "disabled allowlist must not wrap the inner client")
}

func TestAllowlistedClient_FiltersSearchResults(t *testing.T) {
	inner := &fakeRegistry{
		searchResults: []types.ProviderSearchResult{
			{Addr: "hashicorp/aws", Popularity: 100},
			{Addr: "kuwas/github", Popularity: 50},
			{Addr: "hashicorp/random", Popularity: 80},
		},
	}
	al := loadAllowlist(t, `
providers:
  - name: hashicorp/*
`)
	client := NewAllowlistedClient(inner, al)

	results, err := client.SearchProviders(context.Background(), "anything")
	require.NoError(t, err)
	addrs := []string{}
	for _, r := range results {
		addrs = append(addrs, r.Addr)
	}
	assert.ElementsMatch(t, []string{"hashicorp/aws", "hashicorp/random"}, addrs)
}

func TestAllowlistedClient_FindProvider_PicksMostPopularAllowed(t *testing.T) {
	inner := &fakeRegistry{
		searchResults: []types.ProviderSearchResult{
			{Addr: "kuwas/github", Popularity: 99},
			{Addr: "hashicorp/aws", Popularity: 80},
			{Addr: "hashicorp/random", Popularity: 70},
		},
	}
	al := loadAllowlist(t, `
providers:
  - name: hashicorp/*
`)
	client := NewAllowlistedClient(inner, al)

	best, err := client.FindProvider(context.Background(), "anything")
	require.NoError(t, err)
	require.NotNil(t, best)
	assert.Equal(t, "hashicorp/aws", best.Addr)
}

func TestAllowlistedClient_FindProvider_NilWhenAllDenied(t *testing.T) {
	inner := &fakeRegistry{
		searchResults: []types.ProviderSearchResult{
			{Addr: "kuwas/github", Popularity: 99},
		},
	}
	al := loadAllowlist(t, `
providers:
  - name: hashicorp/*
`)
	client := NewAllowlistedClient(inner, al)

	best, err := client.FindProvider(context.Background(), "anything")
	require.NoError(t, err)
	assert.Nil(t, best)
}

func TestAllowlistedClient_GetProviderVersions_RejectsDeniedName(t *testing.T) {
	inner := &fakeRegistry{}
	al := loadAllowlist(t, `
providers:
  - name: hashicorp/*
`)
	client := NewAllowlistedClient(inner, al)

	_, err := client.GetProviderVersions(context.Background(), types.ProviderConfig{Name: "kuwas/github"})
	assert.Error(t, err)
	assert.Equal(t, 0, inner.versionsCalls, "must not call inner registry on denied name")
}

func TestAllowlistedClient_GetProviderDownload_RejectsBeforeNetwork(t *testing.T) {
	inner := &fakeRegistry{}
	al := loadAllowlist(t, `
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
`)
	client := NewAllowlistedClient(inner, al)

	_, err := client.GetProviderDownload(context.Background(), types.ProviderConfig{
		Name:    "hashicorp/aws",
		Version: "4.50.0",
	})
	assert.Error(t, err)
	assert.Equal(t, 0, inner.downloadCalls, "must not call inner registry on denied version")
}

func loadAllowlist(t *testing.T, src string) *allowlist.Allowlist {
	t.Helper()
	al, err := allowlist.LoadFromBytes([]byte(src))
	require.NoError(t, err)
	return al
}
