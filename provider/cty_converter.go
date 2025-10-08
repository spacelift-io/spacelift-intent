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

package provider

import (
	"fmt"
	"math/big"

	"github.com/zclconf/go-cty/cty"
)

// CtyConverter handles conversion between cty.Value and Go map[string]any
type CtyConverter struct{}

// MapToCtyValue converts a map[string]any to cty.Value with the given type
func (c *CtyConverter) MapToCtyValue(data map[string]any, ty cty.Type) (cty.Value, error) {
	if data == nil {
		return cty.NullVal(ty), nil
	}

	if ty == cty.DynamicPseudoType {
		// For dynamic types, we need to infer the type from the data
		return c.inferCtyValue(data), nil
	}

	// Check if this is a single "value" key containing an array/slice for collections
	if (ty.IsListType() || ty.IsSetType()) && len(data) == 1 {
		if value, ok := data["value"]; ok {
			if slice, ok := value.([]any); ok {
				return c.convertSliceToCollection(slice, ty)
			}
		}
	}

	switch {
	case ty.IsPrimitiveType():
		return c.convertPrimitive(data, ty)
	case ty.IsObjectType():
		return c.convertObject(data, ty)
	case ty.IsMapType():
		return c.convertMap(data, ty)
	case ty.IsListType():
		return c.convertList(data, ty)
	case ty.IsSetType():
		return c.convertSet(data, ty)
	default:
		return cty.DynamicVal, fmt.Errorf("unsupported cty type: %s", ty.FriendlyName())
	}
}

// CtyValueToMap converts a cty.Value to map[string]any
func (c *CtyConverter) CtyValueToMap(val cty.Value) (map[string]any, error) {
	result, err := c.ctyValueToAny(val)
	if err != nil {
		return nil, err
	}

	// Ensure we always return a map for top-level calls
	if resultMap, ok := result.(map[string]any); ok {
		return resultMap, nil
	}

	// Handle special cases that need wrapping
	if result == "__cty_unknown__" {
		return map[string]any{"__cty_unknown__": true}, nil
	}

	// For other non-map values at top level, wrap them (this shouldn't happen in normal usage)
	if result == nil {
		return map[string]any{}, nil
	}
	return map[string]any{"value": result}, nil
}

// ctyValueToAny converts a cty.Value to any Go type (used internally)
func (c *CtyConverter) ctyValueToAny(val cty.Value) (any, error) {
	if !val.IsKnown() {
		// Return a string marker for unknown values
		return "__cty_unknown__", nil
	}

	if val.IsNull() {
		return nil, nil
	}

	switch val.Type() {
	case cty.String:
		return val.AsString(), nil
	case cty.Number:
		// For JSON compatibility, convert to appropriate Go numeric type
		bf := val.AsBigFloat()
		if bf.IsInt() {
			if i, accuracy := bf.Int64(); accuracy == big.Exact {
				return i, nil
			}
			// For very large integers that don't fit in int64, return as string
			return val.AsBigFloat().Text('f', -1), nil
		}
		if f, accuracy := bf.Float64(); accuracy == big.Exact {
			return f, nil
		}
		// For very large floats that lose precision in float64, return as string
		return val.AsBigFloat().Text('g', -1), nil
	case cty.Bool:
		return val.True(), nil
	}

	if val.Type().IsObjectType() {
		result := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			key, elemVal := it.Element()
			keyStr := key.AsString()

			if !elemVal.IsKnown() {
				result[keyStr] = "__cty_unknown__"
			} else if elemVal.IsNull() {
				result[keyStr] = nil
			} else {
				elemValue, err := c.ctyValueToAny(elemVal)
				if err != nil {
					return nil, fmt.Errorf("failed to convert element %s: %w", keyStr, err)
				}
				result[keyStr] = elemValue
			}
		}
		return result, nil
	}

	if val.Type().IsMapType() {
		// Handle maps - use map[string]any
		result := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			key, elemVal := it.Element()
			keyStr := key.AsString()

			elemValue, err := c.ctyValueToAny(elemVal)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map element %s: %w", keyStr, err)
			}
			result[keyStr] = elemValue
		}
		return result, nil
	}

	if val.Type().IsListType() || val.Type().IsSetType() || val.Type().IsTupleType() {
		// Handle lists, sets, and tuples - use []any
		result := make([]any, 0)
		for it := val.ElementIterator(); it.Next(); {
			_, elemVal := it.Element()

			elemValue, err := c.ctyValueToAny(elemVal)
			if err != nil {
				return nil, fmt.Errorf("failed to convert collection element: %w", err)
			}
			result = append(result, elemValue)
		}
		return result, nil
	}

	return nil, fmt.Errorf("unsupported cty value type: %s", val.Type().FriendlyName())
}

