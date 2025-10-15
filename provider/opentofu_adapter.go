// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

//go:build !legacy_plugin

package provider

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/apparentlymart/opentofu-providers/tofuprovider"
	"github.com/apparentlymart/opentofu-providers/tofuprovider/providerops"
	"github.com/apparentlymart/opentofu-providers/tofuprovider/providerschema"
	"github.com/zclconf/go-cty/cty"

	"github.com/spacelift-io/spacelift-intent/types"
)

// NewOpenTofuAdapter creates a new adapter using the opentofu-providers library
func NewOpenTofuAdapter(tmpDir string, registry types.RegistryClient) types.ProviderManager {
	return &OpenTofuAdapter{
		tmpDir:          tmpDir,
		registry:        registry,
		providers:       make(map[string]tofuprovider.GRPCPluginProvider),
		schemas:         make(map[string]*types.ProviderSchema),
		rawSchemas:      make(map[string]providerops.GetProviderSchemaResponse),
		converter:       &CtyConverter{},
		schemaConverter: &SchemaConverter{},
		binaries:        make(map[string]string),
	}
}

// OpenTofuAdapter implements ProviderAdapter using the opentofu-providers library
type OpenTofuAdapter struct {
	tmpDir          string
	registry        types.RegistryClient
	providers       map[string]tofuprovider.GRPCPluginProvider
	schemas         map[string]*types.ProviderSchema
	rawSchemas      map[string]providerops.GetProviderSchemaResponse // provider key -> raw schema response
	converter       *CtyConverter
	schemaConverter *SchemaConverter
	binaries        map[string]string // provider key -> binary path
}

func (a *OpenTofuAdapter) GetProviderVersions(ctx context.Context, provider types.ProviderConfig) ([]types.ProviderVersionInfo, error) {
	return a.registry.GetProviderVersions(ctx, provider)
}

func (a *OpenTofuAdapter) LoadProvider(ctx context.Context, providerConfig *types.ProviderConfig) error {
	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	// Check if already loaded
	if _, exists := a.providers[cacheKey]; exists {
		return nil
	}

	provider, schema, rawSchema, err := a.loadProvider(ctx, providerConfig)
	if err != nil {
		return err
	}

	if err := a.configureProvider(ctx, providerConfig, provider, rawSchema); err != nil {
		provider.Close()
		return err
	}

	a.providers[cacheKey] = provider
	a.schemas[cacheKey] = schema
	a.rawSchemas[cacheKey] = rawSchema

	return nil
}

func (a *OpenTofuAdapter) configureProvider(ctx context.Context, providerConfig *types.ProviderConfig, provider tofuprovider.GRPCPluginProvider, schemaResponse providerops.GetProviderSchemaResponse) error {
	configCandidates := []func(providerops.GetProviderSchemaResponse) (providerschema.DynamicValueIn, error){
		a.emptyConfig,               // first, try empty config
		a.configWithEmptyProperties, // second, try config with all properties set to undefined (null or empty)
		/// ^^^ "hashicorp/azurerm" requires "features" block to be present, even if empty
	}

	finalErr := errors.New("no configuration method succeeded")

	for _, configCandidate := range configCandidates {
		config, err := configCandidate(schemaResponse)
		if err != nil {
			finalErr = err
			continue
		}

		configReq := &providerops.ConfigureProviderRequest{
			Config:           config,
			TerraformVersion: "1.6.0",
		}

		configResp, err := provider.ConfigureProvider(ctx, configReq)
		if err != nil {
			finalErr = fmt.Errorf("failed to configure provider %s: %w", providerConfig.Name, err)
			continue
		}

		if configResp.Diagnostics().HasErrors() {
			finalErr = fmt.Errorf("provider configuration failed: %s", a.formatDiagnostics(configResp.Diagnostics()))
			continue
		}

		return nil
	}

	return finalErr
}

