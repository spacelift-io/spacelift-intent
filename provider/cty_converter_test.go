package provider

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestCtyConverter_CtyValueToAny_Primitives(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    cty.Value
		expected any
	}{
		// String values
		{
			name:     "string value",
			input:    cty.StringVal("hello"),
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    cty.StringVal(""),
			expected: "",
		},
		{
			name:     "string with special chars",
			input:    cty.StringVal("hello\nworld\t!@#$%"),
			expected: "hello\nworld\t!@#$%",
		},

		// Number values
		{
			name:     "integer number",
			input:    cty.NumberIntVal(42),
			expected: int64(42),
		},
		{
			name:     "negative integer",
			input:    cty.NumberIntVal(-123),
			expected: int64(-123),
		},
		{
			name:     "zero",
			input:    cty.NumberIntVal(0),
			expected: int64(0),
		},
		{
			name:     "float number",
			input:    cty.NumberFloatVal(3.14),
			expected: 3.14,
		},
		{
			name:     "large integer",
			input:    cty.NumberIntVal(9223372036854775807), // max int64
			expected: int64(9223372036854775807),
		},

		// Boolean values
		{
			name:     "boolean true",
			input:    cty.True,
			expected: true,
		},
		{
			name:     "boolean false",
			input:    cty.False,
			expected: false,
		},

		// Special values
		{
			name:     "null string",
			input:    cty.NullVal(cty.String),
			expected: nil,
		},
		{
			name:     "null number",
			input:    cty.NullVal(cty.Number),
			expected: nil,
		},
		{
			name:     "null boolean",
			input:    cty.NullVal(cty.Bool),
			expected: nil,
		},
		{
			name:     "unknown string",
			input:    cty.UnknownVal(cty.String),
			expected: "__cty_unknown__",
		},
		{
			name:     "unknown number",
			input:    cty.UnknownVal(cty.Number),
			expected: "__cty_unknown__",
		},
		{
			name:     "unknown boolean",
			input:    cty.UnknownVal(cty.Bool),
			expected: "__cty_unknown__",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ctyValueToAny(tt.input)
			require.NoError(t, err, "ctyValueToAny should not return error")
			assert.Equal(t, tt.expected, result, "result should match expected value")
		})
	}
}

func TestCtyConverter_CtyValueToAny_LargeNumbers(t *testing.T) {
	converter := &CtyConverter{}

	// Test very large number that can't fit in float64
	bigNumStr := "999999999999999999999999999999999999999999999999999"
	bigNumber, err := cty.ParseNumberVal(bigNumStr)
	require.NoError(t, err, "should parse large number")

	result, err := converter.ctyValueToAny(bigNumber)
	require.NoError(t, err, "ctyValueToAny should not return error")

	// Should return as string since it can't fit in float64
	assert.Equal(t, bigNumStr, result, "large number should be returned as string")
}

func TestCtyConverter_CtyValueToAny_Collections(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    cty.Value
		expected any
	}{
		// Lists
		{
			name:     "empty list",
			input:    cty.ListValEmpty(cty.String),
			expected: []any{},
		},
		{
			name: "list of strings",
			input: cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			expected: []any{"a", "b", "c"},
		},
		{
			name: "list with mixed known and unknown",
			input: cty.ListVal([]cty.Value{
				cty.StringVal("known"),
				cty.UnknownVal(cty.String),
				cty.NullVal(cty.String),
			}),
			expected: []any{"known", "__cty_unknown__", nil},
		},

		// Sets (should be converted to arrays)
		{
			name:     "empty set",
			input:    cty.SetValEmpty(cty.String),
			expected: []any{},
		},
		{
			name: "set of strings",
			input: cty.SetVal([]cty.Value{
				cty.StringVal("x"),
				cty.StringVal("y"),
			}),
			expected: []any{"x", "y"},
		},

		// Maps
		{
			name:     "empty map",
			input:    cty.MapValEmpty(cty.String),
			expected: map[string]any{},
		},
		{
			name: "map of strings",
			input: cty.MapVal(map[string]cty.Value{
				"key1": cty.StringVal("value1"),
				"key2": cty.StringVal("value2"),
			}),
			expected: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "map with unknown and null values",
			input: cty.MapVal(map[string]cty.Value{
				"str1":    cty.StringVal("hello"),
				"str2":    cty.StringVal("world"),
				"null":    cty.NullVal(cty.String),
				"unknown": cty.UnknownVal(cty.String),
			}),
			expected: map[string]any{
				"str1":    "hello",
				"str2":    "world",
				"null":    nil,
				"unknown": "__cty_unknown__",
			},
		},

		// Tuples
		{
			name:     "empty tuple",
			input:    cty.TupleVal([]cty.Value{}),
			expected: []any{},
		},
		{
			name: "tuple with mixed types",
			input: cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
				cty.NumberIntVal(42),
				cty.True,
			}),
			expected: []any{"hello", int64(42), true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ctyValueToAny(tt.input)
			require.NoError(t, err, "ctyValueToAny should not return error")
			assert.Equal(t, tt.expected, result, "result should match expected value")
		})
	}
}

