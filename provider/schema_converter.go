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
	"maps"

	"github.com/apparentlymart/opentofu-providers/tofuprovider/providerschema"
	"github.com/spacelift-io/spacelift-intent/types"
	"github.com/zclconf/go-cty/cty"
)

type SchemaConverter struct{}

// opentofuSchemaToObjectType converts an opentofu-providers Schema to cty.Type
func (sc *SchemaConverter) opentofuSchemaToObjectType(schema providerschema.Schema) cty.Type {
	// Collect all attributes from the schema
	attrTypes := make(map[string]cty.Type)

	// Add attributes
	for attrName, attr := range maps.Collect(schema.Attributes()) {
		if nestedType := attr.NestedType(); nestedType != nil {
			// Handle nested object types properly
			nestedObjectType := sc.nestedObjectTypeToObjectType(nestedType)

			switch nestedType.Nesting() {
			case providerschema.NestingList:
				attrTypes[attrName] = cty.List(nestedObjectType)
			case providerschema.NestingSet:
				attrTypes[attrName] = cty.Set(nestedObjectType)
			case providerschema.NestingMap:
				attrTypes[attrName] = cty.Map(nestedObjectType)
			case providerschema.NestingSingle:
				attrTypes[attrName] = nestedObjectType
			default:
				attrTypes[attrName] = cty.DynamicPseudoType
			}
		} else {
			ctyType, err := attr.Type().AsCtyType()
			if err != nil {
				attrTypes[attrName] = cty.DynamicPseudoType
			} else {
				attrTypes[attrName] = ctyType
			}
		}
	}

	// Add nested block types - these are typically lists or sets of objects
	for blockName, blockType := range maps.Collect(schema.NestedBlockTypes()) {
		// Build nested object type recursively
		nestedObjectType := sc.blockTypeToObjectType(blockType)

		switch blockType.Nesting() {
		case providerschema.NestingList:
			attrTypes[blockName] = cty.List(nestedObjectType)
		case providerschema.NestingSet:
			attrTypes[blockName] = cty.Set(nestedObjectType)
		case providerschema.NestingMap:
			attrTypes[blockName] = cty.Map(nestedObjectType)
		case providerschema.NestingSingle, providerschema.NestingGroup:
			attrTypes[blockName] = nestedObjectType
		default:
			// Unknown nesting
			attrTypes[blockName] = cty.DynamicPseudoType
		}
	}

	if len(attrTypes) == 0 {
		return cty.DynamicPseudoType
	}

	return cty.Object(attrTypes)
}

// nestedObjectTypeToObjectType converts a nested object type to cty.Type
func (sc *SchemaConverter) nestedObjectTypeToObjectType(nestedType providerschema.ObjectType) cty.Type {
	attrTypes := make(map[string]cty.Type)

	for attrName, attr := range maps.Collect(nestedType.Attributes()) {
		if innerNestedType := attr.NestedType(); innerNestedType != nil {
			// Recursively handle nested types
			innerObjectType := sc.nestedObjectTypeToObjectType(innerNestedType)

			switch innerNestedType.Nesting() {
			case providerschema.NestingList:
				attrTypes[attrName] = cty.List(innerObjectType)
			case providerschema.NestingSet:
				attrTypes[attrName] = cty.Set(innerObjectType)
			case providerschema.NestingMap:
				attrTypes[attrName] = cty.Map(innerObjectType)
			case providerschema.NestingSingle:
				attrTypes[attrName] = innerObjectType
			default:
				attrTypes[attrName] = cty.DynamicPseudoType
			}
		} else {
			ctyType, err := attr.Type().AsCtyType()
			if err != nil {
				attrTypes[attrName] = cty.DynamicPseudoType
			} else {
				attrTypes[attrName] = ctyType
			}
		}
	}

	if len(attrTypes) == 0 {
		return cty.DynamicPseudoType
	}

	return cty.Object(attrTypes)
}