func (a *OpenTofuAdapter) emptyConfig(_ providerops.GetProviderSchemaResponse) (providerschema.DynamicValueIn, error) {
	emptyConfig, err := a.converter.MapToCtyValue(nil, cty.DynamicPseudoType)
	if err != nil {
		return providerschema.DynamicValueIn{}, fmt.Errorf("failed to encode empty config: %w", err)
	}

	return providerschema.NewDynamicValue(emptyConfig, cty.DynamicPseudoType), nil
}

func (a *OpenTofuAdapter) configWithEmptyProperties(schemaResponse providerops.GetProviderSchemaResponse) (providerschema.DynamicValueIn, error) {
	sh := schemaResponse.ProviderSchema().ProviderConfigSchema()
	attrs := maps.Collect(sh.Attributes())
	nested := maps.Collect(sh.NestedBlockTypes())

	types := map[string]cty.Type{}
	for name := range attrs {
		types[name] = cty.DynamicPseudoType
	}
	for name := range nested {
		types[name] = cty.DynamicPseudoType
	}
	values := map[string]cty.Value{}
	for name := range attrs {
		values[name] = cty.NullVal(cty.DynamicPseudoType)
	}
	for name := range nested {
		value, err := a.converter.MapToCtyValue(nil, cty.DynamicPseudoType)
		if err != nil {
			return providerschema.DynamicValueIn{}, fmt.Errorf("failed to map nested value for %s: %w", name, err)
		}
		values[name] = value
	}

	return providerschema.NewDynamicValue(cty.ObjectVal(values), cty.Object(types)), nil
}

func (a *OpenTofuAdapter) loadProvider(ctx context.Context, providerConfig *types.ProviderConfig) (tofuprovider.GRPCPluginProvider, *types.ProviderSchema, providerops.GetProviderSchemaResponse, error) {
	// Parse provider name
	parts := strings.Split(providerConfig.Name, "/")
	if len(parts) != 2 {
		return nil, nil, nil, fmt.Errorf("invalid provider name format, expected 'namespace/type'")
	}

	// Download provider if needed
	binary, err := a.downloadProvider(ctx, providerConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to download provider %s: %w", providerConfig.Name, err)
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	a.binaries[cacheKey] = binary

	// Start the provider using the opentofu-providers library
	provider, err := tofuprovider.StartGRPCPlugin(ctx, binary)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start provider %s: %w", providerConfig.Name, err)
	}

	schema, rawSchema, err := a.getSchemaWithProvider(ctx, providerConfig, provider)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get provider schema for %s: %w", providerConfig.Name, err)
	}

	return provider, schema, rawSchema, nil
}

func (a *OpenTofuAdapter) DescribeProvider(ctx context.Context, providerConfig *types.ProviderConfig) (*types.ProviderSchema, *string, error) {
	provider, schema, rawSchema, err := a.loadProvider(ctx, providerConfig)
	if err != nil {
		return nil, nil, err
	}

	defer provider.Close()

	var configureError *string
	if err := a.configureProvider(ctx, providerConfig, provider, rawSchema); err != nil {
		message := err.Error()
		configureError = &message
	}

	return schema, configureError, nil
}

// getSchema is to be called only once after the provider is loaded
func (a *OpenTofuAdapter) getSchemaWithProvider(ctx context.Context, providerConfig *types.ProviderConfig, provider tofuprovider.GRPCPluginProvider) (*types.ProviderSchema, providerops.GetProviderSchemaResponse, error) {
	// Get raw schema response (this will cache it)
	schemaResp, err := a.getRawSchemaWithProvider(ctx, providerConfig, provider)
	if err != nil {
		return nil, nil, err
	}

	schema := &types.ProviderSchema{
		Provider: &types.TypeDescription{
			ProviderName: providerConfig.Name,
		},
		Resources:   make(map[string]*types.TypeDescription),
		DataSources: make(map[string]*types.TypeDescription),
	}

	providerSchema := schemaResp.ProviderSchema()

	config := providerSchema.ProviderConfigSchema()
	schema.Provider = a.schemaConverter.convertSchemaToTypeDescription(providerConfig.Name, "config", config, "provider")

	// Convert resource schemas
	resourceSchemas := maps.Collect(providerSchema.ManagedResourceTypeSchemas())
	for resourceType, resourceSchema := range resourceSchemas {
		desc := a.schemaConverter.convertSchemaToTypeDescription(providerConfig.Name, resourceType, resourceSchema, "resource")
		schema.Resources[resourceType] = desc
	}

	// Convert data source schemas
	dataSourceSchemas := maps.Collect(providerSchema.DataResourceTypeSchemas())
	for dataSourceType, dataSourceSchema := range dataSourceSchemas {
		desc := a.schemaConverter.convertSchemaToTypeDescription(providerConfig.Name, dataSourceType, dataSourceSchema, "data_source")
		schema.DataSources[dataSourceType] = desc
	}

	return schema, schemaResp, nil
}

