// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" // Import SQLite driver for database/sql.

	"github.com/spacelift-io/spacelift-intent/types"
)

// SQLiteStorage implements types.Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite-based storage that implements all storage interfaces
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	return storage, nil
}

// SaveState stores a state record and automatically records history if context provided
func (s *SQLiteStorage) SaveState(ctx context.Context, record types.StateRecord) error {
	// Serialize state to JSON
	stateJSON, err := json.Marshal(record.State)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// Save the state
	query := `
	INSERT OR REPLACE INTO state_records (id, provider, provider_version, resource_type, state, created_at)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err = s.db.ExecContext(ctx, query, record.ResourceID, record.Provider, record.ProviderVersion, record.ResourceType, string(stateJSON))
	if err != nil {
		return err
	}

	// Automatically record timeline event if context provided
	if operation := ctx.Value(types.OperationContextKey); operation != nil {
		if changedBy := ctx.Value(types.ChangedByContextKey); changedBy != nil {
			event := types.TimelineEvent{
				ResourceID: record.ResourceID,
				Operation:  operation.(string),
				ChangedBy:  changedBy.(string),
				CreatedAt:  s.getCurrentTimestamp(),
			}

			s.addTimelineEvent(ctx, event) // Use internal method to avoid circular calls
		}
	}

	return nil
}

// GetState retrieves a state record by ID
func (s *SQLiteStorage) GetState(ctx context.Context, id string) (*types.StateRecord, error) {
	query := `
	SELECT id, provider, provider_version, resource_type, state, created_at
	FROM state_records
	WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)

	var record types.StateRecord
	var stateJSON string
	err := row.Scan(&record.ResourceID, &record.Provider, &record.ProviderVersion, &record.ResourceType, &stateJSON, &record.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Deserialize state from JSON
	if stateJSON != "" {
		err = json.Unmarshal([]byte(stateJSON), &record.State)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize state: %w", err)
		}
	}

	return &record, nil
}

