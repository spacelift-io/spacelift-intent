# Project Spacelift Intent MCP - Build Configuration

.PHONY: all build clean test lint

# Build output directory
BUILD_DIR := bin

# Go build flags
GO_BUILD_FLAGS := -ldflags="-s -w" -trimpath

# Default target
all: build

# Build all binaries
build:
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/spacelift-intent ./cmd/spacelift-intent

build-legacy:
	@mkdir -p $(BUILD_DIR)
	GO_TAGS=legacy_plugin go build $(GO_BUILD_FLAGS) -tags=legacy_plugin -o $(BUILD_DIR)/spacelift-intent-legacy ./cmd/spacelift-intent

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

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...


# Run standalone server
run:
	./$(BUILD_DIR)/spacelift-intent

# Run server with environment variables from file
run-with-env:
	@if [ -f .env.aws ]; then \
		set -a; \
		. ./.env.aws; \
		set +a; \
		echo "AWS Environment Variables:"; \
		env | grep -E '^AWS_' | sort; \
		echo ""; \
		./$(BUILD_DIR)/spacelift-intent; \
	else \
		echo "Error: .env.aws file not found"; \
		exit 1; \
	fi

# Display help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  all                    - Build all binaries"
	@echo "  build                  - Build binary"
	@echo ""
	@echo "Test targets:"
	@echo "  test                   - Run tests"
	@echo "  lint                   - Run linter"
	@echo "  fmt                    - Format code"
	@echo ""
	@echo "Run targets:"
	@echo "  run         - Run MCP server"
	@echo "  run-with-env - Run MCP server with environment variables from .env.aws"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean                  - Clean build artifacts"
	@echo "  deps                   - Install dependencies"
	@echo "  help                   - Show this help"