// getRawSchema gets or caches the raw provider schema response
func (a *OpenTofuAdapter) getRawSchema(_ context.Context, providerConfig *types.ProviderConfig) (providerops.GetProviderSchemaResponse, error) {
	// Check cache
	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	if rawSchema, exists := a.rawSchemas[cacheKey]; exists {
		return rawSchema, nil
	}

	return nil, errors.New("provider schema not cached, provider must be loaded first")
}

// getRawSchema gets or caches the raw provider schema response
func (a *OpenTofuAdapter) getRawSchemaWithProvider(ctx context.Context, providerConfig *types.ProviderConfig, provider tofuprovider.GRPCPluginProvider) (providerops.GetProviderSchemaResponse, error) {
	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	// Get provider schema
	schemaReq := &providerops.GetProviderSchemaRequest{}
	schemaResp, err := provider.GetProviderSchema(ctx, schemaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider schema: %w", err)
	}

	if schemaResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("schema request failed: %s", a.formatDiagnostics(schemaResp.Diagnostics()))
	}

	// Cache the raw schema
	a.rawSchemas[cacheKey] = schemaResp

	return schemaResp, nil
}

// getOpentofuResourceSchema gets the raw opentofu-providers Schema for a resource type
func (a *OpenTofuAdapter) getOpentofuResourceSchema(ctx context.Context, providerConfig *types.ProviderConfig, resourceType string) (providerschema.Schema, error) {
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	// Get cached raw schema response
	schemaResp, err := a.getRawSchema(ctx, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider schema: %w", err)
	}

	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())
	opentofuSchema, exists := resourceSchemas[resourceType]
	if !exists {
		return nil, fmt.Errorf("resource type %s not found in opentofu schema", resourceType)
	}

	return opentofuSchema, nil
}

// getOpentofuDataSourceSchema gets the raw opentofu-providers Schema for a data source type
func (a *OpenTofuAdapter) getOpentofuDataSourceSchema(ctx context.Context, providerConfig *types.ProviderConfig, dataSourceType string) (providerschema.Schema, error) {
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	// Get cached raw schema response
	schemaResp, err := a.getRawSchema(ctx, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider schema: %w", err)
	}

	dataSourceSchemas := maps.Collect(schemaResp.ProviderSchema().DataResourceTypeSchemas())

	// Check if provider has any data sources first
	if len(dataSourceSchemas) == 0 {
		return nil, fmt.Errorf("provider has no data source schemas")
	}

	opentofuSchema, exists := dataSourceSchemas[dataSourceType]
	if !exists {
		return nil, fmt.Errorf("data source type %s not found in opentofu schema", dataSourceType)
	}

	return opentofuSchema, nil
}

