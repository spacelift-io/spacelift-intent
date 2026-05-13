// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/spacelift-intent/allowlist"
	"github.com/spacelift-io/spacelift-intent/types"
)

// recordingRegistry fails on every network-touching method and records whether
// it was called. Used to assert that LoadProvider short-circuits before any
// registry interaction when the allowlist denies the request.
type recordingRegistry struct {
	called bool
}

func (r *recordingRegistry) SearchProviders(context.Context, string) ([]types.ProviderSearchResult, error) {
	r.called = true
	return nil, errors.New("registry should not be called")
}

func (r *recordingRegistry) FindProvider(context.Context, string) (*types.ProviderSearchResult, error) {
	r.called = true
	return nil, errors.New("registry should not be called")
}

func (r *recordingRegistry) GetProviderVersions(context.Context, types.ProviderConfig) ([]types.ProviderVersionInfo, error) {
	r.called = true
	return nil, errors.New("registry should not be called")
}

func (r *recordingRegistry) GetProviderDownload(context.Context, types.ProviderConfig) (*types.DownloadInfo, error) {
	r.called = true
	return nil, errors.New("registry should not be called")
}

func (r *recordingRegistry) Download(context.Context, string) (io.ReadCloser, error) {
	r.called = true
	return nil, errors.New("registry should not be called")
}

func TestLoadProvider_AllowlistDeniesBeforeRegistry(t *testing.T) {
	al, err := allowlist.LoadFromBytes([]byte(`
providers:
  - name: hashicorp/aws
`))
	require.NoError(t, err)

	reg := &recordingRegistry{}
	adapter := NewOpenTofuAdapter(t.TempDir(), reg, al)
	defer adapter.Cleanup(context.Background())

	err = adapter.LoadProvider(context.Background(), &types.ProviderConfig{
		Name:    "kuwas/github",
		Version: "4.3.0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowlist")
	assert.False(t, reg.called, "registry must not be contacted on allowlist denial")
}

func TestLoadProvider_AllowlistDeniesVersionBeforeRegistry(t *testing.T) {
	al, err := allowlist.LoadFromBytes([]byte(`
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
`))
	require.NoError(t, err)

	reg := &recordingRegistry{}
	adapter := NewOpenTofuAdapter(t.TempDir(), reg, al)
	defer adapter.Cleanup(context.Background())

	err = adapter.LoadProvider(context.Background(), &types.ProviderConfig{
		Name:    "hashicorp/aws",
		Version: "4.50.0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowlist")
	assert.False(t, reg.called, "registry must not be contacted on version denial")
}

func TestLoadProvider_NilAllowlistPermitsLookup(t *testing.T) {
	// With a nil allowlist, the gate is bypassed and the call proceeds to the
	// registry. We don't care about the eventual outcome (the recording registry
	// errors), only that the gate did not reject upfront.
	reg := &recordingRegistry{}
	adapter := NewOpenTofuAdapter(t.TempDir(), reg, nil)
	defer adapter.Cleanup(context.Background())

	err := adapter.LoadProvider(context.Background(), &types.ProviderConfig{
		Name:    "kuwas/github",
		Version: "4.3.0",
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "allowlist")
	assert.True(t, reg.called, "nil allowlist must allow lookup to reach the registry")
}
