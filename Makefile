# Makefile for Lux EVM (Subnet EVM)

# Variables
BINARY_NAME := evm
PLUGIN_PATH := ~/.luxd/plugins/$(BINARY_NAME)
VERSION := $(shell git describe --tags --always --dirty="-dev" 2>/dev/null || echo "unknown")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date +%FT%T%z)

# Go build flags
GOARCH := $(shell go env GOARCH)
GOOS := $(shell go env GOOS)
CGO_CFLAGS := -O -D__BLST_PORTABLE__
LDFLAGS := -X github.com/luxfi/evm/plugin/evm.Version=$(VERSION)

# Default target
.PHONY: all
all: build

# Build the EVM binary
.PHONY: build
build:
	@echo "Building Lux EVM..."
	@mkdir -p build
	CGO_CFLAGS="$(CGO_CFLAGS)" go build -ldflags "$(LDFLAGS)" -o build/$(BINARY_NAME) ./plugin

# Build and install as plugin
.PHONY: install
install: build
	@echo "Installing Lux EVM plugin to $(PLUGIN_PATH)..."
	@mkdir -p $(dir $(PLUGIN_PATH))
	@cp build/$(BINARY_NAME) $(PLUGIN_PATH)
	@echo "Successfully installed to $(PLUGIN_PATH)"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Generate mocks
.PHONY: mocks
mocks:
	@echo "Generating mocks..."
	@if command -v mockgen >/dev/null 2>&1; then \
		go generate ./...; \
	else \
		echo "mockgen not installed. Install with: go install github.com/golang/mock/mockgen@latest"; \
		exit 1; \
	fi

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Run security checks
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi

# Run static analysis
.PHONY: staticcheck
staticcheck:
	@echo "Running static analysis..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
		exit 1; \
	fi

# Update dependencies
.PHONY: deps
deps:
	@echo "Updating dependencies..."
	go mod download
	go mod tidy

# Verify dependencies
.PHONY: verify
verify:
	@echo "Verifying dependencies..."
	go mod verify

# Run all checks (fmt, lint, test)
.PHONY: check
check: fmt lint test

# Display version information
.PHONY: version
version:
	@echo "Lux EVM"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(shell go version)"
	@echo "OS/Arch: $(GOOS)/$(GOARCH)"

# Help message
.PHONY: help
help:
	@echo "Lux EVM Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Build the EVM binary (default)"
	@echo "  build        - Build the EVM binary"
	@echo "  install      - Build and install as Lux plugin"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  bench        - Run benchmarks"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  mocks        - Generate mocks"
	@echo "  clean        - Clean build artifacts"
	@echo "  security     - Run security checks"
	@echo "  staticcheck  - Run static analysis"
	@echo "  deps         - Update dependencies"
	@echo "  verify       - Verify dependencies"
	@echo "  check        - Run fmt, lint, and test"
	@echo "  version      - Display version information"
	@echo "  help         - Display this help message"