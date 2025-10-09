// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// migration represents a single database migration
type migration struct {
	name string
	fn   func(context.Context, *sql.Tx) error
}

// allMigrations contains all migrations in order
var allMigrations = []migration{
	{
		name: "rename_state_records_version_to_provider_version",
		fn:   renameStateRecordsVersionColumn,
	},
	{
		name: "add_operations_provider_version",
		fn:   addOperationsProviderVersion,
	},
}

// migrate runs all database migrations in an idempotent manner
func (s *sqliteStorage) migrate(ctx context.Context) error {
	for _, m := range allMigrations {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", m.name, err)
		}

		if err := m.fn(ctx, tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", m.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.name, err)
		}
	}

	return nil
}

// renameStateRecordsVersionColumn renames version column to provider_version in state_records table
func renameStateRecordsVersionColumn(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "ALTER TABLE state_records RENAME COLUMN version TO provider_version")
	return err
}

// addOperationsProviderVersion adds provider_version column to operations table
func addOperationsProviderVersion(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "ALTER TABLE operations ADD COLUMN provider_version TEXT NOT NULL")
	return err
}
