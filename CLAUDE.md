# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go library that provides a low-level client for interacting with OpenTofu provider plugins without using OpenTofu Core code directly. It abstracts the wire protocol details while maintaining direct access to provider protocol operations.

## Commands

### Building
```bash
go build ./...
```

### Testing
```bash
go test ./...
```

### Running Examples
```bash
# Build and run the example with a provider executable
cd examples/tofuplugin-example-types
go build -o main .
./main <provider-executable> [provider-args...]
```

### Protocol Buffer Generation
To regenerate protocol buffers after updating .proto files:
```bash
cd tofuprovider/grpc/tfplugin5
./generate.sh

cd ../tfplugin6  
./generate.sh
```

## Architecture

### Core Components

- **`tofuprovider/`** - Main package containing the `Provider` interface and core abstractions
- **`tofuprovider/providerops/`** - Request/response types for all provider operations (GetProviderSchema, ConfigureProvider, etc.)
- **`tofuprovider/providerschema/`** - Schema representation types for providers, resources, and data sources
- **`tofuprovider/grpc/`** - GRPC protocol implementations for tfplugin5 and tfplugin6
- **`tofuprovider/internal/`** - Internal utilities and protocol adapters

### Key Interfaces

The main `Provider` interface in `tofuprovider/provider.go` defines operations like:
- `GetProviderSchema()` - Get full provider schema
- `ConfigureProvider()` - Configure provider with given config
- `ValidateProviderConfig()` - Validate provider configuration
- Resource operations (Plan, Apply, Read, Import, etc.)
- Data source operations
- Function calls

### Protocol Support

The library supports multiple OpenTofu provider protocol versions:
- Protocol 5 (`tofuprovider/grpc/tfplugin5/`)  
- Protocol 6 (`tofuprovider/grpc/tfplugin6/`)

### Type System

Uses `go-cty` library for OpenTofu's type system:
- `cty.Type` for type representations
- `cty.Value` for values
- No semantic transformations - values passed verbatim to providers

## Development Notes

### Design Philosophy
- Abstraction level 2: Hides wire protocol details but exposes conceptual provider operations
- Minimal translation overhead using dynamic dispatch and Go iterators
- Follows OpenTofu Core naming conventions for consistency

### Provider Lifecycle
1. Start provider plugin with `tofuprovider.StartGRPCPlugin()`
2. Get schema with `GetProviderSchema()`
3. Validate config with `ValidateProviderConfig()`
4. Configure with `ConfigureProvider()`
5. Perform resource/data source operations
6. Close provider connection

### Wire Protocol Details
Protocol buffers are generated from `.proto` files in the grpc subdirectories. The library handles translation between protocol-specific types and the unified `providerops` types.