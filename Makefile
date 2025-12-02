# Makefile for Fibonacci Calculator
# ===================================

# Variables
BINARY_NAME=fibcalc
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WIN=$(BINARY_NAME).exe
BUILD_DIR=./build
CMD_DIR=./cmd/fibcalc
GO=go

# Version information (can be overridden via environment variables)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags for version injection
LDFLAGS=-ldflags="-s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildDate=$(BUILD_DATE)"
GOFLAGS=$(LDFLAGS)

.PHONY: all build clean test coverage benchmark run help install lint format check

# Default target
all: clean build test

## build: Build the application for current platform
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-pgo: Build with Profile-Guided Optimization (PGO)
build-pgo:
	@echo "Building $(BINARY_NAME) with PGO..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -pgo=./cmd/fibcalc/default.pgo -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "PGO Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## version: Display version information
version: build
	@$(BUILD_DIR)/$(BINARY_NAME) --version

## build-all: Build for all platforms
build-all: build-linux build-windows build-darwin

## build-linux: Build for Linux (amd64)
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) $(CMD_DIR)

## build-windows: Build for Windows (amd64)
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_WIN) $(CMD_DIR)

## build-darwin: Build for macOS (amd64 and arm64)
build-darwin:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 $(CMD_DIR)
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64 $(CMD_DIR)

## install: Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(CMD_DIR)

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@$(GO) clean
	@echo "Clean complete"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

## test-short: Run tests without slow ones
test-short:
	@echo "Running short tests..."
	$(GO) test -v -short ./...

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./internal/fibonacci/

## run: Build and run the application with default settings
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

## run-fast: Quick run with small n value
run-fast: build
	$(BUILD_DIR)/$(BINARY_NAME) -n 1000 -algo fast -d

## run-server: Run in server mode
run-server: build
	$(BUILD_DIR)/$(BINARY_NAME) --server --port 8080

## run-calibrate: Run calibration mode
run-calibrate: build
	$(BUILD_DIR)/$(BINARY_NAME) --calibrate

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

## format: Format Go code
format:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

## check: Run all checks (format, lint, test)
check: format lint test
	@echo "All checks passed!"

## tidy: Tidy up go.mod and go.sum
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy
	$(GO) mod verify

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

## upgrade: Upgrade dependencies
upgrade:
	@echo "Upgrading dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 $(BINARY_NAME):$(VERSION)

## help: Display this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