func (a *OpenTofuAdapter) PlanResource(ctx context.Context, providerConfig *types.ProviderConfig, resourceType string, currentState *map[string]any, newConfig map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	provider := a.providers[cacheKey]
	schema := a.schemas[cacheKey]

	_, exists := schema.Resources[resourceType]
	if !exists {
		return nil, fmt.Errorf("resource type %s not found", resourceType)
	}

	// Get the opentofu-providers schema directly for proper type conversion
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerConfig, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceTypeCty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert current state to cty.Value
	var priorState providerschema.DynamicValueIn
	if currentState != nil && len(*currentState) > 0 {
		priorStateCty, err := a.converter.MapToCtyValue(*currentState, resourceTypeCty)
		if err != nil {
			return nil, fmt.Errorf("failed to convert prior state: %w", err)
		}
		priorState = providerschema.NewDynamicValue(priorStateCty, resourceTypeCty)
	} else {
		// For new resources, use null value of the proper type instead of NoDynamicValue
		nullStateCty := cty.NullVal(resourceTypeCty)
		priorState = providerschema.NewDynamicValue(nullStateCty, resourceTypeCty)
	}

	// Convert new config to cty.Value
	configCty, err := a.converter.MapToCtyValue(newConfig, resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	config := providerschema.NewDynamicValue(configCty, resourceTypeCty)

	// Create plan request
	planReq := &providerops.PlanManagedResourceChangeRequest{
		ResourceType:     resourceType,
		PriorState:       priorState,
		Config:           config,
		ProposedNewState: config, // Use config as proposed state for simplicity
	}

	// Plan the resource
	planResp, err := provider.PlanManagedResourceChange(ctx, planReq)
	if err != nil {
		return nil, fmt.Errorf("plan failed: %w", err)
	}

	if planResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("plan failed: %s", a.formatDiagnostics(planResp.Diagnostics()))
	}

	// Convert planned state back to map
	plannedStateCty, err := planResp.PlannedNewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned state: %w", err)
	}

	plannedStateMap, err := a.converter.CtyValueToMap(plannedStateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert planned state to map: %w", err)
	}

	return plannedStateMap, nil
}

func (a *OpenTofuAdapter) CreateResource(ctx context.Context, providerConfig *types.ProviderConfig, resourceType string, config map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	provider := a.providers[cacheKey]

	// Get resource schema
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerConfig, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceTypeCty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert config to cty.Value
	configCty, err := a.converter.MapToCtyValue(config, resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	configDV := providerschema.NewDynamicValue(configCty, resourceTypeCty)

	// For new resources, use null value as prior state
	nullStateCty := cty.NullVal(resourceTypeCty)
	priorState := providerschema.NewDynamicValue(nullStateCty, resourceTypeCty)

	// First plan the resource
	planReq := &providerops.PlanManagedResourceChangeRequest{
		ResourceType:     resourceType,
		PriorState:       priorState,
		Config:           configDV,
		ProposedNewState: configDV,
	}

	planResp, err := provider.PlanManagedResourceChange(ctx, planReq)
	if err != nil {
		return nil, fmt.Errorf("plan failed: %w", err)
	}

	if planResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("plan failed: %s", a.formatDiagnostics(planResp.Diagnostics()))
	}

	// Convert planned state from DynamicValueOut to DynamicValueIn
	plannedStateCty, err := planResp.PlannedNewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned state: %w", err)
	}
	plannedStateDV := providerschema.NewDynamicValue(plannedStateCty, resourceTypeCty)

	// Now apply the resource
	applyReq := &providerops.ApplyManagedResourceChangeRequest{
		ResourceType:            resourceType,
		PriorState:              priorState,
		Config:                  configDV,
		PlannedNewState:         plannedStateDV,
		PlannedProviderInternal: planResp.PlannedProviderInternal(),
	}

	applyResp, err := provider.ApplyManagedResourceChange(ctx, applyReq)
	if err != nil {
		return nil, fmt.Errorf("apply failed: %w", err)
	}

	if applyResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("apply failed: %s", a.formatDiagnostics(applyResp.Diagnostics()))
	}

	// Convert final state back to map
	finalStateCty, err := applyResp.PlannedNewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode final state: %w", err)
	}

	finalStateMap, err := a.converter.CtyValueToMap(finalStateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert final state to map: %w", err)
	}

	return finalStateMap, nil
}

