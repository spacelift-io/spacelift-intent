// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestAnnotationHintConstants(t *testing.T) {
	// Test that constants have expected bit flag values
	expectedValues := map[AnnotationHint]uint8{
		Readonly:    1, // 1 << 0 = 1
		Destructive: 2, // 1 << 1 = 2
		Idempotent:  4, // 1 << 2 = 4
		OpenWorld:   8, // 1 << 3 = 8
	}

	for hint, expected := range expectedValues {
		if uint8(hint) != expected {
			t.Errorf("Expected %v to have value %d, got %d", hint, expected, uint8(hint))
		}
	}
}

func TestToolAnnotations(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		hints          AnnotationHint
		expectedResult mcp.ToolAnnotation
	}{
		{
			name:  "no hints",
			title: "Test Tool",
			hints: 0,
			expectedResult: mcp.ToolAnnotation{
				Title:           "Test Tool",
				ReadOnlyHint:    mcp.ToBoolPtr(false),
				DestructiveHint: mcp.ToBoolPtr(false),
				IdempotentHint:  mcp.ToBoolPtr(false),
				OpenWorldHint:   mcp.ToBoolPtr(false),
			},
		},
		{
			name:  "readonly hint",
			title: "Read Tool",
			hints: Readonly,
			expectedResult: mcp.ToolAnnotation{
				Title:           "Read Tool",
				ReadOnlyHint:    mcp.ToBoolPtr(true),
				DestructiveHint: mcp.ToBoolPtr(false),
				IdempotentHint:  mcp.ToBoolPtr(false),
				OpenWorldHint:   mcp.ToBoolPtr(false),
			},
		},
		{
			name:  "destructive hint",
			title: "Delete Tool",
			hints: Destructive,
			expectedResult: mcp.ToolAnnotation{
				Title:           "Delete Tool",
				ReadOnlyHint:    mcp.ToBoolPtr(false),
				DestructiveHint: mcp.ToBoolPtr(true),
				IdempotentHint:  mcp.ToBoolPtr(false),
				OpenWorldHint:   mcp.ToBoolPtr(false),
			},
		},
		{
			name:  "idempotent hint",
			title: "Create Tool",
			hints: Idempotent,
			expectedResult: mcp.ToolAnnotation{
				Title:           "Create Tool",
				ReadOnlyHint:    mcp.ToBoolPtr(false),
				DestructiveHint: mcp.ToBoolPtr(false),
				IdempotentHint:  mcp.ToBoolPtr(true),
				OpenWorldHint:   mcp.ToBoolPtr(false),
			},
		},
		{
			name:  "open world hint",
			title: "Search Tool",
			hints: OpenWorld,
			expectedResult: mcp.ToolAnnotation{
				Title:           "Search Tool",
				ReadOnlyHint:    mcp.ToBoolPtr(false),
				DestructiveHint: mcp.ToBoolPtr(false),
				IdempotentHint:  mcp.ToBoolPtr(false),
				OpenWorldHint:   mcp.ToBoolPtr(true),
			},
		},
		{
			name:  "multiple hints combined",
			title: "Multi Tool",
			hints: Readonly | Idempotent,
			expectedResult: mcp.ToolAnnotation{
				Title:           "Multi Tool",
				ReadOnlyHint:    mcp.ToBoolPtr(true),
				DestructiveHint: mcp.ToBoolPtr(false),
				IdempotentHint:  mcp.ToBoolPtr(true),
				OpenWorldHint:   mcp.ToBoolPtr(false),
			},
		},
		{
			name:  "all hints combined",
			title: "All Tool",
			hints: Readonly | Destructive | Idempotent | OpenWorld,
			expectedResult: mcp.ToolAnnotation{
				Title:           "All Tool",
				ReadOnlyHint:    mcp.ToBoolPtr(true),
				DestructiveHint: mcp.ToBoolPtr(true),
				IdempotentHint:  mcp.ToBoolPtr(true),
				OpenWorldHint:   mcp.ToBoolPtr(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToolAnnotations(tt.title, tt.hints)

			if result.Title != tt.expectedResult.Title {
				t.Errorf("Expected title %s, got %s", tt.expectedResult.Title, result.Title)
			}

			if *result.ReadOnlyHint != *tt.expectedResult.ReadOnlyHint {
				t.Errorf("Expected ReadOnlyHint %v, got %v", *tt.expectedResult.ReadOnlyHint, *result.ReadOnlyHint)
			}

			if *result.DestructiveHint != *tt.expectedResult.DestructiveHint {
				t.Errorf("Expected DestructiveHint %v, got %v", *tt.expectedResult.DestructiveHint, *result.DestructiveHint)
			}

			if *result.IdempotentHint != *tt.expectedResult.IdempotentHint {
				t.Errorf("Expected IdempotentHint %v, got %v", *tt.expectedResult.IdempotentHint, *result.IdempotentHint)
			}

			if *result.OpenWorldHint != *tt.expectedResult.OpenWorldHint {
				t.Errorf("Expected OpenWorldHint %v, got %v", *tt.expectedResult.OpenWorldHint, *result.OpenWorldHint)
			}
		})
	}
}

func TestBitMaskOperations(t *testing.T) {
	// Test that bitwise operations work correctly
	combined := Readonly | Destructive

	if combined&Readonly == 0 {
		t.Error("Expected READONLY to be set in combined flags")
	}

	if combined&Destructive == 0 {
		t.Error("Expected DESTRUCTIVE to be set in combined flags")
	}

	if combined&Idempotent != 0 {
		t.Error("Expected IDEMPOTENT to not be set in combined flags")
	}

	if combined&OpenWorld != 0 {
		t.Error("Expected OPEN_WORLD to not be set in combined flags")
	}
}