func TestCtyConverter_CtyValueToAny_Objects(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    cty.Value
		expected any
	}{
		{
			name:     "empty object",
			input:    cty.ObjectVal(map[string]cty.Value{}),
			expected: map[string]any{},
		},
		{
			name: "simple object",
			input: cty.ObjectVal(map[string]cty.Value{
				"name":    cty.StringVal("test"),
				"count":   cty.NumberIntVal(5),
				"enabled": cty.True,
			}),
			expected: map[string]any{
				"name":    "test",
				"count":   int64(5),
				"enabled": true,
			},
		},
		{
			name: "object with null and unknown values",
			input: cty.ObjectVal(map[string]cty.Value{
				"name":    cty.StringVal("test"),
				"maybe":   cty.UnknownVal(cty.String),
				"nothing": cty.NullVal(cty.Number),
			}),
			expected: map[string]any{
				"name":    "test",
				"maybe":   "__cty_unknown__",
				"nothing": nil,
			},
		},
		{
			name: "nested object",
			input: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("parent"),
				"child": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("child"),
					"age":  cty.NumberIntVal(10),
				}),
			}),
			expected: map[string]any{
				"name": "parent",
				"child": map[string]any{
					"name": "child",
					"age":  int64(10),
				},
			},
		},
		{
			name: "object with collections",
			input: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"tags": cty.ListVal([]cty.Value{
					cty.StringVal("tag1"),
					cty.StringVal("tag2"),
				}),
				"config": cty.MapVal(map[string]cty.Value{
					"key1": cty.StringVal("value1"),
					"key2": cty.StringVal("value2"),
				}),
			}),
			expected: map[string]any{
				"name": "test",
				"tags": []any{"tag1", "tag2"},
				"config": map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ctyValueToAny(tt.input)
			require.NoError(t, err, "ctyValueToAny should not return error")
			assert.Equal(t, tt.expected, result, "result should match expected value")
		})
	}
}

func TestCtyConverter_CtyValueToMap_TopLevel(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    cty.Value
		expected map[string]any
	}{
		{
			name: "object value",
			input: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
			}),
			expected: map[string]any{
				"name": "test",
			},
		},
		{
			name:  "unknown value gets wrapped",
			input: cty.UnknownVal(cty.String),
			expected: map[string]any{
				"__cty_unknown__": true,
			},
		},
		{
			name:  "primitive value gets wrapped",
			input: cty.StringVal("hello"),
			expected: map[string]any{
				"value": "hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.CtyValueToMap(tt.input)
			require.NoError(t, err, "CtyValueToMap should not return error")
			assert.Equal(t, tt.expected, result, "result should match expected value")
		})
	}
}

