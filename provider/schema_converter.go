// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"maps"

	"github.com/apparentlymart/opentofu-providers/tofuprovider/providerschema"
	"github.com/zclconf/go-cty/cty"

	"github.com/spacelift-io/spacelift-intent/types"
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

// processBlockTypeInfo processes a NestedBlockType and returns its information map recursively
func (sc *SchemaConverter) processBlockTypeInfo(blockType providerschema.NestedBlockType) map[string]any {
	blockInfo := map[string]any{
		"is_block": true,
	}

	// Get nesting mode
	nesting := blockType.Nesting()
	switch nesting {
	case providerschema.NestingList:
		blockInfo["nesting"] = "list"
		blockInfo["type"] = "list"
	case providerschema.NestingSet:
		blockInfo["nesting"] = "set"
		blockInfo["type"] = "set"
	case providerschema.NestingMap:
		blockInfo["nesting"] = "map"
		blockInfo["type"] = "map"
	case providerschema.NestingSingle:
		blockInfo["nesting"] = "single"
		blockInfo["type"] = "object"
	case providerschema.NestingGroup:
		blockInfo["nesting"] = "group"
		blockInfo["type"] = "object"
	default:
		blockInfo["nesting"] = "unknown"
		blockInfo["type"] = "unknown"
	}

	// Get min and max items constraints
	minItems, maxItems := blockType.ItemLimits()
	blockInfo["min_items"] = minItems
	blockInfo["max_items"] = maxItems

	// Extract nested attributes from the block
	nestedProperties := make(map[string]any)
	nestedRequired := []string{}

	for nestedAttrName, nestedAttr := range maps.Collect(blockType.Attributes()) {
		nestedAttrInfo := map[string]any{
			"type": "string", // Default type
		}

		// Get the attribute type
		if attrType := nestedAttr.Type(); attrType != nil {
			if ctyType, err := attrType.AsCtyType(); err == nil {
				nestedAttrInfo["type"] = sc.ctyTypeToString(ctyType)
			}
		}

		// Check for nested types within the block attribute
		if nestedType := nestedAttr.NestedType(); nestedType != nil {
			nestedAttrInfo["nested"] = true
		}

		// Check if attribute is required
		usage := nestedAttr.Usage()
		if usage == providerschema.AttributeRequired {
			nestedRequired = append(nestedRequired, nestedAttrName)
			nestedAttrInfo["required"] = true
		} else {
			nestedAttrInfo["required"] = false
		}

		// Add usage information
		switch usage {
		case providerschema.AttributeRequired:
			nestedAttrInfo["usage"] = "required"
		case providerschema.AttributeOptional:
			nestedAttrInfo["usage"] = "optional"
		case providerschema.AttributeOptionalComputed:
			nestedAttrInfo["usage"] = "optional_computed"
		case providerschema.AttributeComputed:
			nestedAttrInfo["usage"] = "computed"
		default:
			nestedAttrInfo["usage"] = "unsupported"
		}

		// Add sensitive and deprecated flags
		nestedAttrInfo["sensitive"] = nestedAttr.IsSensitive()
		nestedAttrInfo["deprecated"] = nestedAttr.IsDeprecated()
		nestedAttrInfo["write_only"] = nestedAttr.IsWriteOnly()

		// Add description if available
		if desc, _ := nestedAttr.DocDescription(); desc != "" {
			nestedAttrInfo["description"] = desc
		}

		nestedProperties[nestedAttrName] = nestedAttrInfo
	}

	// Add nested properties to block info
	if len(nestedProperties) > 0 {
		blockInfo["properties"] = nestedProperties
	}
	if len(nestedRequired) > 0 {
		blockInfo["required"] = nestedRequired
	}

	// Recursively process nested blocks
	nestedBlocks := make(map[string]any)
	for nestedBlockName, nestedBlockType := range maps.Collect(blockType.NestedBlockTypes()) {
		nestedBlocks[nestedBlockName] = sc.processBlockTypeInfo(nestedBlockType)
	}
	if len(nestedBlocks) > 0 {
		blockInfo["nested_blocks"] = nestedBlocks
	}

	return blockInfo
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

	// Extract nested block types from schema (recursively)
	for blockName, blockType := range maps.Collect(schema.NestedBlockTypes()) {
		properties[blockName] = sc.processBlockTypeInfo(blockType)
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
