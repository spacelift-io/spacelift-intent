# Project Spacelift Intent MCP - Build Configuration

.PHONY: all build build-standalone clean test lint

# Build output directory
BUILD_DIR := bin

# Go build flags
GO_BUILD_FLAGS := -ldflags="-s -w" -trimpath

# Default target
all: build

# Build all binaries
build: build-standalone

# Build standalone mode binary
build-standalone:
	@echo "Building standalone server..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/spacelift-intent ./cmd/spacelift-intent


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

# Test standalone mode validation
test-validation: build
	@echo "Testing standalone mode validation..."
	./$(BUILD_DIR)/spacelift-intent --validate-only

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...


# Run standalone server
run-standalone:
	./$(BUILD_DIR)/spacelift-intent --server-type=http --port=11995

# Display help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  all                    - Build all binaries"
	@echo "  build                  - Build standalone binary"
	@echo "  build-standalone       - Build standalone server"
	@echo ""
	@echo "Test targets:"
	@echo "  test                   - Run tests"
	@echo "  test-validation        - Test standalone mode validation"
	@echo "  lint                   - Run linter"
	@echo "  fmt                    - Format code"
	@echo ""
	@echo "Run targets:"
	@echo "  run-standalone         - Run standalone server"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean                  - Clean build artifacts"
	@echo "  deps                   - Install dependencies"
	@echo "  help                   - Show this help"