func (a *OpenTofuAdapter) DeleteResource(ctx context.Context, providerConfig *types.ProviderConfig, resourceType string, state map[string]any) error {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	provider := a.providers[cacheKey]

	// Get resource schema
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerConfig, resourceType)
	if err != nil {
		return fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceTypeCty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert current state to cty.Value
	currentStateCty, err := a.converter.MapToCtyValue(state, resourceTypeCty)
	if err != nil {
		return fmt.Errorf("failed to convert current state: %w", err)
	}
	priorState := providerschema.NewDynamicValue(currentStateCty, resourceTypeCty)

	// For deletion, use null value as config and proposed new state
	nullStateCty := cty.NullVal(resourceTypeCty)
	emptyConfig := providerschema.NewDynamicValue(nullStateCty, resourceTypeCty)
	proposedNewState := providerschema.NewDynamicValue(nullStateCty, resourceTypeCty)

	// First plan the deletion
	planReq := &providerops.PlanManagedResourceChangeRequest{
		ResourceType:     resourceType,
		PriorState:       priorState,
		Config:           emptyConfig,
		ProposedNewState: proposedNewState,
	}

	planResp, err := provider.PlanManagedResourceChange(ctx, planReq)
	if err != nil {
		return fmt.Errorf("plan deletion failed: %w", err)
	}

	if planResp.Diagnostics().HasErrors() {
		return fmt.Errorf("plan deletion failed: %s", a.formatDiagnostics(planResp.Diagnostics()))
	}

	// Convert planned state from DynamicValueOut to DynamicValueIn
	plannedStateCty, err := planResp.PlannedNewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return fmt.Errorf("failed to decode planned deletion state: %w", err)
	}
	plannedStateDV := providerschema.NewDynamicValue(plannedStateCty, resourceTypeCty)

	// Now apply the deletion
	applyReq := &providerops.ApplyManagedResourceChangeRequest{
		ResourceType:            resourceType,
		PriorState:              priorState,
		Config:                  emptyConfig,
		PlannedNewState:         plannedStateDV,
		PlannedProviderInternal: planResp.PlannedProviderInternal(),
	}

	applyResp, err := provider.ApplyManagedResourceChange(ctx, applyReq)
	if err != nil {
		return fmt.Errorf("apply deletion failed: %w", err)
	}

	if applyResp.Diagnostics().HasErrors() {
		return fmt.Errorf("apply deletion failed: %s", a.formatDiagnostics(applyResp.Diagnostics()))
	}

	return nil
}