// ListStates returns all state records
func (s *SQLiteStorage) ListStates(ctx context.Context) ([]types.StateRecord, error) {
	query := `
	SELECT id, provider, provider_version, resource_type, state, created_at
	FROM state_records
	ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []types.StateRecord
	for rows.Next() {
		var record types.StateRecord
		var stateJSON string
		err := rows.Scan(&record.ResourceID, &record.Provider, &record.ProviderVersion, &record.ResourceType, &stateJSON, &record.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Deserialize state from JSON
		if stateJSON != "" {
			err = json.Unmarshal([]byte(stateJSON), &record.State)
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize state: %w", err)
			}
		}

		records = append(records, record)
	}

	return records, rows.Err()
}

// UpdateState updates an existing state record
func (s *SQLiteStorage) UpdateState(ctx context.Context, record types.StateRecord) error {
	// This is essentially the same as SaveState since we use INSERT OR REPLACE
	return s.SaveState(ctx, record)
}

// DeleteState removes a state record by ID and automatically records history if context provided
func (s *SQLiteStorage) DeleteState(ctx context.Context, id string) error {
	// Delete the state
	query := `DELETE FROM state_records WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	// Automatically record timeline event if context provided
	if operation := ctx.Value(types.OperationContextKey); operation != nil {
		if changedBy := ctx.Value(types.ChangedByContextKey); changedBy != nil {
			event := types.TimelineEvent{
				ResourceID: id,
				Operation:  operation.(string),
				ChangedBy:  changedBy.(string),
				CreatedAt:  s.getCurrentTimestamp(),
			}

			s.addTimelineEvent(ctx, event) // Use internal method to avoid circular calls
		}
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// AddDependency adds a dependency edge, e.g.
// instance
//
//	↓
//
// instance-profile
//
//	↓
//
// ec2-role
// ---
// Dependencies:
// 1. from_resource_id = instance; to_resource_id = instance_profile
// 2. from_resource_id = instance_profile; to_resource_id = ec2-role
// ec2-role is a dependency for the instance_profile, and instance_profile is a dependent for ec2-role
func (s *SQLiteStorage) AddDependency(ctx context.Context, edge types.DependencyEdge) error {
	// Serialize field mappings to JSON
	fieldMappingsJSON, err := json.Marshal(edge.FieldMappings)
	if err != nil {
		return fmt.Errorf("failed to serialize field mappings: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO dependency_edges (from_resource_id, to_resource_id, dependency_type, explanation, field_mappings, created_at)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err = s.db.ExecContext(ctx, query, edge.FromResourceID, edge.ToResourceID, edge.DependencyType, edge.Explanation, string(fieldMappingsJSON))
	return err
}

// RemoveDependency removes a dependency edge
func (s *SQLiteStorage) RemoveDependency(ctx context.Context, fromID, toID string) error {
	query := `DELETE FROM dependency_edges WHERE from_resource_id = ? AND to_resource_id = ?`
	_, err := s.db.ExecContext(ctx, query, fromID, toID)
	return err
}

// GetDependencies returns all dependencies for a resource, meaning what this resource depends on (what needs to be created before this resource)
func (s *SQLiteStorage) GetDependencies(ctx context.Context, resourceID string) ([]types.DependencyEdge, error) {
	query := `
	SELECT from_resource_id, to_resource_id, dependency_type, explanation, field_mappings, created_at
	FROM dependency_edges
	WHERE from_resource_id = ?
	ORDER BY created_at
	`

	rows, err := s.db.QueryContext(ctx, query, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []types.DependencyEdge
	for rows.Next() {
		var edge types.DependencyEdge
		var fieldMappingsJSON string
		err := rows.Scan(&edge.FromResourceID, &edge.ToResourceID, &edge.DependencyType, &edge.Explanation, &fieldMappingsJSON, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Deserialize field mappings from JSON
		if fieldMappingsJSON != "" {
			err = json.Unmarshal([]byte(fieldMappingsJSON), &edge.FieldMappings)
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize field mappings: %w", err)
			}
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// GetDependents returns all dependents for a resource, meaning what depends on this resource
func (s *SQLiteStorage) GetDependents(ctx context.Context, resourceID string) ([]types.DependencyEdge, error) {
	query := `
	SELECT from_resource_id, to_resource_id, dependency_type, explanation, field_mappings, created_at
	FROM dependency_edges
	WHERE to_resource_id = ?
	ORDER BY created_at
	`

	rows, err := s.db.QueryContext(ctx, query, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []types.DependencyEdge
	for rows.Next() {
		var edge types.DependencyEdge
		var fieldMappingsJSON string
		err := rows.Scan(&edge.FromResourceID, &edge.ToResourceID, &edge.DependencyType, &edge.Explanation, &fieldMappingsJSON, &edge.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Deserialize field mappings from JSON
		if fieldMappingsJSON != "" {
			err = json.Unmarshal([]byte(fieldMappingsJSON), &edge.FieldMappings)
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize field mappings: %w", err)
			}
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// Timeline operations

// addTimelineEvent is an internal method to record timeline events (used by state operations)
func (s *SQLiteStorage) addTimelineEvent(ctx context.Context, event types.TimelineEvent) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate event ID: %w", err)
	}
	event.ID = id.String()

	query := `
	INSERT INTO timeline_events (id, resource_id, operation, changed_by, created_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err = s.db.ExecContext(ctx, query, event.ID, event.ResourceID, event.Operation, event.ChangedBy)
	return err
}

// GetTimeline returns timeline events based on query parameters with pagination
func (s *SQLiteStorage) GetTimeline(ctx context.Context, query types.TimelineQuery) (*types.TimelineResponse, error) {
	// Set defaults
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	// Build the base query
	baseQuery := `FROM timeline_events WHERE 1=1`
	var conditions []string
	var args []any

	// Add resource filter
	if query.ResourceID != "" {
		conditions = append(conditions, "resource_id = ?")
		args = append(args, query.ResourceID)
	}

	// Add time range filters
	if query.FromTime != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, query.FromTime)
	}
	if query.ToTime != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, query.ToTime)
	}

	// Add conditions to base query
	for _, condition := range conditions {
		baseQuery += " AND " + condition
	}

	// Get total count
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get the events
	eventsQuery := "SELECT id, resource_id, operation, changed_by, created_at " +
		baseQuery + " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	args = append(args, query.Limit, query.Offset)
	rows, err := s.db.QueryContext(ctx, eventsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline events: %w", err)
	}
	defer rows.Close()

	var events []types.TimelineEvent
	for rows.Next() {
		var event types.TimelineEvent
		err := rows.Scan(&event.ID, &event.ResourceID, &event.Operation, &event.ChangedBy, &event.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline event: %w", err)
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timeline events: %w", err)
	}

	// Determine if there are more events
	hasMore := query.Offset+len(events) < totalCount

	return &types.TimelineResponse{
		Events:     events,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// getCurrentTimestamp returns the current timestamp in RFC3339 format
func (s *SQLiteStorage) getCurrentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ResourceReference represents a reference found in configuration
type ResourceReference struct {
	ResourceID      string
	FieldPath       string
	ReferencedField string
}

func (s *SQLiteStorage) SaveResourceOperation(ctx context.Context, operation types.ResourceOperation) error {
	query := `
	INSERT INTO operations (id, resource_id, resource_type, provider, provider_version, operation, current_state, proposed_state, created_at, failed)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
	`

	currentState, err := json.Marshal(operation.CurrentState)
	if err != nil {
		return fmt.Errorf("failed to serialize current state: %w", err)
	}
	proposedState, err := json.Marshal(operation.ProposedState)
	if err != nil {
		return fmt.Errorf("failed to serialize proposed state: %w", err)
	}
	failed := sql.NullString{}
	if operation.Failed != nil {
		failed.String = *operation.Failed
		failed.Valid = true
	}

	_, err = s.db.ExecContext(ctx, query,
		operation.ID,
		operation.ResourceID,
		operation.ResourceType,
		operation.Provider,
		operation.ProviderVersion,
		operation.Operation,
		string(currentState),
		string(proposedState),
		failed,
	)
	return err
}

func (s *SQLiteStorage) ListResourceOperations(ctx context.Context, args types.ResourceOperationsArgs) ([]types.ResourceOperation, error) {
	query := `
	SELECT id, resource_id, resource_type, provider, provider_version, operation, current_state, proposed_state, created_at, failed
	FROM operations 
	WHERE 1=1
	`
	var vars []any

	if args.ResourceID != nil {
		query += " AND resource_id = ?"
		vars = append(vars, *args.ResourceID)
	}
	if args.ResourceType != nil {
		query += " AND resource_type = ?"
		vars = append(vars, *args.ResourceType)
	}
	if args.Provider != nil {
		query += " AND provider = ?"
		vars = append(vars, *args.Provider)
	}
	if args.ProviderVersion != nil {
		query += " AND provider_version = ?"
		vars = append(vars, *args.ProviderVersion)
	}
	query += " ORDER BY datetime(created_at) DESC"

	if args.Limit != nil {
		if *args.Limit < 0 {
			return nil, fmt.Errorf("limit must be non-negative")
		}
		query += " LIMIT ?"
		vars = append(vars, *args.Limit)
		if args.Offset > 0 {
			query += " OFFSET ?"
			vars = append(vars, args.Offset)
		}
	} else if args.Offset > 0 {
		query += " LIMIT -1 OFFSET ?"
		vars = append(vars, args.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, vars...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operations []types.ResourceOperation
	for rows.Next() {
		var operation types.ResourceOperation
		var currentStateJSON, proposedStateJSON string
		var failed sql.NullString

		err := rows.Scan(&operation.ID,
			&operation.ResourceID,
			&operation.ResourceType,
			&operation.Provider,
			&operation.ProviderVersion,
			&operation.Operation,
			&currentStateJSON,
			&proposedStateJSON,
			&operation.CreatedAt,
			&failed)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(currentStateJSON), &operation.CurrentState); err != nil {
			return nil, fmt.Errorf("failed to deserialize current state: %w", err)
		}
		if err = json.Unmarshal([]byte(proposedStateJSON), &operation.ProposedState); err != nil {
			return nil, fmt.Errorf("failed to deserialize proposed state: %w", err)
		}

		if failed.Valid && failed.String != "" {
			operation.Failed = &failed.String
		}

		operations = append(operations, operation)
	}

	return operations, rows.Err()
}

func (s *SQLiteStorage) GetResourceOperation(ctx context.Context, resourceID string) (*types.ResourceOperation, error) {
	query := `
	SELECT id, resource_id, resource_type, provider, provider_version, operation, current_state, proposed_state, created_at, failed
	FROM operations 
	WHERE resource_id = ?
	ORDER BY created_at DESC
	LIMIT 1
	`

	row := s.db.QueryRowContext(ctx, query, resourceID)

	var operation types.ResourceOperation
	var currentStateJSON, proposedStateJSON string
	var failed sql.NullString

	err := row.Scan(&operation.ID,
		&operation.ResourceID,
		&operation.ResourceType,
		&operation.Provider,
		&operation.ProviderVersion,
		&operation.Operation,
		&currentStateJSON,
		&proposedStateJSON,
		&operation.CreatedAt,
		&failed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err = json.Unmarshal([]byte(currentStateJSON), &operation.CurrentState); err != nil {
		return nil, fmt.Errorf("failed to deserialize current state: %w", err)
	}
	if err = json.Unmarshal([]byte(proposedStateJSON), &operation.ProposedState); err != nil {
		return nil, fmt.Errorf("failed to deserialize proposed state: %w", err)
	}

	if failed.Valid && failed.String != "" {
		operation.Failed = &failed.String
	}

	return &operation, nil
}
