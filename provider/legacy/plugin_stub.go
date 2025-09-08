// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !legacy_plugin

package legacy

import (
	"fmt"

	pb "github.com/apparentlymart/opentofu-providers/tofuprovider/grpc/tfplugin5"
)

// STUB IMPLEMENTATION - This file is compiled by default to avoid plugin.Empty conflicts
// The hashicorp/go-plugin dependency is hidden behind the legacy_plugin build tag
// This implementation is never used in practice - it's just a compilation stub

// providerInfo holds internal provider information using only opentofu-providers
type providerInfo struct {
	// Remove plugin.Client dependency - use only opentofu-providers
	provider pb.ProviderClient
	schema   *pb.GetProviderSchema_Response
	binary   string
	version  string
}

// Kill cleans up the plugin client (stub - never called)
func (p *providerInfo) Kill() {
	// STUB: This is never called since startProviderPlugin always returns an error
}

// STUB: Default implementation that always fails (avoids plugin.Empty conflict)
// This ensures the code compiles without hashicorp/go-plugin dependency
func startProviderPlugin(binary, providerName string) (*providerInfo, error) {
	// STUB: This implementation is never used - always returns error
	// Real implementation should use opentofu-providers library directly
	return nil, fmt.Errorf("provider functionality disabled - compile with -tags legacy_plugin")
}

// STUB: Cleanup function (never called)
func cleanupPlugins() {
	// STUB: No cleanup needed for this stub implementation
}
