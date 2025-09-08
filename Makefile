# Project Squirrel - Multi-Mode Build Configuration

.PHONY: all build build-standalone build-mcp-server build-executor clean test lint

# Build output directory
BUILD_DIR := bin

# Go build flags
GO_BUILD_FLAGS := -ldflags="-s -w" -trimpath

# Default target
all: build

# Build all binaries
build: build-standalone build-mcp-server build-executor

# Build standalone mode binary
build-standalone:
	@echo "Building standalone server..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/squirrel-standalone ./cmd/standalone

# Build MCP server wrapper binary
build-mcp-server:
	@echo "Building MCP server wrapper..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/squirrel-mcp-server ./cmd/mcpserver

# Build executor binary
build-executor:
	@echo "Building executor server..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/squirrel-executor ./cmd/executor


# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Test all modes validation
test-validation: build
	@echo "Testing standalone mode validation..."
	./$(BUILD_DIR)/squirrel-standalone --validate-only
	@echo "Testing MCP server wrapper validation..."
	./$(BUILD_DIR)/squirrel-mcp-server --spacelift-url=ws://localhost:9090 --validate-only
	@echo "Testing executor validation..."
	./$(BUILD_DIR)/squirrel-executor --spacelift-url=ws://localhost:9090 --validate-only

# Development build (debug mode)
build-dev:
	@echo "Building in development mode..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/squirrel-dev ./cmd/squirrel

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Start full distributed environment
docker-up:
	@echo "Starting distributed environment..."
	docker-compose up -d

# Stop distributed environment
docker-down:
	@echo "Stopping distributed environment..."
	docker-compose down

# View logs from all services
docker-logs:
	docker-compose logs -f

# Run specific mode for testing
run-standalone:
	./$(BUILD_DIR)/squirrel-standalone --server-type=http --port=11995

# Display help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  all                    - Build all binaries"
	@echo "  build                  - Build all binaries"
	@echo "  build-standalone       - Build standalone server"
	@echo "  build-mcp-server       - Build MCP server wrapper"
	@echo "  build-executor         - Build executor server"
	@echo "  build-dev              - Build in development mode"
	@echo ""
	@echo "Test targets:"
	@echo "  test                   - Run tests"
	@echo "  test-validation        - Test all modes validation"
	@echo "  lint                   - Run linter"
	@echo "  fmt                    - Format code"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-up              - Start distributed environment"
	@echo "  docker-down            - Stop distributed environment"
	@echo "  docker-logs            - View logs from all services"
	@echo ""
	@echo "Run targets:"
	@echo "  run-standalone         - Run standalone server"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean                  - Clean build artifacts"
	@echo "  deps                   - Install dependencies"
	@echo "  help                   - Show this help"