func TestCtyConverter_MapToCtyValue_Primitives(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    map[string]any
		ctyType  cty.Type
		expected cty.Value
	}{
		// String values
		{
			name:     "string value",
			input:    map[string]any{"value": "hello"},
			ctyType:  cty.String,
			expected: cty.StringVal("hello"),
		},
		{
			name:     "empty string",
			input:    map[string]any{"value": ""},
			ctyType:  cty.String,
			expected: cty.StringVal(""),
		},
		{
			name:     "string from any value",
			input:    map[string]any{"anything": 123},
			ctyType:  cty.String,
			expected: cty.StringVal("123"),
		},
		{
			name:     "null string",
			input:    map[string]any{"value": nil},
			ctyType:  cty.String,
			expected: cty.NullVal(cty.String),
		},

		// Number values
		{
			name:     "integer number",
			input:    map[string]any{"value": 42},
			ctyType:  cty.Number,
			expected: cty.NumberIntVal(42),
		},
		{
			name:     "int64 number",
			input:    map[string]any{"value": int64(123)},
			ctyType:  cty.Number,
			expected: cty.NumberIntVal(123),
		},
		{
			name:     "float64 number",
			input:    map[string]any{"value": 3.14},
			ctyType:  cty.Number,
			expected: cty.NumberFloatVal(3.14),
		},

		// Boolean values
		{
			name:     "boolean true",
			input:    map[string]any{"value": true},
			ctyType:  cty.Bool,
			expected: cty.True,
		},
		{
			name:     "boolean false",
			input:    map[string]any{"value": false},
			ctyType:  cty.Bool,
			expected: cty.False,
		},

		// Unknown values
		{
			name:     "unknown string marker",
			input:    map[string]any{"value": "__cty_unknown__"},
			ctyType:  cty.String,
			expected: cty.UnknownVal(cty.String),
		},
		{
			name:     "unknown number marker",
			input:    map[string]any{"value": "__cty_unknown__"},
			ctyType:  cty.Number,
			expected: cty.UnknownVal(cty.Number),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.MapToCtyValue(tt.input, tt.ctyType)
			require.NoError(t, err, "MapToCtyValue should not return error")
			assert.True(t, tt.expected.RawEquals(result),
				"expected %#v, got %#v", tt.expected, result)
		})
	}
}

func TestCtyConverter_MapToCtyValue_Objects(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    map[string]any
		ctyType  cty.Type
		expected cty.Value
	}{
		{
			name: "simple object",
			input: map[string]any{
				"name":    "test",
				"count":   5,
				"enabled": true,
			},
			ctyType: cty.Object(map[string]cty.Type{
				"name":    cty.String,
				"count":   cty.Number,
				"enabled": cty.Bool,
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"name":    cty.StringVal("test"),
				"count":   cty.NumberIntVal(5),
				"enabled": cty.True,
			}),
		},
		{
			name: "object with null values",
			input: map[string]any{
				"name": "test",
				"age":  nil,
			},
			ctyType: cty.Object(map[string]cty.Type{
				"name": cty.String,
				"age":  cty.Number,
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"age":  cty.NullVal(cty.Number),
			}),
		},
		{
			name: "object with unknown values",
			input: map[string]any{
				"name": "test",
				"age":  "__cty_unknown__",
			},
			ctyType: cty.Object(map[string]cty.Type{
				"name": cty.String,
				"age":  cty.Number,
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"age":  cty.UnknownVal(cty.Number),
			}),
		},
		{
			name: "object with missing fields",
			input: map[string]any{
				"name": "test",
			},
			ctyType: cty.Object(map[string]cty.Type{
				"name": cty.String,
				"age":  cty.Number,
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"age":  cty.NullVal(cty.Number),
			}),
		},
		{
			name: "nested object",
			input: map[string]any{
				"name": "parent",
				"child": map[string]any{
					"name": "child",
					"age":  10,
				},
			},
			ctyType: cty.Object(map[string]cty.Type{
				"name": cty.String,
				"child": cty.Object(map[string]cty.Type{
					"name": cty.String,
					"age":  cty.Number,
				}),
			}),
			expected: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("parent"),
				"child": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("child"),
					"age":  cty.NumberIntVal(10),
				}),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.MapToCtyValue(tt.input, tt.ctyType)
			require.NoError(t, err, "MapToCtyValue should not return error")
			assert.True(t, tt.expected.RawEquals(result),
				"expected %#v, got %#v", tt.expected, result)
		})
	}
}

