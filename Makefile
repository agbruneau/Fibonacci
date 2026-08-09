# Makefile for Fibonacci Calculator
# ===================================
# POSIX/WSL only: every recipe uses a POSIX shell ([ -f ], mkdir -p, ...).
# On Windows, run via WSL (`wsl make ...`); the native gate is scripts/check.ps1.

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
	-X github.com/agbruneau/FibGo/internal/app.Version=$(VERSION) \
	-X github.com/agbruneau/FibGo/internal/app.Commit=$(COMMIT) \
	-X github.com/agbruneau/FibGo/internal/app.BuildDate=$(BUILD_DATE)"
GOFLAGS=$(LDFLAGS)

.PHONY: all build build-pgo build-all build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin clean test test-win test-short coverage coverage-check benchmark bench-baseline bench-versioned stats run run-fast run-calibrate help version install install-tools lint security format check tidy deps upgrade pgo-profile pgo-check pgo-clean pgo-rebuild build-pgo-linux build-pgo-windows build-pgo-darwin build-pgo-all

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
	$(GO) test -cpuprofile=$(PGO_RAW_PROFILE) -bench='BenchmarkFibonacci/(FastDoubling|MatrixExp|FFTBased)' -benchtime=5s -count=3 -run='^$$' ./internal/fibonacci/
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

## test-win: Run all tests WITHOUT the race detector (Windows / no-CGO hosts)
test-win:
	@echo "Running tests (no -race; Windows/no-CGO)..."
	$(GO) test -v -cover ./...

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

# POSIX-only (requires awk)
## coverage-check: Fail if total coverage drops below the floor (single source: scripts/check.sh step 5)
coverage-check:
	@bash scripts/check.sh --coverage-only

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./internal/fibonacci/

# POSIX-only (requires find/xargs/wc/awk)
## stats: Print package and LOC counts (canonical source for docs/ARCH.md)
##
## Replaces the hardcoded package/LOC numbers that drift on each refactor.
## Run this before refreshing the documentation if the counts look stale.
stats:
	@echo "── Package count ──"
	@printf "Total Go packages:   "; $(GO) list ./... | wc -l
	@printf "Under internal/:     "; $(GO) list ./internal/... | wc -l
	@printf "Under cmd/:          "; $(GO) list ./cmd/... | wc -l
	@echo ""
	@echo "── Lines of code (excluding _test.go) ──"
	@find . -name '*.go' -not -name '*_test.go' -not -path './.understand-anything/*' -print0 \
		| xargs -0 wc -l 2>/dev/null \
		| tail -1 \
		| awk '{print "Production LOC: "$$1}'
	@echo "── Lines of code (test files only) ──"
	@find . -name '*_test.go' -print0 \
		| xargs -0 wc -l 2>/dev/null \
		| tail -1 \
		| awk '{print "Test LOC:       "$$1}'

# POSIX-only (requires bash/date/tee)
## bench-baseline: Refresh docs/audits/bench-baseline.txt regression baseline
# Protocol note: -benchtime=1x keeps the pool/arena warm-up outlier in the
# samples (~46% intra-sample scatter observed), which widens benchstat CIs and
# desensitizes the 5% gate. At the NEXT justified regeneration, prefer
# -benchtime=3x (or -count=6 and drop the first sample). Do not regenerate the
# baseline without a cause: it is the reference the 5% perf gate compares to.
##
## Use benchstat locally to compare new runs against this baseline at the
## documented 5% threshold (see docs/PERFORMANCE.md). Run this target on a
## quiet machine before bumping the baseline.
bench-baseline:
	@echo "Refreshing docs/audits/bench-baseline.txt ..."
	@mkdir -p docs/audits
	@{ \
		echo "goos: $$(go env GOOS)"; \
		echo "goarch: $$(go env GOARCH)"; \
		echo "pkg: github.com/agbruneau/FibGo/internal/fibonacci"; \
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
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 # pinned: .golangci.yml is written for v1 (schema v1, last v1 tag)
	@go install github.com/securego/gosec/v2/cmd/gosec@latest

## format: Format Go code
format:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

## check: Run the canonical pre-commit gate (delegates to scripts/check.sh)
check:
	@bash scripts/check.sh

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
