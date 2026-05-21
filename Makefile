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

.PHONY: all build build-pgo build-all build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin clean test test-short coverage benchmark bench-baseline bench-versioned stats run run-fast run-calibrate help version install install-tools lint security format check tidy deps upgrade pgo-profile pgo-check pgo-clean pgo-rebuild build-pgo-linux build-pgo-windows build-pgo-darwin build-pgo-all

# Default target
all: clean build test

## build: Build the application for current platform (uses PGO if profile exists)
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@if [ -f $(PGO_PROFILE) ]; then \
		echo "PGO profile found, building with PGO..."; \
		$(GO) build $(GOFLAGS) -trimpath -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR); \
	else \
		$(GO) build $(GOFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR); \
	fi
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# POSIX-only (requires bash/date/tee)
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
	$(GO) build $(GOFLAGS) -trimpath -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "PGO Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-pgo-linux: Build for Linux with PGO
build-pgo-linux: pgo-check
	@echo "Building for Linux with PGO..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_UNIX) $(CMD_DIR)

## build-pgo-windows: Build for Windows with PGO
build-pgo-windows: pgo-check
	@echo "Building for Windows with PGO..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_WIN) $(CMD_DIR)

## build-pgo-darwin: Build for macOS with PGO
build-pgo-darwin: pgo-check
	@echo "Building for macOS with PGO..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -trimpath -pgo=$(PGO_PROFILE) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64 $(CMD_DIR)

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

## build-all: Build for all platforms (linux/windows amd64+arm64, macOS amd64+arm64)
build-all: build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin

## build-linux: Build for Linux (amd64)
build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_UNIX) $(CMD_DIR)

## build-linux-arm64: Build for Linux (arm64)
build-linux-arm64:
	@echo "Building for Linux (arm64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)_linux_arm64 $(CMD_DIR)

## build-windows: Build for Windows (amd64)
build-windows:
	@echo "Building for Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_WIN) $(CMD_DIR)

## build-windows-arm64: Build for Windows (arm64)
build-windows-arm64:
	@echo "Building for Windows (arm64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=arm64 $(GO) build $(GOFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)_windows_arm64.exe $(CMD_DIR)

## build-darwin: Build for macOS (amd64 and arm64)
build-darwin:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 $(CMD_DIR)
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64 $(CMD_DIR)

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

## stats: Print package and LOC counts (canonical source for CLAUDE.md / ARCH.md)
##
## Audit-PRD P2-05 / Sprint S4 — replaces the hardcoded "23 packages" /
## "35 500 LOC" numbers that drift each refactor. Run this before
## refreshing the documentation if the counts look stale.
stats:
	@echo "── Package count ──"
	@printf "Total Go packages:   "; $(GO) list ./... | wc -l
	@printf "Under internal/:     "; $(GO) list ./internal/... | wc -l
	@printf "Under cmd/:          "; $(GO) list ./cmd/... | wc -l
	@echo ""
	@echo "── Lines of code (excluding _test.go) ──"
	@find . -name '*.go' -not -name '*_test.go' -not -path './.understand-anything/*' -not -path './docs/dashboard/*' -print0 \
		| xargs -0 wc -l 2>/dev/null \
		| tail -1 \
		| awk '{print "Production LOC: "$$1}'
	@echo "── Lines of code (test files only) ──"
	@find . -name '*_test.go' -print0 \
		| xargs -0 wc -l 2>/dev/null \
		| tail -1 \
		| awk '{print "Test LOC:       "$$1}'

# POSIX-only (requires bash/date/tee)
## bench-baseline: Refresh docs/audits/bench-baseline.txt for the CI regression gate
##
## The CI bench job (.github/workflows/ci.yml) compares each PR against
## this baseline at 5% threshold via benchstat. Run this target on a
## quiet machine before bumping the baseline.
bench-baseline:
	@echo "Refreshing docs/audits/bench-baseline.txt ..."
	@{ \
		echo "goos: $$(go env GOOS)"; \
		echo "goarch: $$(go env GOARCH)"; \
		echo "pkg: github.com/agbru/fibcalc/internal/fibonacci"; \
		echo "cpu: baseline-$$(date -u +%Y-%m-%d)"; \
		$(GO) test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' \
			-benchmem -run='^$$' -count=5 -benchtime=1x ./internal/fibonacci/ \
			| grep -E '^Benchmark'; \
	} > docs/audits/bench-baseline.txt
	@echo "Baseline written. Review the diff and commit."

## bench-versioned: Comparable benchmark run with Go version and Git revision (see docs/PERFORMANCE.md)
##
## The -bench regex targets the three algorithmic subtests of
## BenchmarkFibonacci (FastDoubling / MatrixExp / FFTBased), so a single
## snapshot covers every algorithm (P3-03).
bench-versioned:
	@echo "Recording versioned benchmark snapshot to $(BUILD_DIR)/bench/"
	@mkdir -p $(BUILD_DIR)/bench
	@{ \
		echo "=== FibCalc benchmark snapshot ==="; \
		echo "Date (UTC): $$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)"; \
		echo "Git revision: $$(git rev-parse HEAD 2>/dev/null || echo unknown)"; \
		echo "Git describe: $$(git describe --tags --always --dirty 2>/dev/null || echo unknown)"; \
		echo "Go version: $$($(GO) version)"; \
		echo ""; \
		echo "Command: $(GO) test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -count=3 -benchtime=2s ./internal/fibonacci/"; \
		echo ""; \
		$(GO) test -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchmem -count=3 -benchtime=2s ./internal/fibonacci/; \
	} | tee $(BUILD_DIR)/bench/snapshot-$$(date -u +%Y%m%d-%H%M%SZ 2>/dev/null || echo manual).txt

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
	@awk 'BEGIN {FS = ":[[:space:]]*"} /^##[[:space:]]+/ {sub(/^##[[:space:]]+/, "", $$0); split($$0, a, ":[[:space:]]*"); printf "  %-24s %s\n", a[1], substr($$0, length(a[1]) + 2)}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