// Helper methods

func (c *CtyConverter) convertPrimitive(data map[string]any, ty cty.Type) (cty.Value, error) {
	// Check for unknown value marker first
	if len(data) == 1 {
		unknown, exists := data["__cty_unknown__"]
		if exists && unknown == true {
			return cty.UnknownVal(ty), nil
		}
	}

	// For primitive types, expect the data to have a single "value" key
	// or be a simple value
	var value any
	if len(data) == 1 && data["value"] != nil {
		value = data["value"]
	} else if len(data) == 1 {
		// Get the single value
		for _, v := range data {
			value = v
			break
		}
	} else {
		return cty.NullVal(ty), nil
	}

	// Handle unknown value marker (string form)
	if value == "__cty_unknown__" {
		return cty.UnknownVal(ty), nil
	}

	// Handle nil explicitly
	if value == nil {
		return cty.NullVal(ty), nil
	}

	switch ty {
	case cty.String:
		if s, ok := value.(string); ok {
			return cty.StringVal(s), nil
		}
		return cty.StringVal(fmt.Sprintf("%v", value)), nil
	case cty.Number:
		if num, ok := value.(int); ok {
			return cty.NumberIntVal(int64(num)), nil
		}
		if num, ok := value.(int64); ok {
			return cty.NumberIntVal(num), nil
		}
		if num, ok := value.(float64); ok {
			return cty.NumberFloatVal(num), nil
		}
		return cty.NullVal(ty), fmt.Errorf("cannot convert %T to number", value)
	case cty.Bool:
		if b, ok := value.(bool); ok {
			return cty.BoolVal(b), nil
		}
		return cty.NullVal(ty), fmt.Errorf("cannot convert %T to bool", value)
	}

	return cty.NullVal(ty), fmt.Errorf("unsupported primitive type: %s", ty.FriendlyName())
}

func (c *CtyConverter) convertObject(data map[string]any, ty cty.Type) (cty.Value, error) {
	attrTypes := ty.AttributeTypes()
	values := make(map[string]cty.Value)

	for attrName, attrType := range attrTypes {
		if val, exists := data[attrName]; exists {
			switch val {
			case nil:
				values[attrName] = cty.NullVal(attrType)
			case "__cty_unknown__":
				values[attrName] = cty.UnknownVal(attrType)
			default:
				// Convert the value based on its type
				ctyVal, err := c.convertValue(val, attrType)
				if err != nil {
					return cty.NilVal, fmt.Errorf("failed to convert attribute %s: %w", attrName, err)
				}
				values[attrName] = ctyVal
			}
		} else {
			// Attribute not present, use null
			values[attrName] = cty.NullVal(attrType)
		}
	}

	return cty.ObjectVal(values), nil
}

func (c *CtyConverter) convertMap(data map[string]any, ty cty.Type) (cty.Value, error) {
	elemType := ty.ElementType()

	// Handle empty map
	if len(data) == 0 {
		return cty.MapValEmpty(elemType), nil
	}

	values := make(map[string]cty.Value)

	for key, val := range data {
		switch val {
		case nil:
			values[key] = cty.NullVal(elemType)
		case "__cty_unknown__":
			values[key] = cty.UnknownVal(elemType)
		default:
			ctyVal, err := c.convertValue(val, elemType)
			if err != nil {
				return cty.NilVal, fmt.Errorf("failed to convert map element %s: %w", key, err)
			}
			values[key] = ctyVal
		}
	}

	return cty.MapVal(values), nil
}

