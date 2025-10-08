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

package main

import (
	"fmt"
	"os"
)

const DefaultDirPermissions = 0755

// ensureDir creates a directory with standard permissions if it doesn't exist
func ensureDir(path string) error {
	return ensureDirWithPermissions(path, DefaultDirPermissions)
}

// ensureDirWithPermissions creates a directory with specified permissions if it doesn't exist
func ensureDirWithPermissions(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// ensureDirs creates multiple directories with standard permissions
func ensureDirs(paths ...string) error {
	for _, path := range paths {
		if err := ensureDir(path); err != nil {
			return err
		}
	}
	return nil
}