func TestCtyConverter_MapToCtyValue_Collections(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    map[string]any
		ctyType  cty.Type
		expected cty.Value
	}{
		// Lists
		{
			name:     "empty list from empty map",
			input:    map[string]any{},
			ctyType:  cty.List(cty.String),
			expected: cty.ListValEmpty(cty.String),
		},
		{
			name:    "list from array",
			input:   map[string]any{"value": []any{"a", "b", "c"}},
			ctyType: cty.List(cty.String),
			expected: cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
		},
		{
			name:    "list with mixed values",
			input:   map[string]any{"value": []any{"known", "__cty_unknown__", nil}},
			ctyType: cty.List(cty.String),
			expected: cty.ListVal([]cty.Value{
				cty.StringVal("known"),
				cty.UnknownVal(cty.String),
				cty.NullVal(cty.String),
			}),
		},

		// Sets
		{
			name:     "empty set",
			input:    map[string]any{},
			ctyType:  cty.Set(cty.String),
			expected: cty.SetValEmpty(cty.String),
		},
		{
			name:    "set from array",
			input:   map[string]any{"value": []any{"x", "y"}},
			ctyType: cty.Set(cty.String),
			expected: cty.SetVal([]cty.Value{
				cty.StringVal("x"),
				cty.StringVal("y"),
			}),
		},

		// Maps
		{
			name:     "empty map",
			input:    map[string]any{},
			ctyType:  cty.Map(cty.String),
			expected: cty.MapValEmpty(cty.String),
		},
		{
			name: "simple map",
			input: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			ctyType: cty.Map(cty.String),
			expected: cty.MapVal(map[string]cty.Value{
				"key1": cty.StringVal("value1"),
				"key2": cty.StringVal("value2"),
			}),
		},
		{
			name: "map with unknown and null",
			input: map[string]any{
				"known":   "value",
				"unknown": "__cty_unknown__",
				"null":    nil,
			},
			ctyType: cty.Map(cty.String),
			expected: cty.MapVal(map[string]cty.Value{
				"known":   cty.StringVal("value"),
				"unknown": cty.UnknownVal(cty.String),
				"null":    cty.NullVal(cty.String),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.MapToCtyValue(tt.input, tt.ctyType)
			require.NoError(t, err, "MapToCtyValue should not return error")
			assert.True(t, tt.expected.RawEquals(result),
				"expected %#v, got %#v", tt.expected, result)
		})
	}
}

func TestCtyConverter_MapToCtyValue_DynamicType(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		input    map[string]any
		expected cty.Value
	}{
		{
			name:     "infer string from single value",
			input:    map[string]any{"value": "hello"},
			expected: cty.StringVal("hello"),
		},
		{
			name:     "infer number from single value",
			input:    map[string]any{"value": 42},
			expected: cty.NumberIntVal(42),
		},
		{
			name:     "infer boolean from single value",
			input:    map[string]any{"value": true},
			expected: cty.True,
		},
		{
			name: "infer object from multiple values",
			input: map[string]any{
				"name": "test",
				"age":  25,
			},
			expected: cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("test"),
				"age":  cty.NumberIntVal(25),
			}),
		},
		{
			name:     "empty map to null dynamic",
			input:    map[string]any{},
			expected: cty.NullVal(cty.DynamicPseudoType),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.MapToCtyValue(tt.input, cty.DynamicPseudoType)
			require.NoError(t, err, "MapToCtyValue should not return error")
			assert.True(t, tt.expected.RawEquals(result),
				"expected %#v, got %#v", tt.expected, result)
		})
	}
}