// blockTypeToObjectType converts a block type to cty.Type
func (sc *SchemaConverter) blockTypeToObjectType(blockType providerschema.NestedBlockType) cty.Type {
	attrTypes := make(map[string]cty.Type)

	// Handle attributes in the block type
	for attrName, attr := range maps.Collect(blockType.Attributes()) {
		if nestedType := attr.NestedType(); nestedType != nil {
			// Recursively handle nested types
			nestedObjectType := sc.nestedObjectTypeToObjectType(nestedType)

			switch nestedType.Nesting() {
			case providerschema.NestingList:
				attrTypes[attrName] = cty.List(nestedObjectType)
			case providerschema.NestingSet:
				attrTypes[attrName] = cty.Set(nestedObjectType)
			case providerschema.NestingMap:
				attrTypes[attrName] = cty.Map(nestedObjectType)
			case providerschema.NestingSingle:
				attrTypes[attrName] = nestedObjectType
			default:
				attrTypes[attrName] = cty.DynamicPseudoType
			}
		} else {
			ctyType, err := attr.Type().AsCtyType()
			if err != nil {
				attrTypes[attrName] = cty.DynamicPseudoType
			} else {
				attrTypes[attrName] = ctyType
			}
		}
	}

	// Handle nested block types recursively
	for nestedBlockName, nestedBlockType := range maps.Collect(blockType.NestedBlockTypes()) {
		nestedObjectType := sc.blockTypeToObjectType(nestedBlockType)

		switch nestedBlockType.Nesting() {
		case providerschema.NestingList:
			attrTypes[nestedBlockName] = cty.List(nestedObjectType)
		case providerschema.NestingSet:
			attrTypes[nestedBlockName] = cty.Set(nestedObjectType)
		case providerschema.NestingMap:
			attrTypes[nestedBlockName] = cty.Map(nestedObjectType)
		case providerschema.NestingSingle, providerschema.NestingGroup:
			attrTypes[nestedBlockName] = nestedObjectType
		default:
			attrTypes[nestedBlockName] = cty.DynamicPseudoType
		}
	}

	if len(attrTypes) == 0 {
		return cty.DynamicPseudoType
	}

	return cty.Object(attrTypes)
}

// convertSchemaToTypeDescription converts a providerschema.Schema to types.TypeDescription
func (sc *SchemaConverter) convertSchemaToTypeDescription(providerName, typeName string, schema providerschema.Schema, schemaType string) *types.TypeDescription {
	properties := make(map[string]any)
	required := []string{}

	// Extract attributes from schema
	for attrName, attr := range maps.Collect(schema.Attributes()) {
		attrInfo := map[string]any{
			"type": "string", // Default type
		}

		// Get the attribute type
		if attrType := attr.Type(); attrType != nil {
			if ctyType, err := attrType.AsCtyType(); err == nil {
				attrInfo["type"] = sc.ctyTypeToString(ctyType)
			}
		}

		// Check for nested types
		if nestedType := attr.NestedType(); nestedType != nil {
			attrInfo["nested"] = true
		}

		// Check if attribute is required
		usage := attr.Usage()
		if usage == providerschema.AttributeRequired {
			required = append(required, attrName)
			attrInfo["required"] = true
		} else {
			attrInfo["required"] = false
		}

		// Add usage information
		switch usage {
		case providerschema.AttributeRequired:
			attrInfo["usage"] = "required"
		case providerschema.AttributeOptional:
			attrInfo["usage"] = "optional"
		case providerschema.AttributeOptionalComputed:
			attrInfo["usage"] = "optional_computed"
		case providerschema.AttributeComputed:
			attrInfo["usage"] = "computed"
		default:
			attrInfo["usage"] = "unsupported"
		}

		// Add sensitive and deprecated flags
		attrInfo["sensitive"] = attr.IsSensitive()
		attrInfo["deprecated"] = attr.IsDeprecated()
		attrInfo["write_only"] = attr.IsWriteOnly()

		// Add description if available
		if desc, _ := attr.DocDescription(); desc != "" {
			attrInfo["description"] = desc
		}

		properties[attrName] = attrInfo
	}

	// Try to get the actual schema description by accessing the underlying implementation
	description := fmt.Sprintf("OpenTofu %s %s", schemaType, typeName)
	if schemaDescription, _ := schema.DocDescription(); schemaDescription != "" {
		description = schemaDescription
	}

	return &types.TypeDescription{
		ProviderName: providerName,
		Type:         typeName,
		Description:  description,
		Properties:   properties,
		Required:     required,
	}
}

// ctyTypeToString converts cty.Type to string representation
func (sc *SchemaConverter) ctyTypeToString(t cty.Type) string {
	switch {
	case t.Equals(cty.String):
		return "string"
	case t.Equals(cty.Number):
		return "number"
	case t.Equals(cty.Bool):
		return "boolean"
	case t.IsListType():
		return "list"
	case t.IsSetType():
		return "set"
	case t.IsMapType():
		return "map"
	case t.IsObjectType():
		return "object"
	default:
		return "unknown"
	}
}
