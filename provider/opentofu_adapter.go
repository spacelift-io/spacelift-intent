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
	rawSchemas      map[string]providerops.GetProviderSchemaResponse // provider name -> raw schema response
	converter       *CtyConverter
	schemaConverter *SchemaConverter
	binaries        map[string]string // provider name -> binary path
}

func (a *OpenTofuAdapter) LoadProvider(ctx context.Context, providerName string) error {
	// Check if already loaded
	if _, exists := a.providers[providerName]; exists {
		return nil
	}

	// Parse provider name
	parts := strings.Split(providerName, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid provider name format, expected 'namespace/type'")
	}

	// Download provider if needed
	binary, err := a.downloadProvider(ctx, providerName)
	if err != nil {
		return fmt.Errorf("failed to download provider %s: %w", providerName, err)
	}
	a.binaries[providerName] = binary

	// Start the provider using the opentofu-providers library
	provider, err := tofuprovider.StartGRPCPlugin(ctx, binary)
	if err != nil {
		return fmt.Errorf("failed to start provider %s: %w", providerName, err)
	}

	emptyConfig, err := a.converter.MapToCtyValue(nil, cty.DynamicPseudoType)
	if err != nil {
		return fmt.Errorf("failed to encode empty config: %w", err)
	}

	// Configure the provider
	configReq := &providerops.ConfigureProviderRequest{
		Config:           providerschema.NewDynamicValue(emptyConfig, cty.DynamicPseudoType),
		TerraformVersion: "1.6.0",
	}

	configResp, err := provider.ConfigureProvider(ctx, configReq)
	if err != nil {
		provider.Close()
		return fmt.Errorf("failed to configure provider %s: %w", providerName, err)
	}

	// Check for configuration errors
	if configResp.Diagnostics().HasErrors() {
		provider.Close()
		return fmt.Errorf("provider configuration failed: %s", a.formatDiagnostics(configResp.Diagnostics()))
	}

	a.providers[providerName] = provider

	// Call GetSchema here as some providers might get confused if we call ConfigureProvider without calling GetSchema first
	schema, err := a.getSchema(ctx, providerName)
	if err != nil {
		return fmt.Errorf("failed to get provider schema for %s: %w", providerName, err)
	}

	// Store the provider, binary path
	a.schemas[providerName] = schema

	return nil
}

// getSchema is to be called only once after the provider is loaded
func (a *OpenTofuAdapter) getSchema(ctx context.Context, providerName string) (*types.ProviderSchema, error) {
	// Check cache
	if schema, exists := a.schemas[providerName]; exists {
		return schema, nil
	}

	// Get raw schema response (this will cache it)
	schemaResp, err := a.getRawSchema(ctx, providerName)
	if err != nil {
		return nil, err
	}

	schema := &types.ProviderSchema{
		Resources:   make(map[string]*types.TypeDescription),
		DataSources: make(map[string]*types.TypeDescription),
	}

	providerSchema := schemaResp.ProviderSchema()

	// Convert resource schemas
	resourceSchemas := maps.Collect(providerSchema.ManagedResourceTypeSchemas())
	for resourceType, resourceSchema := range resourceSchemas {
		desc, err := a.schemaConverter.convertSchemaToTypeDescription(providerName, resourceType, resourceSchema, "resource")
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource schema %s: %w", resourceType, err)
		}
		schema.Resources[resourceType] = desc
	}

	// Convert data source schemas
	dataSourceSchemas := maps.Collect(providerSchema.DataResourceTypeSchemas())
	for dataSourceType, dataSourceSchema := range dataSourceSchemas {
		desc, err := a.schemaConverter.convertSchemaToTypeDescription(providerName, dataSourceType, dataSourceSchema, "data_source")
		if err != nil {
			return nil, fmt.Errorf("failed to convert data source schema %s: %w", dataSourceType, err)
		}
		schema.DataSources[dataSourceType] = desc
	}

	// Cache the schema
	a.schemas[providerName] = schema

	return schema, nil
}

// getRawSchema gets or caches the raw provider schema response
func (a *OpenTofuAdapter) getRawSchema(ctx context.Context, providerName string) (providerops.GetProviderSchemaResponse, error) {
	// Check cache
	if rawSchema, exists := a.rawSchemas[providerName]; exists {
		return rawSchema, nil
	}

	provider := a.providers[providerName]

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
	a.rawSchemas[providerName] = schemaResp

	return schemaResp, nil
}