func TestCtyConverter_RoundTrip(t *testing.T) {
	converter := &CtyConverter{}

	tests := []struct {
		name     string
		original cty.Value
		ctyType  cty.Type
	}{
		{
			name:     "string round trip",
			original: cty.StringVal("hello world"),
			ctyType:  cty.String,
		},
		{
			name:     "number round trip",
			original: cty.NumberIntVal(42),
			ctyType:  cty.Number,
		},
		{
			name:     "boolean round trip",
			original: cty.True,
			ctyType:  cty.Bool,
		},
		{
			name:     "null value round trip",
			original: cty.NullVal(cty.String),
			ctyType:  cty.String,
		},
		{
			name:     "unknown value round trip",
			original: cty.UnknownVal(cty.Number),
			ctyType:  cty.Number,
		},
		{
			name: "simple object round trip",
			original: cty.ObjectVal(map[string]cty.Value{
				"name":    cty.StringVal("test"),
				"count":   cty.NumberIntVal(5),
				"enabled": cty.True,
				"empty":   cty.NullVal(cty.String),
				"unknown": cty.UnknownVal(cty.Bool),
			}),
			ctyType: cty.Object(map[string]cty.Type{
				"name":    cty.String,
				"count":   cty.Number,
				"enabled": cty.Bool,
				"empty":   cty.String,
				"unknown": cty.Bool,
			}),
		},
		{
			name: "list round trip",
			original: cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.UnknownVal(cty.String),
				cty.NullVal(cty.String),
			}),
			ctyType: cty.List(cty.String),
		},
		{
			name: "set round trip",
			original: cty.SetVal([]cty.Value{
				cty.StringVal("x"),
				cty.StringVal("y"),
			}),
			ctyType: cty.Set(cty.String),
		},
		{
			name: "map round trip",
			original: cty.MapVal(map[string]cty.Value{
				"key1": cty.StringVal("value1"),
				"key2": cty.UnknownVal(cty.String),
				"key3": cty.NullVal(cty.String),
			}),
			ctyType: cty.Map(cty.String),
		},
		{
			name: "empty collections round trip",
			original: cty.ObjectVal(map[string]cty.Value{
				"empty_list": cty.ListValEmpty(cty.String),
				"empty_set":  cty.SetValEmpty(cty.Number),
				"empty_map":  cty.MapValEmpty(cty.Bool),
			}),
			ctyType: cty.Object(map[string]cty.Type{
				"empty_list": cty.List(cty.String),
				"empty_set":  cty.Set(cty.Number),
				"empty_map":  cty.Map(cty.Bool),
			}),
		},
		{
			name: "nested complex object round trip",
			original: cty.ObjectVal(map[string]cty.Value{
				"metadata": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("test-resource"),
					"version": cty.NumberIntVal(1),
				}),
				"spec": cty.ObjectVal(map[string]cty.Value{
					"replicas": cty.NumberIntVal(3),
					"ports": cty.ListVal([]cty.Value{
						cty.NumberIntVal(80),
						cty.NumberIntVal(443),
					}),
					"env": cty.MapVal(map[string]cty.Value{
						"DEBUG": cty.StringVal("true"),
						"MODE":  cty.StringVal("production"),
					}),
				}),
			}),
			ctyType: cty.Object(map[string]cty.Type{
				"metadata": cty.Object(map[string]cty.Type{
					"name":    cty.String,
					"version": cty.Number,
				}),
				"spec": cty.Object(map[string]cty.Type{
					"replicas": cty.Number,
					"ports":    cty.List(cty.Number),
					"env":      cty.Map(cty.String),
				}),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First direction: cty.Value -> map[string]any
			intermediateMap, err := converter.CtyValueToMap(tt.original)
			require.NoError(t, err, "CtyValueToMap should not return error")
			require.NotNil(t, intermediateMap, "intermediate map should not be nil")

			// Second direction: map[string]any -> cty.Value
			reconstructed, err := converter.MapToCtyValue(intermediateMap, tt.ctyType)
			require.NoError(t, err, "MapToCtyValue should not return error")

			// Verify round trip integrity
			assert.True(t, tt.original.RawEquals(reconstructed),
				"Round trip failed.\nOriginal:      %#v\nReconstructed: %#v\nIntermediate:  %+v",
				tt.original, reconstructed, intermediateMap)
		})
	}
}

