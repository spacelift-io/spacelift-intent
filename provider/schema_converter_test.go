// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestSchemaConverter_CtyTypeToString(t *testing.T) {
	converter := &SchemaConverter{}

	tests := []struct {
		name     string
		input    cty.Type
		expected string
	}{
		{
			name:     "string type",
			input:    cty.String,
			expected: "string",
		},
		{
			name:     "number type",
			input:    cty.Number,
			expected: "number",
		},
		{
			name:     "boolean type",
			input:    cty.Bool,
			expected: "boolean",
		},
		{
			name:     "list type",
			input:    cty.List(cty.String),
			expected: "list",
		},
		{
			name:     "set type",
			input:    cty.Set(cty.Number),
			expected: "set",
		},
		{
			name:     "map type",
			input:    cty.Map(cty.Bool),
			expected: "map",
		},
		{
			name:     "object type",
			input:    cty.Object(map[string]cty.Type{"field": cty.String}),
			expected: "object",
		},
		{
			name:     "dynamic type",
			input:    cty.DynamicPseudoType,
			expected: "unknown",
		},
		{
			name:     "tuple type returns unknown",
			input:    cty.Tuple([]cty.Type{cty.String, cty.Number}),
			expected: "unknown",
		},
		{
			name:     "nested list of strings",
			input:    cty.List(cty.String),
			expected: "list",
		},
		{
			name:     "nested set of numbers",
			input:    cty.Set(cty.Number),
			expected: "set",
		},
		{
			name:     "map of booleans",
			input:    cty.Map(cty.Bool),
			expected: "map",
		},
		{
			name:     "complex object type",
			input:    cty.Object(map[string]cty.Type{
				"name":    cty.String,
				"age":     cty.Number,
				"enabled": cty.Bool,
			}),
			expected: "object",
		},
		{
			name:     "list of objects",
			input:    cty.List(cty.Object(map[string]cty.Type{"id": cty.String})),
			expected: "list",
		},
		{
			name:     "set of objects",
			input:    cty.Set(cty.Object(map[string]cty.Type{"value": cty.Number})),
			expected: "set",
		},
		{
			name:     "map of objects",
			input:    cty.Map(cty.Object(map[string]cty.Type{"config": cty.String})),
			expected: "map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ctyTypeToString(tt.input)
			assert.Equal(t, tt.expected, result, "ctyTypeToString should return correct string representation")
		})
	}
}

func TestSchemaConverter_CtyTypeToString_EdgeCases(t *testing.T) {
	converter := &SchemaConverter{}

	t.Run("empty object type", func(t *testing.T) {
		emptyObject := cty.Object(map[string]cty.Type{})
		result := converter.ctyTypeToString(emptyObject)
		assert.Equal(t, "object", result, "empty object should still return 'object'")
	})

	t.Run("empty list type", func(t *testing.T) {
		// List with empty string as element type
		emptyList := cty.List(cty.String)
		result := converter.ctyTypeToString(emptyList)
		assert.Equal(t, "list", result, "list type should return 'list'")
	})

	t.Run("nested collection types", func(t *testing.T) {
		// List of lists
		nestedList := cty.List(cty.List(cty.String))
		result := converter.ctyTypeToString(nestedList)
		assert.Equal(t, "list", result, "nested list should return 'list'")

		// Set of sets
		nestedSet := cty.Set(cty.Set(cty.Number))
		result = converter.ctyTypeToString(nestedSet)
		assert.Equal(t, "set", result, "nested set should return 'set'")

		// Map of maps
		nestedMap := cty.Map(cty.Map(cty.Bool))
		result = converter.ctyTypeToString(nestedMap)
		assert.Equal(t, "map", result, "nested map should return 'map'")
	})

	t.Run("complex nested types", func(t *testing.T) {
		// Object containing lists, sets, and maps
		complexType := cty.Object(map[string]cty.Type{
			"list_field": cty.List(cty.String),
			"set_field":  cty.Set(cty.Number),
			"map_field":  cty.Map(cty.Bool),
			"nested_obj": cty.Object(map[string]cty.Type{
				"inner": cty.String,
			}),
		})
		result := converter.ctyTypeToString(complexType)
		assert.Equal(t, "object", result, "complex nested object should return 'object'")
	})

	t.Run("list of complex objects", func(t *testing.T) {
		complexObjectType := cty.Object(map[string]cty.Type{
			"id":   cty.String,
			"tags": cty.List(cty.String),
			"metadata": cty.Object(map[string]cty.Type{
				"created": cty.String,
				"updated": cty.String,
			}),
		})
		listOfComplexObjects := cty.List(complexObjectType)
		result := converter.ctyTypeToString(listOfComplexObjects)
		assert.Equal(t, "list", result, "list of complex objects should return 'list'")
	})
}