func (c *CtyConverter) convertList(data map[string]any, ty cty.Type) (cty.Value, error) {
	elemType := ty.ElementType()

	// Handle empty list
	if len(data) == 0 {
		return cty.ListValEmpty(elemType), nil
	}

	var values []cty.Value

	// Convert map keys to indices and sort
	for i := 0; i < len(data); i++ {
		key := fmt.Sprintf("%d", i)
		val, exists := data[key]
		if !exists {
			values = append(values, cty.NullVal(elemType))
			continue
		}

		switch val {
		case nil:
			values = append(values, cty.NullVal(elemType))
		case "__cty_unknown__":
			values = append(values, cty.UnknownVal(elemType))
		default:
			ctyVal, err := c.convertValue(val, elemType)
			if err != nil {
				return cty.NilVal, fmt.Errorf("failed to convert list element %d: %w", i, err)
			}
			values = append(values, ctyVal)
		}
	}

	return cty.ListVal(values), nil
}

func (c *CtyConverter) convertSet(data map[string]any, ty cty.Type) (cty.Value, error) {
	elemType := ty.ElementType()

	// Handle empty set
	if len(data) == 0 {
		return cty.SetValEmpty(elemType), nil
	}

	var values []cty.Value

	for _, val := range data {
		switch val {
		case nil:
			values = append(values, cty.NullVal(elemType))
		case "__cty_unknown__":
			values = append(values, cty.UnknownVal(elemType))
		default:
			ctyVal, err := c.convertValue(val, elemType)
			if err != nil {
				return cty.NilVal, fmt.Errorf("failed to convert set element: %w", err)
			}
			values = append(values, ctyVal)
		}
	}

	return cty.SetVal(values), nil
}

func (c *CtyConverter) convertValue(val any, ty cty.Type) (cty.Value, error) {
	if val == nil {
		return cty.NullVal(ty), nil
	}

	if val == "__cty_unknown__" {
		return cty.UnknownVal(ty), nil
	}

	// Handle primitive types directly
	switch ty {
	case cty.String:
		return cty.StringVal(fmt.Sprintf("%v", val)), nil
	case cty.Number:
		switch v := val.(type) {
		case int:
			return cty.NumberIntVal(int64(v)), nil
		case int64:
			return cty.NumberIntVal(v), nil
		case float64:
			return cty.NumberFloatVal(v), nil
		default:
			return cty.NullVal(ty), fmt.Errorf("cannot convert %T to number", val)
		}
	case cty.Bool:
		if b, ok := val.(bool); ok {
			return cty.BoolVal(b), nil
		}
		return cty.NullVal(ty), fmt.Errorf("cannot convert %T to bool", val)
	}

	// Handle slices for collections
	if slice, ok := val.([]interface{}); ok {
		return c.convertSlice(slice, ty)
	}
	if slice, ok := val.([]any); ok {
		// Convert []any to []interface{} for compatibility
		interfaceSlice := make([]interface{}, len(slice))
		copy(interfaceSlice, slice)
		return c.convertSlice(interfaceSlice, ty)
	}

	// For complex types, convert to map and recurse
	if valMap, ok := val.(map[string]any); ok {
		return c.MapToCtyValue(valMap, ty)
	}

	return cty.NullVal(ty), fmt.Errorf("cannot convert %T to cty type %s", val, ty.FriendlyName())
}

func (c *CtyConverter) inferCtyValue(data map[string]any) cty.Value {
	if len(data) == 0 {
		return cty.NullVal(cty.DynamicPseudoType)
	}

	if len(data) == 1 && data["value"] != nil {
		// Single value - infer type
		val := data["value"]
		switch v := val.(type) {
		case string:
			return cty.StringVal(v)
		case int:
			return cty.NumberIntVal(int64(v))
		case int64:
			return cty.NumberIntVal(v)
		case float64:
			return cty.NumberFloatVal(v)
		case bool:
			return cty.BoolVal(v)
		default:
			return cty.StringVal(fmt.Sprintf("%v", v))
		}
	}

	// Multiple values - create object
	values := make(map[string]cty.Value)
	for key, val := range data {
		// Recursively convert each value without wrapping it in a map
		values[key] = c.convertValueToCtyValue(val)
	}

	attrTypes := make(map[string]cty.Type)
	for key, val := range values {
		attrTypes[key] = val.Type()
	}

	return cty.ObjectVal(values)
}

