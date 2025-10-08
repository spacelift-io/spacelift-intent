// Copyright 2025 Spacelift, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