func TestSchemaConverter_CtyTypeToString_PrimitiveTypes(t *testing.T) {
	converter := &SchemaConverter{}

	t.Run("all primitive types", func(t *testing.T) {
		primitives := []struct {
			name     string
			ctyType  cty.Type
			expected string
		}{
			{"string", cty.String, "string"},
			{"number", cty.Number, "number"},
			{"bool", cty.Bool, "boolean"},
		}

		for _, tc := range primitives {
			t.Run(tc.name, func(t *testing.T) {
				result := converter.ctyTypeToString(tc.ctyType)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

func TestSchemaConverter_CtyTypeToString_CollectionTypes(t *testing.T) {
	converter := &SchemaConverter{}

	elementTypes := []struct {
		name        string
		elementType cty.Type
	}{
		{"string elements", cty.String},
		{"number elements", cty.Number},
		{"bool elements", cty.Bool},
		{"object elements", cty.Object(map[string]cty.Type{"field": cty.String})},
	}

	for _, elem := range elementTypes {
		t.Run("list of "+elem.name, func(t *testing.T) {
			listType := cty.List(elem.elementType)
			result := converter.ctyTypeToString(listType)
			assert.Equal(t, "list", result)
		})

		t.Run("set of "+elem.name, func(t *testing.T) {
			setType := cty.Set(elem.elementType)
			result := converter.ctyTypeToString(setType)
			assert.Equal(t, "set", result)
		})

		t.Run("map of "+elem.name, func(t *testing.T) {
			mapType := cty.Map(elem.elementType)
			result := converter.ctyTypeToString(mapType)
			assert.Equal(t, "map", result)
		})
	}
}

func TestSchemaConverter_CtyTypeToString_ObjectVariations(t *testing.T) {
	converter := &SchemaConverter{}

	t.Run("object with single attribute", func(t *testing.T) {
		objType := cty.Object(map[string]cty.Type{
			"single": cty.String,
		})
		result := converter.ctyTypeToString(objType)
		assert.Equal(t, "object", result)
	})

	t.Run("object with multiple primitive attributes", func(t *testing.T) {
		objType := cty.Object(map[string]cty.Type{
			"str":  cty.String,
			"num":  cty.Number,
			"bool": cty.Bool,
		})
		result := converter.ctyTypeToString(objType)
		assert.Equal(t, "object", result)
	})

	t.Run("object with collection attributes", func(t *testing.T) {
		objType := cty.Object(map[string]cty.Type{
			"list": cty.List(cty.String),
			"set":  cty.Set(cty.Number),
			"map":  cty.Map(cty.Bool),
		})
		result := converter.ctyTypeToString(objType)
		assert.Equal(t, "object", result)
	})

	t.Run("deeply nested object", func(t *testing.T) {
		innermost := cty.Object(map[string]cty.Type{"value": cty.String})
		middle := cty.Object(map[string]cty.Type{"inner": innermost})
		outer := cty.Object(map[string]cty.Type{"middle": middle})

		result := converter.ctyTypeToString(outer)
		assert.Equal(t, "object", result)
	})

	t.Run("object with mixed nesting", func(t *testing.T) {
		objType := cty.Object(map[string]cty.Type{
			"primitive":  cty.String,
			"collection": cty.List(cty.Number),
			"nested": cty.Object(map[string]cty.Type{
				"field": cty.Bool,
			}),
		})
		result := converter.ctyTypeToString(objType)
		assert.Equal(t, "object", result)
	})
}

// TestSchemaConverter_CtyTypeToString_ComprehensiveCoverage ensures all code paths are tested
func TestSchemaConverter_CtyTypeToString_ComprehensiveCoverage(t *testing.T) {
	converter := &SchemaConverter{}

	// Test all supported types to ensure 100% coverage of the switch statement
	allTypes := []struct {
		name     string
		ctyType  cty.Type
		expected string
	}{
		// Primitive types
		{"string", cty.String, "string"},
		{"number", cty.Number, "number"},
		{"bool", cty.Bool, "boolean"},

		// Collection types
		{"list", cty.List(cty.String), "list"},
		{"set", cty.Set(cty.String), "set"},
		{"map", cty.Map(cty.String), "map"},
		{"object", cty.Object(map[string]cty.Type{"f": cty.String}), "object"},

		// Special types
		{"dynamic", cty.DynamicPseudoType, "unknown"},

		// Types that fall through to default case
		{"tuple", cty.Tuple([]cty.Type{cty.String}), "unknown"},
	}

	for _, tt := range allTypes {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.ctyTypeToString(tt.ctyType)
			assert.Equal(t, tt.expected, result,
				"ctyTypeToString(%s) should return %q", tt.name, tt.expected)
		})
	}
}

// TestSchemaConverter_IntegrationTest tests the converter with realistic AWS-like schema patterns
func TestSchemaConverter_IntegrationTest(t *testing.T) {
	converter := &SchemaConverter{}

	t.Run("AWS instance-like types", func(t *testing.T) {
		// Simulate types found in aws_instance resource
		types := map[string]cty.Type{
			"id":            cty.String,
			"ami":           cty.String,
			"instance_type": cty.String,
			"tags":          cty.Map(cty.String),
			"security_groups": cty.Set(cty.String),
			"ebs_block_device": cty.List(cty.Object(map[string]cty.Type{
				"device_name": cty.String,
				"volume_size": cty.Number,
				"encrypted":   cty.Bool,
			})),
		}

		expected := map[string]string{
			"id":               "string",
			"ami":              "string",
			"instance_type":    "string",
			"tags":             "map",
			"security_groups":  "set",
			"ebs_block_device": "list",
		}

		for field, ctyType := range types {
			result := converter.ctyTypeToString(ctyType)
			assert.Equal(t, expected[field], result,
				"Field %s should be %s", field, expected[field])
		}
	})

	t.Run("CloudFront distribution-like nested types", func(t *testing.T) {
		// Simulate deeply nested CloudFront structure
		originType := cty.Object(map[string]cty.Type{
			"origin_id":   cty.String,
			"domain_name": cty.String,
			"custom_origin_config": cty.Object(map[string]cty.Type{
				"http_port":              cty.Number,
				"https_port":             cty.Number,
				"origin_protocol_policy": cty.String,
			}),
		})

		cacheBehaviorType := cty.Object(map[string]cty.Type{
			"target_origin_id":       cty.String,
			"viewer_protocol_policy": cty.String,
			"allowed_methods":        cty.Set(cty.String),
			"cached_methods":         cty.Set(cty.String),
		})

		distributionType := cty.Object(map[string]cty.Type{
			"enabled":                  cty.Bool,
			"origins":                  cty.Set(originType),
			"default_cache_behavior":   cacheBehaviorType,
			"ordered_cache_behaviors":  cty.List(cacheBehaviorType),
		})

		result := converter.ctyTypeToString(distributionType)
		assert.Equal(t, "object", result, "complex distribution type should be object")

		// Test nested types
		assert.Equal(t, "set", converter.ctyTypeToString(cty.Set(originType)))
		assert.Equal(t, "object", converter.ctyTypeToString(cacheBehaviorType))
		assert.Equal(t, "list", converter.ctyTypeToString(cty.List(cacheBehaviorType)))
	})

	t.Run("RabbitMQ broker-like types", func(t *testing.T) {
		// Simulate aws_mq_broker structure
		userType := cty.Object(map[string]cty.Type{
			"username":         cty.String,
			"password":         cty.String,
			"console_access":   cty.Bool,
			"groups":           cty.Set(cty.String),
			"replication_user": cty.Bool,
		})

		brokerType := cty.Object(map[string]cty.Type{
			"broker_name":              cty.String,
			"engine_type":              cty.String,
			"engine_version":           cty.String,
			"host_instance_type":       cty.String,
			"users":                    cty.Set(userType),
			"security_groups":          cty.Set(cty.String),
			"subnet_ids":               cty.Set(cty.String),
			"publicly_accessible":      cty.Bool,
			"auto_minor_version_upgrade": cty.Bool,
		})

		result := converter.ctyTypeToString(brokerType)
		assert.Equal(t, "object", result)

		// Verify user type
		assert.Equal(t, "set", converter.ctyTypeToString(cty.Set(userType)))
		assert.Equal(t, "object", converter.ctyTypeToString(userType))
	})
}