// IsUnknownValue checks if a value represents an unknown cty value
func (c *CtyConverter) IsUnknownValue(data any) bool {
	// Check for string marker
	if str, ok := data.(string); ok && str == "__cty_unknown__" {
		return true
	}

	// Check for object marker (backward compatibility)
	if dataMap, ok := data.(map[string]any); ok && dataMap != nil {
		unknown, exists := dataMap["__cty_unknown__"]
		return exists && unknown == true
	}

	return false
}

// CreateUnknownValue creates a value representing an unknown cty value
func (c *CtyConverter) CreateUnknownValue() any {
	return "__cty_unknown__"
}

// convertValueToCtyValue converts a single value to cty.Value without infinite recursion
func (c *CtyConverter) convertValueToCtyValue(val any) cty.Value {
	switch v := val.(type) {
	case string:
		return cty.StringVal(v)
	case int:
		return cty.NumberIntVal(int64(v))
	case int64:
		return cty.NumberIntVal(v)
	case float64:
		return cty.NumberFloatVal(v)
	case bool:
		return cty.BoolVal(v)
	case map[string]any:
		// For nested maps, use inferCtyValue (but only if it's not empty)
		if len(v) == 0 {
			return cty.NullVal(cty.DynamicPseudoType)
		}
		return c.inferCtyValue(v)
	case []any:
		// Convert slice to list
		if len(v) == 0 {
			return cty.ListValEmpty(cty.DynamicPseudoType)
		}
		values := make([]cty.Value, len(v))
		for i, item := range v {
			values[i] = c.convertValueToCtyValue(item)
		}
		return cty.ListVal(values)
	case nil:
		return cty.NullVal(cty.DynamicPseudoType)
	default:
		// Convert unknown types to string
		return cty.StringVal(fmt.Sprintf("%v", v))
	}
}

// convertSliceToCollection handles conversion of []any to list or set types
func (c *CtyConverter) convertSliceToCollection(slice []any, ty cty.Type) (cty.Value, error) {
	elemType := ty.ElementType()

	// Handle empty slice
	if len(slice) == 0 {
		if ty.IsListType() {
			return cty.ListValEmpty(elemType), nil
		} else if ty.IsSetType() {
			return cty.SetValEmpty(elemType), nil
		}
	}

	var values []cty.Value
	for i, item := range slice {
		ctyVal, err := c.convertValue(item, elemType)
		if err != nil {
			return cty.NilVal, fmt.Errorf("failed to convert slice element %d: %w", i, err)
		}
		values = append(values, ctyVal)
	}

	if ty.IsListType() {
		return cty.ListVal(values), nil
	} else if ty.IsSetType() {
		return cty.SetVal(values), nil
	}

	return cty.NilVal, fmt.Errorf("unsupported collection type: %s", ty.FriendlyName())
}

// convertSlice handles conversion of []interface{} to various collection types
func (c *CtyConverter) convertSlice(slice []interface{}, ty cty.Type) (cty.Value, error) {
	elemType := ty.ElementType()
	var values []cty.Value

	for i, item := range slice {
		ctyVal, err := c.convertValue(item, elemType)
		if err != nil {
			return cty.NilVal, fmt.Errorf("failed to convert slice element %d: %w", i, err)
		}
		values = append(values, ctyVal)
	}

	// Handle empty slices - sets cannot be empty, so use null values
	if len(values) == 0 {
		switch {
		case ty.IsListType():
			return cty.ListValEmpty(elemType), nil
		case ty.IsSetType():
			return cty.SetValEmpty(elemType), nil
		case ty.IsTupleType():
			return cty.TupleVal(values), nil // Empty tuple is allowed
		default:
			return cty.NullVal(ty), nil
		}
	}

	switch {
	case ty.IsListType():
		return cty.ListVal(values), nil
	case ty.IsSetType():
		return cty.SetVal(values), nil
	case ty.IsTupleType():
		return cty.TupleVal(values), nil
	default:
		return cty.NilVal, fmt.Errorf("cannot convert slice to %s", ty.FriendlyName())
	}
}
