package filesystem

import (
	"fmt"
	"os"
)

const DefaultDirPermissions = 0755

// EnsureDir creates a directory with standard permissions if it doesn't exist
func EnsureDir(path string) error {
	return EnsureDirWithPermissions(path, DefaultDirPermissions)
}

// EnsureDirWithPermissions creates a directory with specified permissions if it doesn't exist
func EnsureDirWithPermissions(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// EnsureDirs creates multiple directories with standard permissions
func EnsureDirs(paths ...string) error {
	for _, path := range paths {
		if err := EnsureDir(path); err != nil {
			return err
		}
	}
	return nil
}