func (a *OpenTofuAdapter) ReadDataSource(ctx context.Context, providerConfig *types.ProviderConfig, dataSourceType string, config map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	provider := a.providers[cacheKey]

	// Get data source schema for proper type information
	opentofuSchema, err := a.getOpentofuDataSourceSchema(ctx, providerConfig, dataSourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	dataSourceTypeCty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert config to cty.Value
	configCty, err := a.converter.MapToCtyValue(config, dataSourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	configDV := providerschema.NewDynamicValue(configCty, dataSourceTypeCty)

	// Create read request
	readReq := &providerops.ReadDataResourceRequest{
		ResourceType: dataSourceType,
		Config:       configDV,
		ProviderMeta: configDV,
	}

	// Read the data source
	readResp, err := provider.ReadDataResource(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("read data source failed: %w", err)
	}

	if readResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("read data source failed: %s", a.formatDiagnostics(readResp.Diagnostics()))
	}

	// Convert state back to map
	stateCty, err := readResp.State().AsCtyValue(dataSourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	stateMap, err := a.converter.CtyValueToMap(stateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert state to map: %w", err)
	}

	return stateMap, nil
}

func (a *OpenTofuAdapter) ImportResource(ctx context.Context, providerConfig *types.ProviderConfig, resourceType, importID string) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	provider := a.providers[cacheKey]

	// Get resource schema for proper type information
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerConfig, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceTypeCty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Create import request
	importReq := &providerops.ImportManagedResourceStateRequest{
		ResourceType: resourceType,
		ID:           importID,
	}

	// Import the resource
	importResp, err := provider.ImportManagedResourceState(ctx, importReq)
	if err != nil {
		return nil, fmt.Errorf("import failed: %w", err)
	}

	if importResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("import failed: %s", a.formatDiagnostics(importResp.Diagnostics()))
	}

	// Get the first imported resource state (providers can return multiple)
	importedResourcesIter := importResp.ImportedResources()
	var firstResource providerops.ImportedManagedResource
	var found bool

	// Get the first resource from the iterator
	for resource := range importedResourcesIter {
		firstResource = resource
		found = true
		break
	}

	if !found {
		return nil, fmt.Errorf("no resources were imported")
	}

	// Now read the resource to get the complete current state, following the same pattern as legacy manager
	// This ensures we capture the full resource data, not just what import returned

	// First convert the imported state from DynamicValueOut to DynamicValueIn
	importedStateCty, err := firstResource.State().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode imported state: %w", err)
	}
	importedStateDV := providerschema.NewDynamicValue(importedStateCty, resourceTypeCty)

	readReq := &providerops.ReadManagedResourceRequest{
		ResourceType:     resourceType,
		CurrentState:     importedStateDV,
		ProviderInternal: firstResource.ProviderInternal(),
	}

	readResp, err := provider.ReadManagedResource(ctx, readReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read imported resource: %w", err)
	}

	if readResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("read after import failed: %s", a.formatDiagnostics(readResp.Diagnostics()))
	}

	// Convert the complete current state back to map
	stateCty, err := readResp.NewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode current state: %w", err)
	}

	stateMap, err := a.converter.CtyValueToMap(stateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert current state to map: %w", err)
	}

	return stateMap, nil
}

func (a *OpenTofuAdapter) RefreshResource(ctx context.Context, providerConfig *types.ProviderConfig, resourceType string, currentState map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	provider := a.providers[cacheKey]

	// Get resource schema
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerConfig, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceTypeCty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert current state to cty.Value
	currentStateCty, err := a.converter.MapToCtyValue(currentState, resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert current state: %w", err)
	}
	currentStateDV := providerschema.NewDynamicValue(currentStateCty, resourceTypeCty)

	// Create refresh request - for OpenTofu, refresh is typically done through ReadManagedResource
	refreshReq := &providerops.ReadManagedResourceRequest{
		ResourceType:     resourceType,
		CurrentState:     currentStateDV,
		ProviderInternal: nil, // No provider internal data for basic refresh
	}

	// Refresh the resource
	refreshResp, err := provider.ReadManagedResource(ctx, refreshReq)
	if err != nil {
		return nil, fmt.Errorf("refresh failed: %w", err)
	}

	if refreshResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("refresh failed: %s", a.formatDiagnostics(refreshResp.Diagnostics()))
	}

	// Convert refreshed state back to map
	refreshedStateCty, err := refreshResp.NewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode refreshed state: %w", err)
	}

	refreshedStateMap, err := a.converter.CtyValueToMap(refreshedStateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert refreshed state to map: %w", err)
	}

	return refreshedStateMap, nil
}

// GetProviderVersion returns the version of the provider - not implemented
func (a *OpenTofuAdapter) GetProviderVersion(ctx context.Context, providerName string) (string, error) {
	return "", fmt.Errorf("GetProviderVersion not implemented")
}

