# Copyright (C) 2024, Lux Partners Limited. All rights reserved.
# See the file LICENSE for licensing terms.

.PHONY: all build test lint clean install

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint
GOVET=$(GOCMD) vet

# Build parameters
BINARY_NAME=lux-sdk
VERSION?=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Directories
SRC_DIR=.
TEST_DIR=./tests
EXAMPLES_DIR=./examples

all: clean lint test build

build:
	@echo "Building SDK..."
	$(GOBUILD) -v $(LDFLAGS) -o $(BINARY_NAME) $(SRC_DIR)
	@echo "Building examples..."
	cd $(EXAMPLES_DIR)/boot-mainnet && $(GOBUILD) -v ./...

test:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./blockchain ./network ./heap ./crypto ./internal/...

test-integration:
	@echo "Running full integration tests..."
	$(GOTEST) -v -tags=integration -timeout=30m $(TEST_DIR)

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem -run=^$$ ./... | tee benchmark.txt

lint:
	@echo "Running linters..."
	$(GOFMT) -s -w .
	$(GOVET) ./...
	$(GOLINT) run --timeout=10m

lint-fix:
	@echo "Fixing lint issues..."
	$(GOFMT) -s -w .
	$(GOLINT) run --fix

deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang/mock/mockgen@latest
	go install gotest.tools/gotestsum@latest

generate:
	@echo "Generating code..."
	go generate ./...

clean:
	@echo "Cleaning..."
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -f benchmark.txt

install: build
	@echo "Installing SDK..."
	$(GOCMD) install $(LDFLAGS) $(SRC_DIR)

docker-build:
	@echo "Building Docker image..."
	docker build -t luxfi/sdk:$(VERSION) -t luxfi/sdk:latest .

docker-test:
	@echo "Running tests in Docker..."
	docker-compose -f tests/docker-compose.yml up --build --abort-on-container-exit
	docker-compose -f tests/docker-compose.yml down

ci-local:
	@echo "Running CI checks locally..."
	make lint
	make test
	make test-coverage
	make build

release-dry-run:
	@echo "Running release dry run..."
	goreleaser release --snapshot --skip-publish --rm-dist

help:
	@echo "Available targets:"
	@echo "  all              - Clean, lint, test, and build"
	@echo "  build            - Build the SDK and examples"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  benchmark        - Run benchmarks"
	@echo "  lint             - Run linters"
	@echo "  lint-fix         - Fix lint issues"
	@echo "  deps             - Download dependencies"
	@echo "  deps-update      - Update dependencies"
	@echo "  install-tools    - Install development tools"
	@echo "  generate         - Generate code"
	@echo "  clean            - Clean build artifacts"
	@echo "  install          - Install the SDK"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-test      - Run tests in Docker"
	@echo "  ci-local         - Run CI checks locally"
	@echo "  help             - Show this help message"