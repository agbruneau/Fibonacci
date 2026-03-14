# Makefile for Fibonacci Calculator
# ===================================

# Variables
BINARY_NAME=fibcalc
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WIN=$(BINARY_NAME).exe
BUILD_DIR=./build
CMD_DIR=./cmd/fibcalc
GO=go

# PGO Profile paths
PGO_PROFILE=$(CMD_DIR)/default.pgo
PGO_RAW_PROFILE=$(BUILD_DIR)/cpu.prof

# Version information (can be overridden via environment variables)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags for version injection
LDFLAGS=-ldflags="-s -w \
	-X github.com/agbru/fibcalc/internal/app.Version=$(VERSION) \
	-X github.com/agbru/fibcalc/internal/app.Commit=$(COMMIT) \
	-X github.com/agbru/fibcalc/internal/app.BuildDate=$(BUILD_DATE)"
GOFLAGS=$(LDFLAGS)

.PHONY: all build clean test coverage benchmark run help install lint format check pgo-profile pgo-check pgo-clean pgo-rebuild build-pgo-linux build-pgo-windows build-pgo-darwin build-pgo-all generate-mocks install-mockgen

# Default target
all: clean build test

## build: Build the application for current platform (uses PGO if profile exists)
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f $(PGO_PROFILE) ]; then \
		echo "PGO profile found, building with PGO..."; \
		$(GO) build $(GOFLAGS) -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR); \
	else \
		$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR); \
	fi
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## pgo-profile: Generate CPU profile from benchmarks for PGO
pgo-profile:
	@echo "Generating CPU profile for PGO..."
	@mkdir -p $(BUILD_DIR)
	$(GO) test -cpuprofile=$(PGO_RAW_PROFILE) -bench=BenchmarkFastDoubling -benchtime=5s -count=3 ./internal/fibonacci/
	@if [ -f $(PGO_RAW_PROFILE) ]; then \
		mv $(PGO_RAW_PROFILE) $(PGO_PROFILE); \
		echo "Profile generated: $(PGO_PROFILE)"; \
	else \
		echo "Error: Profile generation failed"; \
		exit 1; \
	fi

## pgo-check: Verify PGO profile exists and is valid
pgo-check:
	@if [ ! -f $(PGO_PROFILE) ]; then \
		echo "Error: PGO profile not found at $(PGO_PROFILE)"; \
		echo "Run 'make pgo-profile' to generate it"; \
		exit 1; \
	fi
	@echo "PGO profile found: $(PGO_PROFILE)"

## build-pgo: Build with Profile-Guided Optimization (PGO)
build-pgo: pgo-check
	@echo "Building $(BINARY_NAME) with PGO..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "PGO Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-pgo-linux: Build for Linux with PGO
build-pgo-linux: pgo-check
	@echo "Building for Linux with PGO..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_UNIX) $(CMD_DIR)

## build-pgo-windows: Build for Windows with PGO
build-pgo-windows: pgo-check
	@echo "Building for Windows with PGO..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_WIN) $(CMD_DIR)

## build-pgo-darwin: Build for macOS with PGO
build-pgo-darwin: pgo-check
	@echo "Building for macOS with PGO..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64 $(CMD_DIR)

## build-pgo-all: Build for all platforms with PGO
build-pgo-all: build-pgo-linux build-pgo-windows build-pgo-darwin

## pgo-rebuild: Regenerate profile and build with PGO (full workflow)
pgo-rebuild: pgo-profile build-pgo
	@echo "PGO rebuild complete!"

## pgo-clean: Clean PGO profile and related artifacts
pgo-clean:
	@echo "Cleaning PGO artifacts..."
	@rm -f $(PGO_PROFILE) $(PGO_RAW_PROFILE)
	@echo "PGO clean complete"

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

## run-calibrate: Run calibration mode
run-calibrate: build
	$(BUILD_DIR)/$(BINARY_NAME) --calibrate

## lint: Run linter (golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

## security: Run security audit (gosec)
security:
	@echo "Running security audit..."
	@gosec ./...

## install-tools: Install development tools (golangci-lint, gosec)
install-tools:
	@echo "Installing tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest

## generate-mocks: Generate mock implementations for all interfaces
generate-mocks:
	@echo "Generating mocks..."
	@go generate ./...

## install-mockgen: Install mockgen tool for mock generation
install-mockgen:
	@echo "Installing mockgen..."
	@go install go.uber.org/mock/mockgen@latest

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

## help: Display this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
