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

//go:build legacy_plugin

package provider

import (
	"github.com/spacelift-io/spacelift-intent/types"
)

// NewOpenTofuAdapter creates a new adapter using the opentofu-providers library
func NewOpenTofuAdapter(tmpDir string, registry types.RegistryClient) types.ProviderManager {
	return nil
}
