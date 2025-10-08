package legacy

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/apparentlymart/opentofu-providers/tofuprovider/grpc/tfplugin5"
	"github.com/spacelift-io/spacelift-intent/types"
	"github.com/vmihailenco/msgpack/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrorBinaryNotFound = fmt.Errorf("provider binary not found")

// DefaultManager implements types.ProviderManager
type DefaultManager struct {
	providers map[string]*providerInfo
	tmpDir    string
	registry  types.RegistryClient
}

// NewManager creates a new provider manager
func NewManager(tmpDir string, registry types.RegistryClient) types.ProviderManager {
	return &DefaultManager{
		providers: make(map[string]*providerInfo),
		tmpDir:    tmpDir,
		registry:  registry,
	}
}

func (pm *DefaultManager) DescribeProvider(ctx context.Context, providerConfig *types.ProviderConfig) (*types.ProviderSchema, *string, error) {
	if err := pm.LoadProvider(ctx, providerConfig); err != nil {
		return nil, nil, err
	}

	schema, err := pm.getSchema(ctx, providerConfig)
	return schema, nil, err
}

func (pm *DefaultManager) getSchema(ctx context.Context, provider *types.ProviderConfig) (*types.ProviderSchema, error) {
	// Get resources list
	resources, err := pm.ListResources(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Get data sources list
	dataSources, err := pm.ListDataSources(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to list data sources: %w", err)
	}

	schema := &types.ProviderSchema{
		Resources:   make(map[string]*types.TypeDescription),
		DataSources: make(map[string]*types.TypeDescription),
	}

	sh, err := pm.getProviderSchema(ctx, provider)
	if err != nil {
		return nil, err
	}

	desc := pm.describeConfig(ctx, provider, sh)
	schema.Provider = desc

	// Build resource schemas
	for _, resourceType := range resources {
		desc, err := pm.describeResource(ctx, provider, resourceType, sh)
		if err != nil {
			return nil, fmt.Errorf("failed to describe resource %s: %w", resourceType, err)
		}
		schema.Resources[resourceType] = desc
	}

	// Build data source schemas
	for _, dataSourceType := range dataSources {
		desc, err := pm.describeDataSource(ctx, provider, dataSourceType, sh)
		if err != nil {
			return nil, fmt.Errorf("failed to describe data source %s: %w", dataSourceType, err)
		}
		schema.DataSources[dataSourceType] = desc
	}

	// Get provider version
	version, err := pm.GetProviderVersion(ctx, provider.Name)
	if err == nil {
		schema.Version = version
	}

	return schema, nil
}

// getProviderInfo returns internal provider info (private helper)
func (pm *DefaultManager) getProviderInfo(provider *types.ProviderConfig) (*providerInfo, error) {
	providerInfo, exists := pm.providers[provider.Name]
	if !exists {
		return nil, fmt.Errorf("provider %s not loaded", provider.Name)
	}
	return providerInfo, nil
}

// getProviderSchema loads a provider and returns its schema
func (pm *DefaultManager) getProviderSchema(ctx context.Context, provider *types.ProviderConfig) (*pb.GetProviderSchema_Response, error) {
	defer pm.Cleanup(ctx)
	if err := pm.LoadProvider(ctx, provider); err != nil {
		return nil, fmt.Errorf("failed to load provider: %w", err)
	}

	providerInfo, err := pm.getProviderInfo(provider)
	if err != nil {
		return nil, err
	}

	if providerInfo.schema == nil {
		return nil, fmt.Errorf("provider schema not loaded")
	}

	return providerInfo.schema, nil
}

// getProviderClient loads a provider and returns both its client and schema
func (pm *DefaultManager) getProviderClient(ctx context.Context, provider *types.ProviderConfig) (pb.ProviderClient, *pb.GetProviderSchema_Response, error) {
	if err := pm.LoadProvider(ctx, provider); err != nil {
		return nil, nil, fmt.Errorf("failed to load provider: %w", err)
	}

	providerInfo, err := pm.getProviderInfo(provider)
	if err != nil {
		return nil, nil, err
	}

	if providerInfo.schema == nil {
		return nil, nil, fmt.Errorf("provider schema not loaded")
	}

	return providerInfo.provider, providerInfo.schema, nil
}

func (pm *DefaultManager) GetProviderVersions(ctx context.Context, providerName string) ([]types.ProviderVersionInfo, error) {
	return pm.registry.GetProviderVersions(ctx, providerName)
}

// LoadProvider downloads and initializes a provider if not already loaded
func (pm *DefaultManager) LoadProvider(ctx context.Context, provider *types.ProviderConfig) error {
	// Check if provider is already loaded
	if _, exists := pm.providers[provider.Name]; exists {
		return nil
	}

	// Parse provider name
	parts := strings.Split(provider.Name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid provider name format, expected 'namespace/type'")
	}

	// Get download info from registry
	downloadInfo, err := pm.registry.GetProviderDownload(ctx, provider.Name, provider.Version)
	if err != nil {
		return fmt.Errorf("failed to get provider download info: %w", err)
	}

	// Download and extract provider
	binaryPath, err := pm.downloadAndExtractProvider(ctx, provider.Name, downloadInfo.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download provider: %w", err)
	}

	// Initialize the provider
	return pm.initializeProvider(ctx, provider.Name, binaryPath, downloadInfo.Version)
}

// downloadAndExtractProvider downloads and extracts a provider binary
func (pm *DefaultManager) downloadAndExtractProvider(ctx context.Context, providerName, downloadURL string) (string, error) {

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

// downloadFile downloads a file from URL to local path
func (pm *DefaultManager) downloadFile(ctx context.Context, url, path string) error {
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
func (pm *DefaultManager) extractZip(src, dest string) error {
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

// findProviderBinary finds the provider binary in a directory
func (pm *DefaultManager) findProviderBinary(dir string) (string, error) {
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
		return "", ErrorBinaryNotFound
	}
	return binaryPath, nil
}

// initializeProvider initializes a provider plugin
func (pm *DefaultManager) initializeProvider(ctx context.Context, providerName, binaryPath, version string) error {

	// Initialize provider using build-tagged implementation
	providerInfo, err := startProviderPlugin(binaryPath, providerName)
	if err != nil {
		return fmt.Errorf("failed to start provider plugin: %w", err)
	}

	// Load schema - kill on error
	schema, err := providerInfo.provider.GetSchema(ctx, &pb.GetProviderSchema_Request{})
	if err != nil {
		providerInfo.Kill()
		return fmt.Errorf("failed to get schema: %w", err)
	}
	providerInfo.schema = schema

	// Configure provider - kill on error
	if err := pm.configureProvider(ctx, providerInfo.provider, providerName); err != nil {
		providerInfo.Kill()
		return fmt.Errorf("failed to configure provider: %w", err)
	}

	// Set additional info and store
	providerInfo.version = version
	pm.providers[providerName] = providerInfo

	return nil
}

// configureProvider configures a provider with empty config
func (pm *DefaultManager) configureProvider(ctx context.Context, provider pb.ProviderClient, providerName string) error {
	emptyConfig, err := pm.encodeDynamicValue(nil)
	if err != nil {
		return fmt.Errorf("failed to encode empty config: %w", err)
	}

	// Try PrepareProviderConfig
	_, err = provider.PrepareProviderConfig(ctx, &pb.PrepareProviderConfig_Request{
		Config: emptyConfig,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
			fmt.Fprintf(os.Stderr, "[DEBUG] Provider %s does not implement PrepareProviderConfig\n", providerName)
		} else {
			fmt.Fprintf(os.Stderr, "[WARN] Provider %s PrepareProviderConfig failed: %v\n", providerName, err)
		}
	}

	// Configure provider
	configResp, err := provider.Configure(ctx, &pb.Configure_Request{
		TerraformVersion: "1.6.0",
		Config:           emptyConfig,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
			fmt.Fprintf(os.Stderr, "[DEBUG] Provider %s does not implement Configure\n", providerName)
		} else {
			return fmt.Errorf("provider Configure failed: %w", err)
		}
	} else if len(configResp.Diagnostics) > 0 {
		// Check for configuration errors
		for _, diag := range configResp.Diagnostics {
			if diag.Severity == pb.Diagnostic_ERROR {
				return fmt.Errorf("provider configuration failed with errors")
			}
		}
	}

	return nil
}

// GetProviderVersion returns the version of a loaded provider
func (pm *DefaultManager) GetProviderVersion(ctx context.Context, providerName string) (string, error) {
	provider, exists := pm.providers[providerName]
	if !exists {
		return "", fmt.Errorf("provider %s is not loaded", providerName)
	}

	return provider.version, nil
}

// Cleanup shuts down all provider clients
func (pm *DefaultManager) Cleanup(ctx context.Context) {
	for _, provider := range pm.providers {
		provider.Kill()
	}
	pm.providers = make(map[string]*providerInfo)
	cleanupPlugins()
}

// Resource management methods

// ListResources lists all available resource types for a provider
func (pm *DefaultManager) ListResources(ctx context.Context, provider *types.ProviderConfig) ([]string, error) {
	schema, err := pm.getProviderSchema(ctx, provider)
	if err != nil {
		return nil, err
	}

	if schema.ResourceSchemas == nil {
		return nil, nil
	}

	var resources []string
	for resourceType := range schema.ResourceSchemas {
		resources = append(resources, resourceType)
	}

	return resources, nil
}

func (pm *DefaultManager) describeConfig(_ context.Context, provider *types.ProviderConfig, schema *pb.GetProviderSchema_Response) *types.TypeDescription {
	properties, required := pm.convertSchema(schema.Provider)
	return &types.TypeDescription{
		ProviderName: provider.Name,
		Properties:   properties,
		Required:     required,
	}
}

// DescribeResource gets the schema and documentation for a resource type
func (pm *DefaultManager) DescribeResource(ctx context.Context, provider *types.ProviderConfig, resourceType string) (*types.TypeDescription, error) {
	schema, err := pm.getProviderSchema(ctx, provider)
	if err != nil {
		return nil, err
	}

	return pm.describeResource(ctx, provider, resourceType, schema)
}

// describeResource gets the schema and documentation for a resource type
func (pm *DefaultManager) describeResource(_ context.Context, provider *types.ProviderConfig, resourceType string, schema *pb.GetProviderSchema_Response) (*types.TypeDescription, error) {
	if err := pm.validateResourceExists(schema, provider.Name, resourceType); err != nil {
		return nil, err
	}

	resourceSchema := schema.ResourceSchemas[resourceType]
	properties, required := pm.convertSchema(resourceSchema)

	description := fmt.Sprintf("OpenTofu %s resource", resourceType)
	if resourceSchema.Block != nil {
		description = resourceSchema.Block.Description
	}

	return &types.TypeDescription{
		ProviderName: provider.Name,
		Type:         resourceType,
		Description:  description,
		Properties:   properties,
		Required:     required,
	}, nil
}

// PlanResource plans a resource instance
func (pm *DefaultManager) PlanResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, currentState *map[string]any, newConfig map[string]any) (map[string]any, error) {
	providerClient, schema, err := pm.getProviderClient(ctx, provider)
	defer pm.Cleanup(ctx)
	if err != nil {
		return nil, err
	}

	// Get resource schema
	if err := pm.validateResourceExists(schema, provider.Name, resourceType); err != nil {
		return nil, err
	}
	resourceSchema := schema.ResourceSchemas[resourceType]
	completeConfig := pm.buildCompleteConfig(newConfig, resourceSchema)

	// Encode configurations
	encodedCurrentState, err := pm.encodeDynamicValue(currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to encode current state: %w", err)
	}

	encodedNewConfig, err := pm.encodeDynamicValue(completeConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid new configuration: %w", err)
	}

	// Validate new configuration
	if err := pm.validateResource(ctx, providerClient, resourceType, encodedNewConfig); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Plan resource change
	planResp, err := providerClient.PlanResourceChange(ctx, &pb.PlanResourceChange_Request{
		TypeName:         resourceType,
		PriorState:       encodedCurrentState,
		ProposedNewState: encodedNewConfig,
		Config:           encodedNewConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("plan failed: %w", err)
	}

	if err := pm.checkDiagnostics(planResp.Diagnostics, "plan"); err != nil {
		return nil, err
	}

	// TODO: is this a correct assumption?
	// Current state was empty, and planning succeeded, so we can return just the params that were provided, all others are auto-generated
	if currentState == nil || len(*currentState) == 0 {
		return completeConfig, nil
	}

	var plannedState map[string]any
	if err := pm.decodeDynamicValue(planResp.PlannedState, &plannedState); err != nil {
		return nil, fmt.Errorf("failed to decode planned state: %w", err)
	}

	return plannedState, nil
}

// CreateResource creates a new resource instance
func (pm *DefaultManager) CreateResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, config map[string]any) (map[string]any, error) {
	providerClient, schema, err := pm.getProviderClient(ctx, provider)
	defer pm.Cleanup(ctx)
	if err != nil {
		return nil, err
	}

	// Get resource schema
	if err := pm.validateResourceExists(schema, provider.Name, resourceType); err != nil {
		return nil, err
	}
	resourceSchema := schema.ResourceSchemas[resourceType]

	// Build complete configuration
	completeConfig := pm.buildCompleteConfig(config, resourceSchema)

	// Encode configuration
	encodedConfig, err := pm.encodeDynamicValue(completeConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate configuration
	if err := pm.validateResource(ctx, providerClient, resourceType, encodedConfig); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Plan and apply resource creation
	return pm.planAndApplyResource(ctx, providerClient, resourceType, encodedConfig)
}

// UpdateResource updates an existing resource instance
func (pm *DefaultManager) UpdateResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, currentState, newConfig map[string]any) (map[string]any, error) {
	providerClient, schema, err := pm.getProviderClient(ctx, provider)
	defer pm.Cleanup(ctx)
	if err != nil {
		return nil, err
	}

	// Get resource schema
	if err := pm.validateResourceExists(schema, provider.Name, resourceType); err != nil {
		return nil, err
	}
	resourceSchema := schema.ResourceSchemas[resourceType]

	// Build complete new configuration
	completeConfig := pm.buildCompleteConfig(newConfig, resourceSchema)

	// Encode configurations
	encodedCurrentState, err := pm.encodeDynamicValue(currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to encode current state: %w", err)
	}

	encodedNewConfig, err := pm.encodeDynamicValue(completeConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid new configuration: %w", err)
	}

	// Validate new configuration
	if err := pm.validateResource(ctx, providerClient, resourceType, encodedNewConfig); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Plan resource update
	planResp, err := providerClient.PlanResourceChange(ctx, &pb.PlanResourceChange_Request{
		TypeName:         resourceType,
		PriorState:       encodedCurrentState,
		ProposedNewState: encodedNewConfig,
		Config:           encodedNewConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("plan failed: %w", err)
	}

	if err := pm.checkDiagnostics(planResp.Diagnostics, "plan"); err != nil {
		return nil, err
	}

	// Apply resource update
	applyResp, err := providerClient.ApplyResourceChange(ctx, &pb.ApplyResourceChange_Request{
		TypeName:       resourceType,
		PriorState:     encodedCurrentState,
		PlannedState:   planResp.PlannedState,
		Config:         encodedNewConfig,
		PlannedPrivate: planResp.PlannedPrivate,
	})
	if err != nil {
		return nil, fmt.Errorf("apply failed: %w", err)
	}

	if err := pm.checkDiagnostics(applyResp.Diagnostics, "apply"); err != nil {
		return nil, err
	}

	// Decode final state
	var state map[string]any
	if err := pm.decodeDynamicValue(applyResp.NewState, &state); err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	return state, nil
}

// DeleteResource deletes an existing resource instance
func (pm *DefaultManager) DeleteResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, state map[string]any) error {
	providerClient, schema, err := pm.getProviderClient(ctx, provider)
	defer pm.Cleanup(ctx)
	if err != nil {
		return err
	}

	// Get resource schema
	if err := pm.validateResourceExists(schema, provider.Name, resourceType); err != nil {
		return err
	}

	// Encode current state (old state)
	encodedState, err := pm.encodeDynamicValue(state)
	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	// Plan and apply resource deletion
	return pm.planAndApplyDeletion(ctx, providerClient, resourceType, encodedState)
}

// RefreshResource reads the current state of an existing resource from the provider
func (pm *DefaultManager) RefreshResource(ctx context.Context, provider *types.ProviderConfig, resourceType string, currentState map[string]any) (map[string]any, error) {
	providerClient, schema, err := pm.getProviderClient(ctx, provider)
	defer pm.Cleanup(ctx)
	if err != nil {
		return nil, err
	}

	// Validate that the resource type exists
	if err := pm.validateResourceExists(schema, provider.Name, resourceType); err != nil {
		return nil, err
	}
	resourceSchema := schema.ResourceSchemas[resourceType]
	complateState := pm.buildCompleteConfig(currentState, resourceSchema)

	// Encode current state
	encodedState, err := pm.encodeDynamicValue(complateState)
	if err != nil {
		return nil, fmt.Errorf("failed to encode current state: %w", err)
	}

	// Read the resource to get the current state from the provider
	readResp, err := providerClient.ReadResource(ctx, &pb.ReadResource_Request{
		TypeName:     resourceType,
		CurrentState: encodedState,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to refresh resource: %w", err)
	}

	if err := pm.checkDiagnostics(readResp.Diagnostics, "refresh"); err != nil {
		return nil, err
	}

	// Decode the refreshed state
	var refreshedState map[string]any
	if err := pm.decodeDynamicValue(readResp.NewState, &refreshedState); err != nil {
		return nil, fmt.Errorf("failed to decode refreshed state: %w", err)
	}

	return refreshedState, nil
}

// ImportResource imports an existing resource into the state management system
func (pm *DefaultManager) ImportResource(ctx context.Context, provider *types.ProviderConfig, resourceType, importID string) (map[string]any, error) {
	providerClient, schema, err := pm.getProviderClient(ctx, provider)
	defer pm.Cleanup(ctx)
	if err != nil {
		return nil, err
	}

	// Validate that the resource type exists
	if err := pm.validateResourceExists(schema, provider.Name, resourceType); err != nil {
		return nil, err
	}

	// Create an import request
	importResp, err := providerClient.ImportResourceState(ctx, &pb.ImportResourceState_Request{
		TypeName: resourceType,
		Id:       importID,
	})
	if err != nil {
		return nil, fmt.Errorf("import failed: %w", err)
	}

	if err := pm.checkDiagnostics(importResp.Diagnostics, "import"); err != nil {
		return nil, err
	}

	// Terraform can return multiple imported resources, but we'll use the first one
	if len(importResp.ImportedResources) == 0 {
		return nil, fmt.Errorf("no resources were imported")
	}

	importedResource := importResp.ImportedResources[0]

	// Now read the resource to get the complete current state
	readResp, err := providerClient.ReadResource(ctx, &pb.ReadResource_Request{
		TypeName:     resourceType,
		CurrentState: importedResource.State,
		Private:      importedResource.Private,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read imported resource: %w", err)
	}

	if err := pm.checkDiagnostics(readResp.Diagnostics, "read after import"); err != nil {
		return nil, err
	}

	// Decode the complete current state
	var state map[string]any
	if err := pm.decodeDynamicValue(readResp.NewState, &state); err != nil {
		return nil, fmt.Errorf("failed to decode current state: %w", err)
	}

	return state, nil
}

// Data source management methods

// ListDataSources lists all available data source types for a provider
func (pm *DefaultManager) ListDataSources(ctx context.Context, provider *types.ProviderConfig) ([]string, error) {
	schema, err := pm.getProviderSchema(ctx, provider)
	if err != nil {
		return nil, err
	}

	if schema.DataSourceSchemas == nil {
		return nil, nil
	}

	var dataSources []string
	for dataSourceType := range schema.DataSourceSchemas {
		dataSources = append(dataSources, dataSourceType)
	}

	return dataSources, nil
}

// DescribeDataSource gets the schema and documentation for a data source type
func (pm *DefaultManager) DescribeDataSource(ctx context.Context, provider *types.ProviderConfig, dataSourceType string) (*types.TypeDescription, error) {
	schema, err := pm.getProviderSchema(ctx, provider)
	if err != nil {
		return nil, err
	}

	return pm.describeDataSource(ctx, provider, dataSourceType, schema)
}

// describeDataSource gets the schema and documentation for a data source type
func (pm *DefaultManager) describeDataSource(_ context.Context, provider *types.ProviderConfig, dataSourceType string, schema *pb.GetProviderSchema_Response) (*types.TypeDescription, error) {
	if err := pm.validateDataSourceExists(schema, provider.Name, dataSourceType); err != nil {
		return nil, err
	}

	dataSourceSchema := schema.DataSourceSchemas[dataSourceType]

	properties, required := pm.convertSchema(dataSourceSchema)

	return &types.TypeDescription{
		ProviderName: provider.Name,
		Type:         dataSourceType,
		Description:  fmt.Sprintf("OpenTofu %s data source", dataSourceType),
		Properties:   properties,
		Required:     required,
	}, nil
}

// ReadDataSource reads data from a data source
func (pm *DefaultManager) ReadDataSource(ctx context.Context, provider *types.ProviderConfig, dataSourceType string, config map[string]any) (map[string]any, error) {
	providerClient, schema, err := pm.getProviderClient(ctx, provider)
	defer pm.Cleanup(ctx)
	if err != nil {
		return nil, err
	}

	// Get data source schema
	if err := pm.validateDataSourceExists(schema, provider.Name, dataSourceType); err != nil {
		return nil, err
	}
	dataSourceSchema := schema.DataSourceSchemas[dataSourceType]

	// Validate that all required fields are provided
	properties, required := pm.convertSchema(dataSourceSchema)
	var missingFields []string
	for _, field := range required {
		if _, exists := config[field]; !exists {
			missingFields = append(missingFields, field)
		}
	}
	if len(missingFields) > 0 {
		return nil, fmt.Errorf("missing required fields for data source '%s': %v. Available fields: %v",
			dataSourceType, missingFields, func() []string {
				var props []string
				for k := range properties {
					props = append(props, k)
				}
				return props
			}())
	}

	// For data sources, use the config as-is (don't add nil values for optional fields)
	completeConfig := pm.buildCompleteConfig(config, dataSourceSchema)

	// Encode configuration
	encodedConfig, err := pm.encodeDynamicValue(completeConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate configuration
	if err := pm.validateDataSource(ctx, providerClient, dataSourceType, encodedConfig); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Read data source
	readResp, err := providerClient.ReadDataSource(ctx, &pb.ReadDataSource_Request{
		TypeName: dataSourceType,
		Config:   encodedConfig,
	})

	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	if err := pm.checkDiagnostics(readResp.Diagnostics, "read"); err != nil {
		return nil, err
	}

	// Decode state
	var state map[string]any
	if err := pm.decodeDynamicValue(readResp.State, &state); err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	return state, nil
}

// Helper methods

// validateResourceExists checks if a resource type exists in the provider schema
func (pm *DefaultManager) validateResourceExists(schema *pb.GetProviderSchema_Response, providerName, resourceType string) error {
	if schema.ResourceSchemas == nil {
		return fmt.Errorf("provider has no resource schemas")
	}
	if _, exists := schema.ResourceSchemas[resourceType]; !exists {
		return fmt.Errorf("resource type '%s' not found in provider '%s'", resourceType, providerName)
	}
	return nil
}

// validateDataSourceExists checks if a data source type exists in the provider schema
func (pm *DefaultManager) validateDataSourceExists(schema *pb.GetProviderSchema_Response, providerName, dataSourceType string) error {
	if schema.DataSourceSchemas == nil {
		return fmt.Errorf("provider has no data source schemas")
	}
	if _, exists := schema.DataSourceSchemas[dataSourceType]; !exists {
		return fmt.Errorf("data source type '%s' not found in provider '%s'", dataSourceType, providerName)
	}
	return nil
}

// planAndApplyResource handles the plan and apply cycle for resource creation
func (pm *DefaultManager) planAndApplyResource(ctx context.Context, provider pb.ProviderClient, resourceType string, config *pb.DynamicValue) (map[string]any, error) {
	nullState, err := pm.encodeDynamicValue(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encode null state: %w", err)
	}

	// Plan resource
	planResp, err := provider.PlanResourceChange(ctx, &pb.PlanResourceChange_Request{
		TypeName:         resourceType,
		PriorState:       nullState,
		ProposedNewState: config,
		Config:           config,
	})
	if err != nil {
		return nil, fmt.Errorf("plan failed: %w", err)
	}

	if err := pm.checkDiagnostics(planResp.Diagnostics, "plan"); err != nil {
		return nil, err
	}

	// Apply resource
	applyResp, err := provider.ApplyResourceChange(ctx, &pb.ApplyResourceChange_Request{
		TypeName:       resourceType,
		PriorState:     nullState,
		PlannedState:   planResp.PlannedState,
		Config:         config,
		PlannedPrivate: planResp.PlannedPrivate,
	})
	if err != nil {
		return nil, fmt.Errorf("apply failed: %w", err)
	}

	if err := pm.checkDiagnostics(applyResp.Diagnostics, "apply"); err != nil {
		return nil, err
	}

	// Decode final state
	var state map[string]any
	if err := pm.decodeDynamicValue(applyResp.NewState, &state); err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	return state, nil
}

// planAndApplyDeletion handles the plan and apply cycle for resource deletion
func (pm *DefaultManager) planAndApplyDeletion(ctx context.Context, provider pb.ProviderClient, resourceType string, currentState *pb.DynamicValue) error {
	emptyState, err := pm.encodeDynamicValue(nil)
	if err != nil {
		return fmt.Errorf("failed to encode empty state: %w", err)
	}

	// Plan resource deletion
	planResp, err := provider.PlanResourceChange(ctx, &pb.PlanResourceChange_Request{
		TypeName:         resourceType,
		PriorState:       currentState,
		ProposedNewState: emptyState,
		Config:           emptyState,
	})
	if err != nil {
		return fmt.Errorf("plan deletion failed: %w", err)
	}

	if err := pm.checkDiagnostics(planResp.Diagnostics, "plan deletion"); err != nil {
		return err
	}

	// Apply resource deletion
	applyResp, err := provider.ApplyResourceChange(ctx, &pb.ApplyResourceChange_Request{
		TypeName:       resourceType,
		PriorState:     currentState,
		PlannedState:   planResp.PlannedState,
		Config:         emptyState,
		PlannedPrivate: planResp.PlannedPrivate,
	})
	if err != nil {
		return fmt.Errorf("apply deletion failed: %w", err)
	}

	return pm.checkDiagnostics(applyResp.Diagnostics, "apply deletion")
}

// buildCompleteConfig builds a complete configuration with defaults
func (pm *DefaultManager) buildCompleteConfig(config map[string]any, schema *pb.Schema) map[string]any {
	completeConfig := make(map[string]any)

	// Copy provided configuration
	maps.Copy(completeConfig, config)

	// Add defaults for optional/computed attributes
	if schema.Block != nil && schema.Block.Attributes != nil {
		for _, attr := range schema.Block.Attributes {
			if _, exists := completeConfig[attr.Name]; !exists {
				if attr.Optional || attr.Computed {
					completeConfig[attr.Name] = nil
				}
			}
		}
	}

	// Add defaults for block_types
	if schema.Block != nil && schema.Block.BlockTypes != nil {
		for _, blockType := range schema.Block.BlockTypes {
			if _, exists := completeConfig[blockType.TypeName]; !exists {
				// Add nil for optional blocks, unless they have minimum items requirement
				if blockType.MinItems == 0 {
					completeConfig[blockType.TypeName] = nil
				}
			}
		}
	}

	return completeConfig
}

// validateResource validates a resource configuration
func (pm *DefaultManager) validateResource(ctx context.Context, provider pb.ProviderClient, resourceType string, config *pb.DynamicValue) error {
	validateResp, err := provider.ValidateResourceTypeConfig(ctx, &pb.ValidateResourceTypeConfig_Request{
		TypeName: resourceType,
		Config:   config,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
			// Provider doesn't implement validation, continue
			return nil
		}
		return err
	}

	return pm.checkDiagnostics(validateResp.Diagnostics, "validate")
}

// validateDataSource validates a data source configuration
func (pm *DefaultManager) validateDataSource(ctx context.Context, provider pb.ProviderClient, dataSourceType string, config *pb.DynamicValue) error {
	validateResp, err := provider.ValidateDataSourceConfig(ctx, &pb.ValidateDataSourceConfig_Request{
		TypeName: dataSourceType,
		Config:   config,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
			// Provider doesn't implement validation, continue
			return nil
		}
		return err
	}

	return pm.checkDiagnostics(validateResp.Diagnostics, "validate")
}

// convertSchema converts an OpenTofu schema to JSON schema format
func (pm *DefaultManager) convertSchema(schema *pb.Schema) (map[string]any, []string) {

	properties := make(map[string]any)
	var required []string

	if schema.Block == nil {
		return properties, required
	}

	// Process attributes
	if schema.Block.Attributes != nil {
		for _, attr := range schema.Block.Attributes {
			if attr.Name == "" {
				continue
			}

			prop := pm.convertTypeToJSONSchema(attr.Type)

			if attr.Description != "" {
				prop["description"] = attr.Description
			}

			if attr.Computed {
				if desc, ok := prop["description"].(string); ok {
					prop["description"] = fmt.Sprintf("%s (computed - will be generated if not provided)", desc)
				} else {
					prop["description"] = "computed value - will be generated if not provided"
				}
			}

			properties[attr.Name] = prop

			if attr.Required && !attr.Computed {
				required = append(required, attr.Name)
			}
		}
	}

	// Process block_types
	if schema.Block.BlockTypes != nil {
		blockProps, blockRequired := pm.convertBlockTypes(schema.Block.BlockTypes)
		for name, prop := range blockProps {
			properties[name] = prop
		}
		required = append(required, blockRequired...)
	}

	return properties, required
}

// convertBlockTypes converts block_types from OpenTofu schema to JSON schema format
func (pm *DefaultManager) convertBlockTypes(blockTypes []*pb.Schema_NestedBlock) (map[string]any, []string) {
	properties := make(map[string]any)
	var required []string

	for _, blockType := range blockTypes {
		if blockType.TypeName == "" {
			continue
		}

		blockSchema := pm.convertNestedBlock(blockType)
		properties[blockType.TypeName] = blockSchema

		// Block types are typically optional by default unless specified otherwise
		// We could add logic here to make certain block types required based on min_items
		if blockType.MinItems > 0 {
			required = append(required, blockType.TypeName)
		}
	}

	return properties, required
}

// convertNestedBlock converts a single nested block to JSON schema format
func (pm *DefaultManager) convertNestedBlock(nestedBlock *pb.Schema_NestedBlock) map[string]any {
	blockSchema := make(map[string]any)

	if nestedBlock.Block == nil {
		return map[string]any{"type": "object"}
	}

	// Recursively convert the nested block's attributes and block_types
	blockProperties := make(map[string]any)
	var blockRequired []string

	// Process attributes within the nested block
	if nestedBlock.Block.Attributes != nil {
		for _, attr := range nestedBlock.Block.Attributes {
			if attr.Name == "" {
				continue
			}

			prop := pm.convertTypeToJSONSchema(attr.Type)

			if attr.Description != "" {
				prop["description"] = attr.Description
			}

			if attr.Computed {
				if desc, ok := prop["description"].(string); ok {
					prop["description"] = fmt.Sprintf("%s (computed - will be generated if not provided)", desc)
				} else {
					prop["description"] = "computed value - will be generated if not provided"
				}
			}

			blockProperties[attr.Name] = prop

			if attr.Required && !attr.Computed {
				blockRequired = append(blockRequired, attr.Name)
			}
		}
	}

	// Process nested block_types within this block
	if nestedBlock.Block.BlockTypes != nil {
		nestedBlockProps, nestedBlockRequired := pm.convertBlockTypes(nestedBlock.Block.BlockTypes)
		for name, prop := range nestedBlockProps {
			blockProperties[name] = prop
		}
		blockRequired = append(blockRequired, nestedBlockRequired...)
	}

	// Create the appropriate schema based on nesting mode
	switch nestedBlock.Nesting {
	case pb.Schema_NestedBlock_SINGLE:
		// Single nested block - represented as an object
		blockSchema["type"] = "object"
		blockSchema["properties"] = blockProperties
		if len(blockRequired) > 0 {
			blockSchema["required"] = blockRequired
		}
		if nestedBlock.MaxItems > 0 {
			blockSchema["maxItems"] = nestedBlock.MaxItems
		}

	case pb.Schema_NestedBlock_LIST:
		// List of nested blocks - represented as array of objects
		blockSchema["type"] = "array"
		itemSchema := map[string]any{
			"type":       "object",
			"properties": blockProperties,
		}
		if len(blockRequired) > 0 {
			itemSchema["required"] = blockRequired
		}
		blockSchema["items"] = itemSchema
		if nestedBlock.MinItems > 0 {
			blockSchema["minItems"] = nestedBlock.MinItems
		}
		if nestedBlock.MaxItems > 0 {
			blockSchema["maxItems"] = nestedBlock.MaxItems
		}

	case pb.Schema_NestedBlock_SET:
		// Set of nested blocks - represented as array of objects with unique items
		blockSchema["type"] = "array"
		itemSchema := map[string]any{
			"type":       "object",
			"properties": blockProperties,
		}
		if len(blockRequired) > 0 {
			itemSchema["required"] = blockRequired
		}
		blockSchema["items"] = itemSchema
		blockSchema["uniqueItems"] = true
		if nestedBlock.MinItems > 0 {
			blockSchema["minItems"] = nestedBlock.MinItems
		}
		if nestedBlock.MaxItems > 0 {
			blockSchema["maxItems"] = nestedBlock.MaxItems
		}

	case pb.Schema_NestedBlock_MAP:
		// Map of nested blocks - represented as object with additional properties
		blockSchema["type"] = "object"
		blockSchema["additionalProperties"] = map[string]any{
			"type":       "object",
			"properties": blockProperties,
		}
		if len(blockRequired) > 0 {
			blockSchema["additionalProperties"].(map[string]any)["required"] = blockRequired
		}

	default:
		// Default to object for unknown nesting modes
		blockSchema["type"] = "object"
		blockSchema["properties"] = blockProperties
		if len(blockRequired) > 0 {
			blockSchema["required"] = blockRequired
		}
	}

	// Add block description if available
	if nestedBlock.Block.Description != "" {
		blockSchema["description"] = nestedBlock.Block.Description
	}

	return blockSchema
}

// convertTypeToJSONSchema converts Terraform type information to JSON schema
func (pm *DefaultManager) convertTypeToJSONSchema(tfType []byte) map[string]any {
	if len(tfType) == 0 {
		return map[string]any{"type": "string"}
	}

	var typeData any
	if err := json.Unmarshal(tfType, &typeData); err != nil {
		return map[string]any{"type": "string"}
	}
	return pm.parseTypeSpecification(typeData)
}

// parseTypeSpecification parses Terraform type specifications into JSON schema
func (pm *DefaultManager) parseTypeSpecification(typeSpec any) map[string]any {
	// Handle array format: ["list", elementType] or ["map", elementType] etc
	if typeArray, ok := typeSpec.([]any); ok && len(typeArray) > 0 {
		typeKind, ok := typeArray[0].(string)
		if !ok {
			return map[string]any{"type": "string"}
		}

		switch typeKind {
		case "list":
			elementType := map[string]any{"type": "string"}
			if len(typeArray) > 1 {
				elementType = pm.parseTypeSpecification(typeArray[1])
			}
			return map[string]any{
				"type":  "array",
				"items": elementType,
			}

		case "map":
			elementType := map[string]any{"type": "string"}
			if len(typeArray) > 1 {
				elementType = pm.parseTypeSpecification(typeArray[1])
			}
			return map[string]any{
				"type":                 "object",
				"additionalProperties": elementType,
			}

		case "set":
			elementType := map[string]any{"type": "string"}
			if len(typeArray) > 1 {
				elementType = pm.parseTypeSpecification(typeArray[1])
			}
			return map[string]any{
				"type":        "array",
				"items":       elementType,
				"uniqueItems": true,
			}

		default:
			return map[string]any{"type": "string"}
		}
	}

	// Handle string types
	if typeStr, ok := typeSpec.(string); ok {
		switch typeStr {
		case "string":
			return map[string]any{"type": "string"}
		case "number":
			return map[string]any{"type": "number"}
		case "bool":
			return map[string]any{"type": "boolean"}
		default:
			return map[string]any{"type": "string"}
		}
	}

	return map[string]any{"type": "string"}
}

// encodeDynamicValue encodes data as msgpack (like OpenTofu does)
func (pm *DefaultManager) encodeDynamicValue(data any) (*pb.DynamicValue, error) {
	msgpackBytes, err := msgpack.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode as msgpack: %w", err)
	}
	return &pb.DynamicValue{Msgpack: msgpackBytes}, nil
}

// TODO: this is a hack to handle the fact that the provider is using a custom msgpack decoder
// that doesn't count the number of fields in the msgpack data.
// We need to find a better way to handle this.
// type FieldCountMsgpackDecoderEncoder struct {
// 	Data []byte
// }

// func (m *FieldCountMsgpackDecoderEncoder) MarshalMsgpack() ([]byte, error) {
// 	return m.Data, nil
// }

// func (m *FieldCountMsgpackDecoderEncoder) UnmarshalMsgpack(data []byte) error {
// 	m.Data = data
// 	return nil
// }

// decodeDynamicValue decodes a DynamicValue, trying both JSON and Msgpack fields
func (pm *DefaultManager) decodeDynamicValue(dv *pb.DynamicValue, target any) error {
	// Try Msgpack first
	if len(dv.Msgpack) > 0 {
		/*
			Mysterious 0 type, my wild guess is that it's a field count
			TODO: find out what it is and how to handle it
			It surfaces only during plan resource change decoding
			Most likely it can be deleted due to workaround in PlanResource
			if len(currentState) == 0 {
				return completeConfig, nil
			}
			James suggested to use cty github.com/zclconf/go-cty instead of using msgpack directly
			this is a big TODO!
		*/

		// msgpack.RegisterExt(0, &FieldCountMsgpackDecoderEncoder{})

		err := msgpack.Unmarshal(dv.Msgpack, target)
		if err != nil {
			return fmt.Errorf("msgpack decode failed: %w", err)
		}
		return nil
	}

	// Try JSON as fallback
	if len(dv.Json) > 0 {
		return json.Unmarshal(dv.Json, target)
	}

	// Neither JSON nor Msgpack data available
	return fmt.Errorf("no data in DynamicValue (both JSON and Msgpack fields are empty)")
}

// checkDiagnostics validates diagnostics and returns an error if there are any errors
func (pm *DefaultManager) checkDiagnostics(diagnostics []*pb.Diagnostic, operation string) error {
	var errors []string
	var warnings []string

	for _, diag := range diagnostics {
		message := diag.Summary
		if diag.Detail != "" {
			message += ": " + diag.Detail
		}

		switch diag.Severity {
		case pb.Diagnostic_ERROR:
			errors = append(errors, message)
		case pb.Diagnostic_WARNING:
			warnings = append(warnings, message)
		}
	}

	// Log warnings even if they don't cause failures
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "[WARN] %s warning: %s\n", operation, warning)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s errors: %s", operation, strings.Join(errors, "; "))
	}

	return nil
}