// getOpentofuResourceSchema gets the raw opentofu-providers Schema for a resource type
func (a *OpenTofuAdapter) getOpentofuResourceSchema(ctx context.Context, providerName, resourceType string) (providerschema.Schema, error) {
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	// Get cached raw schema response
	schemaResp, err := a.getRawSchema(ctx, providerName)
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
func (a *OpenTofuAdapter) getOpentofuDataSourceSchema(ctx context.Context, providerName, dataSourceType string) (providerschema.Schema, error) {
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	// Get cached raw schema response
	schemaResp, err := a.getRawSchema(ctx, providerName)
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

func (a *OpenTofuAdapter) PlanResource(ctx context.Context, providerName, resourceType string, currentState *map[string]any, newConfig map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	provider := a.providers[providerName]

	// Get resource schema to determine the cty.Type
	schema, err := a.getSchema(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	_, exists := schema.Resources[resourceType]
	if !exists {
		return nil, fmt.Errorf("resource type %s not found", resourceType)
	}

	// Get the opentofu-providers schema directly for proper type conversion
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerName, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceType_cty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert current state to cty.Value
	var priorState providerschema.DynamicValueIn
	if currentState != nil && len(*currentState) > 0 {
		priorStateCty, err := a.converter.MapToCtyValue(*currentState, resourceType_cty)
		if err != nil {
			return nil, fmt.Errorf("failed to convert prior state: %w", err)
		}
		priorState = providerschema.NewDynamicValue(priorStateCty, resourceType_cty)
	} else {
		// For new resources, use null value of the proper type instead of NoDynamicValue
		nullStateCty := cty.NullVal(resourceType_cty)
		priorState = providerschema.NewDynamicValue(nullStateCty, resourceType_cty)
	}

	// Convert new config to cty.Value
	configCty, err := a.converter.MapToCtyValue(newConfig, resourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	config := providerschema.NewDynamicValue(configCty, resourceType_cty)

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
	plannedStateCty, err := planResp.PlannedNewState().AsCtyValue(resourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned state: %w", err)
	}

	plannedStateMap, err := a.converter.CtyValueToMap(plannedStateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert planned state to map: %w", err)
	}

	return plannedStateMap, nil
}

func (a *OpenTofuAdapter) CreateResource(ctx context.Context, providerName, resourceType string, config map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	provider := a.providers[providerName]

	// Get resource schema
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerName, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceType_cty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert config to cty.Value
	configCty, err := a.converter.MapToCtyValue(config, resourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	configDV := providerschema.NewDynamicValue(configCty, resourceType_cty)

	// For new resources, use null value as prior state
	nullStateCty := cty.NullVal(resourceType_cty)
	priorState := providerschema.NewDynamicValue(nullStateCty, resourceType_cty)

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
	plannedStateCty, err := planResp.PlannedNewState().AsCtyValue(resourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned state: %w", err)
	}
	plannedStateDV := providerschema.NewDynamicValue(plannedStateCty, resourceType_cty)

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
	finalStateCty, err := applyResp.PlannedNewState().AsCtyValue(resourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode final state: %w", err)
	}

	finalStateMap, err := a.converter.CtyValueToMap(finalStateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert final state to map: %w", err)
	}

	return finalStateMap, nil
}

func (a *OpenTofuAdapter) DeleteResource(ctx context.Context, providerName, resourceType string, state map[string]any) error {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return err
	}

	provider := a.providers[providerName]

	// Get resource schema
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerName, resourceType)
	if err != nil {
		return fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceType_cty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert current state to cty.Value
	currentStateCty, err := a.converter.MapToCtyValue(state, resourceType_cty)
	if err != nil {
		return fmt.Errorf("failed to convert current state: %w", err)
	}
	priorState := providerschema.NewDynamicValue(currentStateCty, resourceType_cty)

	// For deletion, use null value as config and proposed new state
	nullStateCty := cty.NullVal(resourceType_cty)
	emptyConfig := providerschema.NewDynamicValue(nullStateCty, resourceType_cty)
	proposedNewState := providerschema.NewDynamicValue(nullStateCty, resourceType_cty)

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

	for diag := range planResp.Diagnostics().All() {
		fmt.Printf(
			"PLAN [%s] %s\n  %s\n\n",
			diag.Severity(),
			diag.Summary(),
			diag.Detail(),
		)
	}

	if planResp.Diagnostics().HasErrors() {
		return fmt.Errorf("plan deletion failed: %s", a.formatDiagnostics(planResp.Diagnostics()))
	}

	// Convert planned state from DynamicValueOut to DynamicValueIn
	plannedStateCty, err := planResp.PlannedNewState().AsCtyValue(resourceType_cty)
	if err != nil {
		return fmt.Errorf("failed to decode planned deletion state: %w", err)
	}
	plannedStateDV := providerschema.NewDynamicValue(plannedStateCty, resourceType_cty)

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

	for diag := range applyResp.Diagnostics().All() {
		fmt.Printf(
			"APPLY [%s] %s\n  %s\n\n",
			diag.Severity(),
			diag.Summary(),
			diag.Detail(),
		)
	}

	if applyResp.Diagnostics().HasErrors() {
		return fmt.Errorf("apply deletion failed: %s", a.formatDiagnostics(applyResp.Diagnostics()))
	}

	return nil
}

func (a *OpenTofuAdapter) ReadDataSource(ctx context.Context, providerName, dataSourceType string, config map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	provider := a.providers[providerName]

	// Get data source schema for proper type information
	opentofuSchema, err := a.getOpentofuDataSourceSchema(ctx, providerName, dataSourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	dataSourceType_cty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert config to cty.Value
	configCty, err := a.converter.MapToCtyValue(config, dataSourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}
	configDV := providerschema.NewDynamicValue(configCty, dataSourceType_cty)

	// Create read request
	readReq := &providerops.ReadDataResourceRequest{
		ResourceType: dataSourceType,
		Config:       configDV,
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
	stateCty, err := readResp.State().AsCtyValue(dataSourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	stateMap, err := a.converter.CtyValueToMap(stateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert state to map: %w", err)
	}

	return stateMap, nil
}

func (a *OpenTofuAdapter) ImportResource(ctx context.Context, providerName, resourceType, resourceID string) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	provider := a.providers[providerName]

	// Get resource schema for proper type information
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerName, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceType_cty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Create import request
	importReq := &providerops.ImportManagedResourceStateRequest{
		ResourceType: resourceType,
		ID:           resourceID,
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

	// Convert first imported resource state back to map
	stateCty, err := firstResource.State().AsCtyValue(resourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to decode imported state: %w", err)
	}

	stateMap, err := a.converter.CtyValueToMap(stateCty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert imported state to map: %w", err)
	}

	return stateMap, nil
}

func (a *OpenTofuAdapter) RefreshResource(ctx context.Context, providerName, resourceType string, currentState map[string]any) (map[string]any, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	provider := a.providers[providerName]

	// Get resource schema
	opentofuSchema, err := a.getOpentofuResourceSchema(ctx, providerName, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get opentofu schema: %w", err)
	}

	// Convert opentofu-providers schema to proper cty.Type
	resourceType_cty := a.schemaConverter.opentofuSchemaToObjectType(opentofuSchema)

	// Convert current state to cty.Value
	currentStateCty, err := a.converter.MapToCtyValue(currentState, resourceType_cty)
	if err != nil {
		return nil, fmt.Errorf("failed to convert current state: %w", err)
	}
	currentStateDV := providerschema.NewDynamicValue(currentStateCty, resourceType_cty)

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
	refreshedStateCty, err := refreshResp.NewState().AsCtyValue(resourceType_cty)
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
func (a *OpenTofuAdapter) UpdateResource(ctx context.Context, providerName, resourceType string, currentState, newConfig map[string]any) (map[string]any, error) {
	// First plan the change
	plannedState, err := a.PlanResource(ctx, providerName, resourceType, &currentState, newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to plan resource update: %w", err)
	}

	// For now, return the planned state since CreateResource is not fully implemented
	// TODO: Implement proper apply logic when CreateResource is implemented
	return plannedState, nil
}

// ListResources lists all available resource types for a provider
func (a *OpenTofuAdapter) ListResources(ctx context.Context, providerName string) ([]string, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	// Get cached raw schema response
	schemaResp, err := a.getRawSchema(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider schema: %w", err)
	}

	// Extract resource type names
	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())
	resourceTypes := make([]string, 0, len(resourceSchemas))

	for resourceType := range resourceSchemas {
		resourceTypes = append(resourceTypes, resourceType)
	}

	return resourceTypes, nil
}

// DescribeResource returns detailed information about a resource type
func (a *OpenTofuAdapter) DescribeResource(ctx context.Context, providerName, resourceType string) (*types.TypeDescription, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	// Get cached raw schema response
	schemaResp, err := a.getRawSchema(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider schema: %w", err)
	}

	// Find the specific resource schema
	resourceSchemas := maps.Collect(schemaResp.ProviderSchema().ManagedResourceTypeSchemas())
	resourceSchema, exists := resourceSchemas[resourceType]
	if !exists {
		return nil, fmt.Errorf("resource type %s not found in provider %s", resourceType, providerName)
	}

	// Convert to TypeDescription
	desc, err := a.schemaConverter.convertSchemaToTypeDescription(providerName, resourceType, resourceSchema, "resource")
	if err != nil {
		return nil, fmt.Errorf("failed to convert resource schema: %w", err)
	}

	return desc, nil
}

// DescribeDataSource returns detailed information about a data source type
func (a *OpenTofuAdapter) DescribeDataSource(ctx context.Context, providerName, dataSourceType string) (*types.TypeDescription, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	// Get cached schema
	schema, err := a.getSchema(ctx, providerName)
	if err != nil {
		return nil, err
	}

	// Check if provider has no data source schemas
	if len(schema.DataSources) == 0 {
		return nil, fmt.Errorf("provider has no data source schemas")
	}

	// Find the specific data source in cached schema
	desc, exists := schema.DataSources[dataSourceType]
	if !exists {
		return nil, fmt.Errorf("data source type %s not found in provider %s", dataSourceType, providerName)
	}

	return desc, nil
}

// ListDataSources lists all available data source types for a provider
func (a *OpenTofuAdapter) ListDataSources(ctx context.Context, providerName string) ([]string, error) {
	// Ensure provider is loaded
	if err := a.LoadProvider(ctx, providerName); err != nil {
		return nil, err
	}

	// Get cached schema
	schema, err := a.getSchema(ctx, providerName)
	if err != nil {
		return nil, err
	}

	// Extract data source type names from cached schema
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

func (a *OpenTofuAdapter) downloadProvider(ctx context.Context, providerName string) (string, error) {
	// Use the existing registry logic to download the provider
	downloadInfo, err := a.registry.GetProviderDownload(ctx, providerName)
	if err != nil {
		return "", fmt.Errorf("failed to get provider download info: %w", err)
	}

	// Download and extract provider
	binaryPath, err := a.downloadAndExtractProvider(ctx, providerName, downloadInfo.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download provider: %w", err)
	}

	return binaryPath, nil
}

// downloadAndExtractProvider downloads and extracts a provider binary
func (pm *OpenTofuAdapter) downloadAndExtractProvider(ctx context.Context, providerName, downloadURL string) (string, error) {

	// Create provider directory
	providerDir := filepath.Join(pm.tmpDir, strings.ReplaceAll(providerName, "/", "_"))
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create provider directory: %w", err)
	}

	// Check if binary already exists
	binaryPath, err := pm.findProviderBinary(providerDir)
	if err == nil {
		// Binary exists, check if it's executable
		if info, err := os.Stat(binaryPath); err == nil && info.Mode()&0111 != 0 {
			return binaryPath, nil
		}
	}

	// Download zip file
	zipPath := filepath.Join(providerDir, "provider.zip")
	if err := pm.downloadFile(ctx, downloadURL, zipPath); err != nil {
		return "", fmt.Errorf("failed to download provider: %w", err)
	}

	// Extract zip file
	if err := pm.extractZip(zipPath, providerDir); err != nil {
		return "", fmt.Errorf("failed to extract provider: %w", err)
	}

	// Find and make binary executable
	binaryPath, err = pm.findProviderBinary(providerDir)
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
func (pm *OpenTofuAdapter) downloadFile(ctx context.Context, url, path string) error {
	resp, err := pm.registry.Download(ctx, url)
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
func (pm *OpenTofuAdapter) extractZip(src, dest string) error {
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