func TestCtyConverter_ErrorCases(t *testing.T) {
	converter := &CtyConverter{}

	t.Run("MapToCtyValue errors", func(t *testing.T) {
		errorTests := []struct {
			name    string
			input   map[string]any
			ctyType cty.Type
		}{
			{
				name:    "invalid number type",
				input:   map[string]any{"value": "not-a-number"},
				ctyType: cty.Number,
			},
			{
				name:    "invalid bool type",
				input:   map[string]any{"value": "not-a-bool"},
				ctyType: cty.Bool,
			},
		}

		for _, tt := range errorTests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := converter.MapToCtyValue(tt.input, tt.ctyType)
				assert.Error(t, err, "should return error for invalid input")
				// Result should be null value of the expected type on error
				if !result.IsNull() {
					t.Errorf("Expected null value on error, got %#v", result)
				}
			})
		}
	})

	t.Run("ctyValueToAny unsupported type", func(t *testing.T) {
		// Test with a capsule type (unsupported)
		type testStruct struct{ Value string }
		capsuleType := cty.Capsule("test", reflect.TypeOf(testStruct{}))
		testVal := &testStruct{Value: "test"}
		capsuleVal := cty.CapsuleVal(capsuleType, testVal)

		_, err := converter.ctyValueToAny(capsuleVal)
		assert.Error(t, err, "should return error for unsupported capsule type")
		assert.Contains(t, err.Error(), "unsupported cty value type", "error should mention unsupported type")
	})
}

func TestCtyConverter_UnknownValueHelpers(t *testing.T) {
	converter := &CtyConverter{}

	t.Run("IsUnknownValue", func(t *testing.T) {
		tests := []struct {
			name     string
			input    any
			expected bool
		}{
			{
				name:     "string marker",
				input:    "__cty_unknown__",
				expected: true,
			},
			{
				name:     "object marker",
				input:    map[string]any{"__cty_unknown__": true},
				expected: true,
			},
			{
				name:     "object marker false",
				input:    map[string]any{"__cty_unknown__": false},
				expected: false,
			},
			{
				name:     "regular string",
				input:    "hello",
				expected: false,
			},
			{
				name:     "regular object",
				input:    map[string]any{"name": "test"},
				expected: false,
			},
			{
				name:     "nil value",
				input:    nil,
				expected: false,
			},
			{
				name:     "empty map",
				input:    map[string]any{},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := converter.IsUnknownValue(tt.input)
				assert.Equal(t, tt.expected, result, "IsUnknownValue result should match expected")
			})
		}
	})

	t.Run("CreateUnknownValue", func(t *testing.T) {
		result := converter.CreateUnknownValue()
		assert.Equal(t, "__cty_unknown__", result, "CreateUnknownValue should return string marker")
		assert.True(t, converter.IsUnknownValue(result), "created unknown value should be recognized as unknown")
	})
}

