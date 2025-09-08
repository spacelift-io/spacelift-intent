// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/spacelift-io/spacelift-intent/types"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // Import SQLite driver for database/sql.
)

func newTestSQLiteStorage(t *testing.T) (*sqliteStorage, context.Context) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "sqlite.db")
	store, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.db.Close())
	})
	return store, ctx
}

func TestSQLiteGetTimelinePagination(t *testing.T) {
	t.Parallel()

	store, ctx := newTestSQLiteStorage(t)

	baseTime := time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		createdAt := baseTime.Add(time.Duration(i) * time.Minute).Format(time.RFC3339)
		_, err := store.db.ExecContext(ctx, `
			INSERT INTO timeline_events (id, resource_id, operation, changed_by, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, fmt.Sprintf("event-%d", i), fmt.Sprintf("resource-%d", i%2), "create", "tester", createdAt)
		require.NoError(t, err)
	}

	testCases := []struct {
		name          string
		query         types.TimelineQuery
		expectedIDs   []string
		expectedCount int
		expectedMore  bool
	}{
		{
			name: "first page returns newest events and indicates more",
			query: types.TimelineQuery{
				Limit:  2,
				Offset: 0,
			},
			expectedIDs:   []string{"event-4", "event-3"},
			expectedCount: 5,
			expectedMore:  true,
		},
		{
			name: "last partial page returns remaining events",
			query: types.TimelineQuery{
				Limit:  2,
				Offset: 4,
			},
			expectedIDs:   []string{"event-0"},
			expectedCount: 5,
			expectedMore:  false,
		},
		{
			name: "offset beyond total returns empty slice",
			query: types.TimelineQuery{
				Limit:  3,
				Offset: 5,
			},
			expectedIDs:   []string{},
			expectedCount: 5,
			expectedMore:  false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := store.GetTimeline(ctx, tc.query)
			require.NoError(t, err)
			require.Equal(t, tc.expectedCount, res.TotalCount)
			require.Equal(t, tc.expectedMore, res.HasMore)

			gotIDs := make([]string, len(res.Events))
			for i, event := range res.Events {
				gotIDs[i] = event.ID
			}
			require.Equal(t, tc.expectedIDs, gotIDs)
		})
	}
}

func TestSQLiteListResourceOperations(t *testing.T) {
	t.Parallel()

	store, ctx := newTestSQLiteStorage(t)

	baseTime := time.Date(2025, time.January, 2, 8, 0, 0, 0, time.UTC)
	failedMsg := "plan failed"

	records := []struct {
		op        types.ResourceOperation
		createdAt time.Time
	}{
		{
			op: types.ResourceOperation{
				ID: "op-0",
				ResourceOperationInput: types.ResourceOperationInput{
					ResourceID:    "res-1",
					ResourceType:  "aws_instance",
					Provider:      "aws",
					Operation:     "create",
					CurrentState:  map[string]any{"status": "current-0"},
					ProposedState: map[string]any{"status": "next-0"},
				},
				ResourceOperationResult: types.ResourceOperationResult{},
			},
			createdAt: baseTime,
		},
		{
			op: types.ResourceOperation{
				ID: "op-1",
				ResourceOperationInput: types.ResourceOperationInput{
					ResourceID:    "res-1",
					ResourceType:  "aws_instance",
					Provider:      "aws",
					Operation:     "update",
					CurrentState:  map[string]any{"status": "current-1"},
					ProposedState: map[string]any{"status": "next-1"},
				},
				ResourceOperationResult: types.ResourceOperationResult{},
			},
			createdAt: baseTime.Add(1 * time.Minute),
		},
		{
			op: types.ResourceOperation{
				ID: "op-2",
				ResourceOperationInput: types.ResourceOperationInput{
					ResourceID:    "res-2",
					ResourceType:  "aws_s3_bucket",
					Provider:      "aws",
					Operation:     "delete",
					CurrentState:  map[string]any{"status": "current-2"},
					ProposedState: map[string]any{"status": "next-2"},
				},
				ResourceOperationResult: types.ResourceOperationResult{Failed: &failedMsg},
			},
			createdAt: baseTime.Add(2 * time.Minute),
		},
		{
			op: types.ResourceOperation{
				ID: "op-3",
				ResourceOperationInput: types.ResourceOperationInput{
					ResourceID:    "res-3",
					ResourceType:  "gcp_instance",
					Provider:      "gcp",
					Operation:     "create",
					CurrentState:  map[string]any{"status": "current-3"},
					ProposedState: map[string]any{"status": "next-3"},
				},
				ResourceOperationResult: types.ResourceOperationResult{},
			},
			createdAt: baseTime.Add(3 * time.Minute),
		},
	}

	for _, record := range records {
		require.NoError(t, store.SaveResourceOperation(ctx, record.op))
		_, err := store.db.ExecContext(ctx, `UPDATE operations SET created_at = ? WHERE id = ?`, record.createdAt.Format(time.RFC3339), record.op.ID)
		require.NoError(t, err)
	}

	collectIDs := func(ops []types.ResourceOperation) []string {
		ids := make([]string, len(ops))
		for i, op := range ops {
			ids[i] = op.ID
		}
		return ids
	}

	t.Run("orders by created_at desc and hydrates state", func(t *testing.T) {
		opList, err := store.ListResourceOperations(ctx, types.ResourceOperationsArgs{})
		require.NoError(t, err)
		require.Len(t, opList, len(records))
		require.Equal(t, []string{"op-3", "op-2", "op-1", "op-0"}, collectIDs(opList))
		require.Equal(t, "current-3", opList[0].CurrentState["status"])
		require.Equal(t, "next-3", opList[0].ProposedState["status"])
		require.Nil(t, opList[0].Failed)
		require.NotNil(t, opList[1].Failed)
		require.Equal(t, failedMsg, *opList[1].Failed)
	})

	t.Run("applies limit offset and filters by resource id", func(t *testing.T) {
		resourceID := "res-1"
		limit := 1
		args := types.ResourceOperationsArgs{
			ResourceID: &resourceID,
			Limit:      &limit,
		}
		opList, err := store.ListResourceOperations(ctx, args)
		require.NoError(t, err)
		require.Equal(t, []string{"op-1"}, collectIDs(opList))

		args.Offset = 1
		opList, err = store.ListResourceOperations(ctx, args)
		require.NoError(t, err)
		require.Equal(t, []string{"op-0"}, collectIDs(opList))
	})

	t.Run("offset without limit works with provider filter", func(t *testing.T) {
		provider := "aws"
		args := types.ResourceOperationsArgs{
			Provider: &provider,
			Offset:   2,
		}
		opList, err := store.ListResourceOperations(ctx, args)
		require.NoError(t, err)
		require.Equal(t, []string{"op-0"}, collectIDs(opList))
	})

	t.Run("negative limit returns error", func(t *testing.T) {
		limit := -1
		_, err := store.ListResourceOperations(ctx, types.ResourceOperationsArgs{Limit: &limit})
		require.Error(t, err)
		require.Contains(t, err.Error(), "limit must be non-negative")
	})
}
