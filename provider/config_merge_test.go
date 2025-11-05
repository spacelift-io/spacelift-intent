package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMergeMaps_NilInputs(t *testing.T) {
	tests := []struct {
		name         string
		currentState map[string]any
		newConfig    map[string]any
		expected     map[string]any
	}{
		{
			name:         "both nil",
			currentState: nil,
			newConfig:    nil,
			expected:     map[string]any{},
		},
		{
			name:         "currentState nil, newConfig empty",
			currentState: nil,
			newConfig:    map[string]any{},
			expected:     map[string]any{},
		},
		{
			name:         "currentState nil, newConfig has values",
			currentState: nil,
			newConfig:    map[string]any{"key": "value"},
			expected:     map[string]any{"key": "value"},
		},
		{
			name:         "currentState has values, newConfig nil",
			currentState: map[string]any{"key": "value"},
			newConfig:    nil,
			expected:     map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMaps(tt.currentState, tt.newConfig)
			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("mergeMaps() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeMaps_SimpleFields(t *testing.T) {
	tests := []struct {
		name         string
		currentState map[string]any
		newConfig    map[string]any
		expected     map[string]any
	}{
		{
			name: "preserve field from currentState when missing in newConfig",
			currentState: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			newConfig: map[string]any{
				"field1": "updated1",
			},
			expected: map[string]any{
				"field1": "updated1",
				"field2": "value2",
			},
		},
		{
			name: "add new field from newConfig",
			currentState: map[string]any{
				"field1": "value1",
			},
			newConfig: map[string]any{
				"field2": "value2",
			},
			expected: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
		},
		{
			name: "overwrite with explicit nil",
			currentState: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			newConfig: map[string]any{
				"field1": nil,
			},
			expected: map[string]any{
				"field1": nil,
				"field2": "value2",
			},
		},
		{
			name: "overwrite with different types",
			currentState: map[string]any{
				"field1": "string",
				"field2": 123,
				"field3": true,
			},
			newConfig: map[string]any{
				"field1": 456,
				"field2": "updated",
				"field3": false,
			},
			expected: map[string]any{
				"field1": 456,
				"field2": "updated",
				"field3": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMaps(tt.currentState, tt.newConfig)
			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("mergeMaps() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeMaps_NestedMaps(t *testing.T) {
	tests := []struct {
		name         string
		currentState map[string]any
		newConfig    map[string]any
		expected     map[string]any
	}{
		{
			name: "merge nested maps preserving missing fields",
			currentState: map[string]any{
				"outer": map[string]any{
					"field1": "value1",
					"field2": "value2",
				},
			},
			newConfig: map[string]any{
				"outer": map[string]any{
					"field1": "updated1",
				},
			},
			expected: map[string]any{
				"outer": map[string]any{
					"field1": "updated1",
					"field2": "value2",
				},
			},
		},
		{
			name: "merge deeply nested maps",
			currentState: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"field1": "value1",
						"field2": "value2",
					},
					"other": "data",
				},
			},
			newConfig: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"field1": "updated1",
					},
				},
			},
			expected: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"field1": "updated1",
						"field2": "value2",
					},
					"other": "data",
				},
			},
		},
		{
			name: "replace nested map with primitive",
			currentState: map[string]any{
				"field": map[string]any{
					"nested1": "value1",
					"nested2": "value2",
				},
			},
			newConfig: map[string]any{
				"field": "simple_string",
			},
			expected: map[string]any{
				"field": "simple_string",
			},
		},
		{
			name: "replace primitive with nested map",
			currentState: map[string]any{
				"field": "simple_string",
			},
			newConfig: map[string]any{
				"field": map[string]any{
					"nested1": "value1",
					"nested2": "value2",
				},
			},
			expected: map[string]any{
				"field": map[string]any{
					"nested1": "value1",
					"nested2": "value2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMaps(tt.currentState, tt.newConfig)
			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("mergeMaps() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeMaps_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name         string
		currentState map[string]any
		newConfig    map[string]any
		expected     map[string]any
	}{
		{
			name: "EC2 security group update scenario",
			currentState: map[string]any{
				"ami":                 "ami-12345",
				"instance_type":       "t3.micro",
				"subnet_id":           "subnet-abc",
				"vpc_security_group_ids": []any{"sg-111"},
				"tags": map[string]any{
					"Name": "web-server",
				},
			},
			newConfig: map[string]any{
				"ami":                 "ami-12345",
				"instance_type":       "t3.micro",
				"subnet_id":           "subnet-abc",
				"vpc_security_group_ids": []any{"sg-111", "sg-222"},
				"tags": map[string]any{
					"Name": "web-server",
				},
			},
			expected: map[string]any{
				"ami":                 "ami-12345",
				"instance_type":       "t3.micro",
				"subnet_id":           "subnet-abc",
				"vpc_security_group_ids": []any{"sg-111", "sg-222"},
				"tags": map[string]any{
					"Name": "web-server",
				},
			},
		},
		{
			name: "partial update with nested structures",
			currentState: map[string]any{
				"name":        "resource-1",
				"description": "original description",
				"config": map[string]any{
					"enabled": true,
					"timeout": 30,
					"retries": 3,
				},
				"metadata": map[string]any{
					"created_at": "2024-01-01",
					"owner":      "team-a",
				},
			},
			newConfig: map[string]any{
				"name": "resource-1-updated",
				"config": map[string]any{
					"timeout": 60,
				},
			},
			expected: map[string]any{
				"name":        "resource-1-updated",
				"description": "original description",
				"config": map[string]any{
					"enabled": true,
					"timeout": 60,
					"retries": 3,
				},
				"metadata": map[string]any{
					"created_at": "2024-01-01",
					"owner":      "team-a",
				},
			},
		},
		{
			name: "arrays and slices are replaced not merged",
			currentState: map[string]any{
				"tags": []any{"tag1", "tag2", "tag3"},
			},
			newConfig: map[string]any{
				"tags": []any{"tag1", "tag4"},
			},
			expected: map[string]any{
				"tags": []any{"tag1", "tag4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMaps(tt.currentState, tt.newConfig)
			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("mergeMaps() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeMaps_DoesNotMutateInputs(t *testing.T) {
	currentState := map[string]any{
		"field1": "value1",
		"nested": map[string]any{
			"field2": "value2",
		},
	}
	newConfig := map[string]any{
		"field1": "updated1",
		"nested": map[string]any{
			"field3": "value3",
		},
	}

	// Make copies to compare against later
	originalCurrentState := map[string]any{
		"field1": "value1",
		"nested": map[string]any{
			"field2": "value2",
		},
	}
	originalNewConfig := map[string]any{
		"field1": "updated1",
		"nested": map[string]any{
			"field3": "value3",
		},
	}

	_ = mergeMaps(currentState, newConfig)

	// Verify currentState wasn't mutated
	if diff := cmp.Diff(originalCurrentState, currentState); diff != "" {
		t.Errorf("currentState was mutated (-want +got):\n%s", diff)
	}

	// Verify newConfig wasn't mutated
	if diff := cmp.Diff(originalNewConfig, newConfig); diff != "" {
		t.Errorf("newConfig was mutated (-want +got):\n%s", diff)
	}
}