// UpdateResource updates an existing resource
func (a *OpenTofuAdapter) UpdateResource(ctx context.Context, providerConfig *types.ProviderConfig, resourceType string, currentState, newConfig map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	provider := a.providers[cacheKey]

	// Get resource schema
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerConfig, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceTypeCty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert current state to cty.Value
	currentStateCty, err := a.converter.MapToCtyValue(currentState, resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert current state: %w", err)
	}
	priorState := providerschema.NewDynamicValue(currentStateCty, resourceTypeCty)

	// Convert new config to cty.Value
	configCty, err := a.converter.MapToCtyValue(newConfig, resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	configDV := providerschema.NewDynamicValue(configCty, resourceTypeCty)

	// Plan the update
	planReq := &providerops.PlanManagedResourceChangeRequest{
		ResourceType:     resourceType,
		PriorState:       priorState,
		Config:           configDV,
		ProposedNewState: configDV, // Use config as proposed state for simplicity
	}

	planResp, err := provider.PlanManagedResourceChange(ctx, planReq)
	if err != nil {
		return nil, fmt.Errorf("plan update failed: %w", err)
	}

	if planResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("plan update failed: %s", a.formatDiagnostics(planResp.Diagnostics()))
	}

	// Convert planned state from DynamicValueOut to DynamicValueIn
	plannedStateCty, err := planResp.PlannedNewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned state: %w", err)
	}
	plannedStateDV := providerschema.NewDynamicValue(plannedStateCty, resourceTypeCty)

	// Now apply the update
	applyReq := &providerops.ApplyManagedResourceChangeRequest{
		ResourceType:            resourceType,
		PriorState:              priorState,
		Config:                  configDV,
		PlannedNewState:         plannedStateDV,
		PlannedProviderInternal: planResp.PlannedProviderInternal(),
	}

	applyResp, err := provider.ApplyManagedResourceChange(ctx, applyReq)
	if err != nil {
		return nil, fmt.Errorf("apply update failed: %w", err)
	}

	if applyResp.Diagnostics().HasErrors() {
		return nil, fmt.Errorf("apply update failed: %s", a.formatDiagnostics(applyResp.Diagnostics()))
	}

	// Convert final state back to map
	finalStateCty, err := applyResp.PlannedNewState().AsCtyValue(resourceTypeCty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode final state: %w", err)
	}

	finalStateMap, err := a.converter.CtyValueToMap(finalStateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert final state to map: %w", err)
	}

	return finalStateMap, nil
}

// ListResources lists all available resource types for a provider
func (a *OpenTofuAdapter) ListResources(ctx context.Context, providerConfig *types.ProviderConfig) ([]string, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	schema, exists := a.schemas[cacheKey]
	if !exists {
		return nil, fmt.Errorf("provider schema not found for %s", providerConfig.Name)
	}

	resourcesTypes := make([]string, 0, len(schema.Resources))
	for resourceType := range schema.Resources {
		resourcesTypes = append(resourcesTypes, resourceType)
	}

	return resourcesTypes, nil
}

// DescribeResource returns detailed information about a resource type
func (a *OpenTofuAdapter) DescribeResource(ctx context.Context, providerConfig *types.ProviderConfig, resourceType string) (*types.TypeDescription, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	schema, exists := a.schemas[cacheKey]
	if !exists {
		return nil, fmt.Errorf("provider schema not found for %s", providerConfig.Name)
	}

	resourceDescription, exists := schema.Resources[resourceType]
	if !exists {
		return nil, fmt.Errorf("resource type %s not found in provider %s", resourceType, providerConfig.Name)
	}

	return resourceDescription, nil
}

// DescribeDataSource returns detailed information about a data source type
func (a *OpenTofuAdapter) DescribeDataSource(ctx context.Context, providerConfig *types.ProviderConfig, dataSourceType string) (*types.TypeDescription, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	schema, exists := a.schemas[cacheKey]
	if !exists {
		return nil, fmt.Errorf("provider schema not found for %s", providerConfig.Name)
	}

	dataSourceDescription, exists := schema.DataSources[dataSourceType]
	if !exists {
		return nil, fmt.Errorf("data source type %s not found in provider %s", dataSourceType, providerConfig.Name)
	}

	return dataSourceDescription, nil
}

// ListDataSources lists all available data source types for a provider
func (a *OpenTofuAdapter) ListDataSources(ctx context.Context, providerConfig *types.ProviderConfig) ([]string, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerConfig); err != nil {
		return nil, err
	}

	cacheKey, err := providerConfig.FullName()
	if err != nil {
		return nil, fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	schema, exists := a.schemas[cacheKey]
	if !exists {
		return nil, fmt.Errorf("provider schema not found for %s", providerConfig.Name)
	}

	dataSourceTypes := make([]string, 0, len(schema.DataSources))
	for dataSourceType := range schema.DataSources {
		dataSourceTypes = append(dataSourceTypes, dataSourceType)
	}

	return dataSourceTypes, nil
}