func TestCtyConverter_EdgeCases(t *testing.T) {
	converter := &CtyConverter{}

	t.Run("very large numbers", func(t *testing.T) {
		// Test number that exceeds float64 precision
		bigNumStr := "12345678901234567890123456789012345678901234567890"
		bigNum, err := cty.ParseNumberVal(bigNumStr)
		require.NoError(t, err, "should parse big number")

		result, err := converter.ctyValueToAny(bigNum)
		require.NoError(t, err, "should convert big number")
		assert.Equal(t, bigNumStr, result, "big number should be returned as string")
	})

	t.Run("deeply nested structures", func(t *testing.T) {
		// Create deeply nested object
		deepObject := cty.ObjectVal(map[string]cty.Value{
			"level1": cty.ObjectVal(map[string]cty.Value{
				"level2": cty.ObjectVal(map[string]cty.Value{
					"level3": cty.ObjectVal(map[string]cty.Value{
						"value": cty.StringVal("deep"),
					}),
				}),
			}),
		})

		// Convert to map
		resultMap, err := converter.CtyValueToMap(deepObject)
		require.NoError(t, err, "should convert deeply nested object")

		// Verify structure
		level1, ok := resultMap["level1"].(map[string]any)
		require.True(t, ok, "level1 should be map")
		level2, ok := level1["level2"].(map[string]any)
		require.True(t, ok, "level2 should be map")
		level3, ok := level2["level3"].(map[string]any)
		require.True(t, ok, "level3 should be map")
		value, ok := level3["value"].(string)
		require.True(t, ok, "value should be string")
		assert.Equal(t, "deep", value, "deeply nested value should be preserved")
	})

	t.Run("empty and null handling", func(t *testing.T) {
		tests := []struct {
			name     string
			input    cty.Value
			expected any
		}{
			{
				name:     "empty object",
				input:    cty.ObjectVal(map[string]cty.Value{}),
				expected: map[string]any{},
			},
			{
				name:     "empty list",
				input:    cty.ListValEmpty(cty.String),
				expected: []any{},
			},
			{
				name:     "empty set",
				input:    cty.SetValEmpty(cty.String),
				expected: []any{},
			},
			{
				name:     "empty map",
				input:    cty.MapValEmpty(cty.String),
				expected: map[string]any{},
			},
			{
				name:     "empty tuple",
				input:    cty.TupleVal([]cty.Value{}),
				expected: []any{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := converter.ctyValueToAny(tt.input)
				require.NoError(t, err, "should convert empty/null values")
				assert.Equal(t, tt.expected, result, "empty value should match expected")
			})
		}
	})
}

func TestCtyConverter_InferCtyValue_InfiniteRecursionFix(t *testing.T) {
	converter := &CtyConverter{}

	t.Run("multiple values should not cause infinite recursion", func(t *testing.T) {
		// This test case specifically covers the infinite recursion bug that was fixed
		// The bug was in inferCtyValue when it had multiple keys and would recursively
		// call itself with wrapped values: map[string]any{"value": val}
		input := map[string]any{
			"name":        "test-resource",
			"description": "A test resource",
			"count":       5,
			"enabled":     true,
		}

		result := converter.inferCtyValue(input)

		// Should create an object with the correct values
		expectedType := cty.Object(map[string]cty.Type{
			"name":        cty.String,
			"description": cty.String,
			"count":       cty.Number,
			"enabled":     cty.Bool,
		})

		assert.True(t, result.Type().Equals(expectedType), "should infer correct object type")
		assert.Equal(t, "test-resource", result.GetAttr("name").AsString())
		assert.Equal(t, "A test resource", result.GetAttr("description").AsString())
		count, _ := result.GetAttr("count").AsBigFloat().Int64()
		assert.Equal(t, int64(5), count)
		assert.Equal(t, true, result.GetAttr("enabled").True())
	})

	t.Run("single value should unwrap correctly", func(t *testing.T) {
		input := map[string]any{"value": "hello world"}
		result := converter.inferCtyValue(input)

		assert.Equal(t, cty.String, result.Type())
		assert.Equal(t, "hello world", result.AsString())
	})

	t.Run("nested objects should not cause infinite recursion", func(t *testing.T) {
		input := map[string]any{
			"outer": "value1",
			"nested": map[string]any{
				"inner1": "value2",
				"inner2": 42,
				"deeply_nested": map[string]any{
					"deep": "value3",
				},
			},
		}

		result := converter.inferCtyValue(input)

		// Should successfully create nested object structure without stack overflow
		assert.True(t, result.Type().IsObjectType())
		assert.Equal(t, "value1", result.GetAttr("outer").AsString())

		nested := result.GetAttr("nested")
		assert.Equal(t, "value2", nested.GetAttr("inner1").AsString())

		inner2, _ := nested.GetAttr("inner2").AsBigFloat().Int64()
		assert.Equal(t, int64(42), inner2)

		deeplyNested := nested.GetAttr("deeply_nested")
		assert.Equal(t, "value3", deeplyNested.GetAttr("deep").AsString())
	})

	t.Run("empty map should return null", func(t *testing.T) {
		input := map[string]any{}
		result := converter.inferCtyValue(input)

		assert.True(t, result.IsNull())
		assert.Equal(t, cty.DynamicPseudoType, result.Type())
	})
}
