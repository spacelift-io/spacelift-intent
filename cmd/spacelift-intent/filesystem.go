// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

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