func (a *OpenTofuAdapter) Cleanup(ctx context.Context) {
	// Close all provider connections
	for _, provider := range a.providers {
		provider.Close()
	}

	// Clear caches
	a.providers = make(map[string]tofuprovider.GRPCPluginProvider)
	a.schemas = make(map[string]*types.ProviderSchema)
	a.rawSchemas = make(map[string]providerops.GetProviderSchemaResponse)
	a.binaries = make(map[string]string)
}

// Helper methods

// findProviderBinary finds the provider binary in a directory
func (a *OpenTofuAdapter) findProviderBinary(dir string) (string, error) {
	var binaryPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.Contains(info.Name(), "terraform-provider-") {
			binaryPath = path
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if binaryPath == "" {
		return "", errors.New("binary not found")
	}
	return binaryPath, nil
}

func (a *OpenTofuAdapter) downloadProvider(ctx context.Context, providerConfig *types.ProviderConfig) (string, error) {
	// Use the existing registry logic to download the provider
	downloadInfo, err := a.registry.GetProviderDownload(ctx, *providerConfig)
	if err != nil {
		return "", fmt.Errorf("failed to get provider download info: %w", err)
	}

	// Download and extract provider
	binaryPath, err := a.downloadAndExtractProvider(ctx, providerConfig, downloadInfo.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download provider: %w", err)
	}

	return binaryPath, nil
}

// downloadAndExtractProvider downloads and extracts a provider binary
func (a *OpenTofuAdapter) downloadAndExtractProvider(ctx context.Context, providerConfig *types.ProviderConfig, downloadURL string) (string, error) {
	// Create provider directory using name@version as key
	versionedName, err := providerConfig.FullName()
	if err != nil {
		return "", fmt.Errorf("failed to get versioned provider name: %w", err)
	}

	providerDir := filepath.Join(a.tmpDir, strings.ReplaceAll(versionedName, "/", "_"))
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create provider directory: %w", err)
	}

	// Check if binary already exists
	binaryPath, err := a.findProviderBinary(providerDir)
	if err == nil {
		// Binary exists, check if it's executable
		if info, err := os.Stat(binaryPath); err == nil && info.Mode()&0111 != 0 {
			return binaryPath, nil
		}
	}

	// Download zip file
	zipPath := filepath.Join(providerDir, "provider.zip")
	if err := a.downloadFile(ctx, downloadURL, zipPath); err != nil {
		return "", fmt.Errorf("failed to download provider: %w", err)
	}

	// Extract zip file
	if err := a.extractZip(zipPath, providerDir); err != nil {
		return "", fmt.Errorf("failed to extract provider: %w", err)
	}

	// Find and make binary executable
	binaryPath, err = a.findProviderBinary(providerDir)
	if err != nil {
		return "", fmt.Errorf("failed to find provider binary: %w", err)
	}

	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	return binaryPath, nil
}

func (a *OpenTofuAdapter) formatDiagnostics(diags providerops.Diagnostics) string {
	if !diags.HasErrors() {
		return "no errors"
	}

	var messages []string
	for diag := range diags.All() {
		if diag.Severity() == providerops.DiagnosticError {
			msg := diag.Summary()
			if detail := diag.Detail(); detail != "" {
				msg += ": " + detail
			}
			messages = append(messages, msg)
		}
	}

	return strings.Join(messages, "; ")
}

// downloadFile downloads a file from URL to local path
func (a *OpenTofuAdapter) downloadFile(ctx context.Context, url, path string) error {
	resp, err := a.registry.Download(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Close()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp)
	return err
}

// extractZip extracts a zip file to a destination directory
func (a *OpenTofuAdapter) extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			rc.Close()
			continue
		}

		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(file, rc)
		file.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
