// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

//go:build legacy_plugin

package provider

import (
	"github.com/spacelift-io/spacelift-intent/types"
)

// NewOpenTofuAdapter creates a new adapter using the opentofu-providers library
func NewOpenTofuAdapter(tmpDir string, registry types.RegistryClient) types.ProviderManager {
	return nil
}
