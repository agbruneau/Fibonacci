# Testing Strategy

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 744 nodes / 8 layers / 13-step tour)

## Overview

The Fibonacci Calculator project uses a layered testing strategy that
combines unit tests, golden file validation, fuzz testing, property-based
testing, panic-contract testing, an architecture-layering gate, benchmark
testing, and end-to-end testing. The test suite contains 100+ test files
distributed across all packages, with a coverage target of 80 % verified
locally via `make coverage` / `make coverage-check` (do not freeze a
percentage here — run the commands for the current figure; see A5-04 below).

All tests follow standard Go conventions: table-driven subtests, `t.Parallel()` for independent cases, and the `-race` flag run locally (requires CGO).

## Quick Reference Commands

```bash
go test -v -race -cover ./...                          # All tests with race detector
go test -v -short ./...                                # Skip slow tests
go test -v -run TestFastDoubling ./internal/fibonacci/  # Single test
go test -bench=. -benchmem ./internal/fibonacci/        # Benchmarks
go test -fuzz=FuzzFastDoubling ./internal/fibonacci/    # Fuzz tests
```

Makefile targets (require `make`):

```bash
make test              # go test -v -race -cover ./...
make coverage          # Generate coverage.html
make check             # format + lint + test
```

## Table-Driven Unit Tests

The standard test pattern uses table-driven subtests with `t.Parallel()`. Every algorithm is validated against a shared test oracle (`knownFibResults`) with reference values from F(0) through F(1000).

```go
var knownFibResults = []struct {
    n      uint64
    result string
}{
    {0, "0"}, {1, "1"}, {2, "1"}, {10, "55"}, {20, "6765"},
    {50, "12586269025"},
    {92, "7540113804746346429"},      // Near max uint64
    {93, "12200160415121876738"},     // Max Fibonacci that fits in uint64
    {100, "354224848179261915075"},
    {128, "251728825683549488150424261"},        // Power of 2
    {256, "14169381771405651323470996587..."},   // Power of 2
    {1000, "43466557686937456435688527..."},     // 209 digits
}

func TestFibonacciCalculators(t *testing.T) {
    calculators := map[string]Calculator{
        "FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
        "MatrixExp":    NewCalculator(&MatrixExponentiation{}),
        "FFTBased":     NewCalculator(&FFTBasedCalculator{}),
    }
    for name, calc := range calculators {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            for _, tc := range knownFibResults {
                t.Run(fmt.Sprintf("N=%d", tc.n), func(t *testing.T) {
                    t.Parallel()
                    expected := new(big.Int)
                    expected.SetString(tc.result, 10)
                    got, err := calc.Calculate(ctx, nil, 0, tc.n, opts)
                    if err != nil {
                        t.Fatalf("Unexpected error: %v", err)
                    }
                    if got.Cmp(expected) != 0 {
                        t.Errorf("Expected: %s\nGot: %s",
                            expected.String(), got.String())
                    }
                })
            }
        })
    }
}
```

Key conventions:

- All three calculators (Fast Doubling, Matrix Exponentiation, FFT-Based) run against the same oracle
- Subtests are named `N=<value>` for clear identification in failure output
- `t.Parallel()` is used at both the calculator and individual test case level
- Edge cases include F(0), F(1), powers of 2 (N=64, 128, 256), and uint64 overflow boundaries (N=92, 93, 94)

## Golden File Tests

Golden file testing validates all calculators against precomputed Fibonacci values stored in JSON.

| File | Purpose |
|------|---------|
| `internal/fibonacci/testdata/fibonacci_golden.json` | Canonical golden data (N and result pairs) |
| `cmd/generate-golden/main.go` | Generator tool for rebuilding golden data |
| `internal/fibonacci/fibonacci_golden_test.go` | Validates all 3 calculators against golden data |
| `internal/cli/goldens_test.go` | Golden tests for CLI output formatting |

The golden file is a JSON array of `{"n": <uint64>, "result": "<decimal string>"}` entries. The test loads it, then runs each calculator against every entry with `t.Parallel()`.

### Regeneration

```bash
go run ./cmd/generate-golden/
```

This rebuilds `fibonacci_golden.json` using Fast Doubling as the reference implementation.

### CLI Output Goldens

The CLI package has separate golden tests (`goldens_test.go`) that validate exact output formatting. These disable color output with `ui.InitTheme(false)` and use `testutil.StripAnsiCodes()` for deterministic comparison.

## Fuzz Testing

Seven fuzz targets use Go's built-in fuzzing framework (`testing.F`) to
explore the input space beyond manual test cases.

| Fuzz Test | Package | Strategy | Input Limit |
|-----------|---------|----------|-------------|
| `FuzzFastDoublingConsistency` | `internal/fibonacci` | Cross-validates Fast Doubling vs Matrix | n up to **200 000** (raised from 50 000 to exercise the FFT regime) |
| `FuzzFFTBasedConsistency` | `internal/fibonacci` | Cross-validates FFT vs Fast Doubling | n up to **200 000** (raised from 20 000) |
| `FuzzFibonacciIdentities` | `internal/fibonacci` | Verifies mathematical identities | n up to 10,000 |
| `FuzzProgressMonotonicity` | `internal/fibonacci` | Ensures progress is monotonically increasing | n 10 to 20,000 |
| `FuzzFastDoublingMod` | `internal/fibonacci` | Validates modular Fast Doubling output range | n up to 100,000, mod up to 1B |
| `FuzzMul` | `internal/bigfft` | Cross-validates `bigfft.Mul` against `math/big.Int.Mul` (seed corpus pushes past the FFT threshold) | operand size up to 32 000 bytes |
| `FuzzSqr` | `internal/bigfft` | Cross-validates `bigfft.Sqr` against `math/big` squaring | operand size up to 32 000 bytes |

Fibonacci targets live in `internal/fibonacci/fibonacci_fuzz_test.go`;
`bigfft` targets in `internal/bigfft/fft_fuzz_test.go`.

### Running Fuzz Tests

```bash
go test -fuzz=FuzzFastDoublingConsistency -fuzztime=30s ./internal/fibonacci/
go test -fuzz=FuzzFFTBasedConsistency -fuzztime=1m ./internal/fibonacci/
```

### Mathematical Identities Verified

`FuzzFibonacciIdentities` checks two properties:

1. **Doubling identity**: `F(2n) = F(n) * (2*F(n+1) - F(n))`
2. **d'Ocagne's identity**: `|F(m)*F(n+1) - F(m+1)*F(n)| = F(n-m)` for n > m

These provide independent verification without comparing two calculator implementations.

Each fuzz test is seeded with known interesting values (0, 1, 92, 93, 1000, 5000) to guide the fuzzer toward productive exploration.

## Architecture-Layering Gate

`internal/arch_test.go` is a runtime sentinel that fails the build if a
forbidden upward import is reintroduced. It inspects each importer
package via `go list -f '{{.Imports}}'` (production code only — `_test.go`
files are excluded). Currently three rules :

| Importer | Forbidden direct import | Rationale |
|---|---|---|
| `internal/fibonacci/threshold` | `internal/config` | Would close a cycle through `config → fibonacci/memory`. The threshold package consumes `Tuning` via `SetTuning`. |
| `internal/errors` | `internal/format` | Leaf utility ; uses local `formatBytesLocal` instead. |
| `internal/tui` (production) | `internal/fibonacci` | UI must reach domain types through `orchestration.Calculator`/`Options` aliases. |

Adding a new rule is a one-line append to `architectureRules`.

## Panic-Contract Tests

`internal/bigfft` distinguishes two panic classes :

- **Pre-conditions** (operand-size mismatches): documented and exercised
  by `TestFermatPanicSites` (`fermat_panic_test.go`). The four entry
  points convert these to errors via the `recover()` handlers.
- **Post-conditions** (algorithmic invariants like
  `"unexpected carry after normalization"`): re-propagated via
  `panic(r)` so genuine bugs are not silently coerced into opaque
  errors. `TestFermatPostConditionPanicClassifier` and
  `TestMulRepanicsOnPostCondition` guard the sentinel list and the
  classifier behaviour. See [`docs/adr/0002-recover-strategy.md`](adr/0002-recover-strategy.md).

## Property-Based Testing (gopter)

Property-based testing uses `github.com/leanovate/gopter` to verify mathematical properties with randomly generated inputs.

File: `internal/fibonacci/fibonacci_property_test.go`

### Cassini's Identity

The primary property tested is Cassini's Identity:

```
F(n-1) * F(n+1) - F(n)^2 = (-1)^n
```

This identity holds for all positive integers and provides a correctness guarantee independent of any reference implementation.

```go
parameters := gopter.DefaultTestParameters()
parameters.MinSuccessfulTests = 100
properties := gopter.NewProperties(parameters)

for _, calculator := range calculators {
    properties.Property(
        calculator.Name()+" satisfies Cassini's Identity",
        prop.ForAll(func(n uint64) bool {
            // Calculate F(n-1), F(n), F(n+1)
            // Verify: F(n-1)*F(n+1) - F(n)^2 == (-1)^n
        }, gen.UInt64Range(1, 25000)),
    )
}
```

- **MinSuccessfulTests**: 100 per property per calculator (300 total)
- **Input range**: n from 1 to 25,000
- **All 3 calculators** verified independently

## Benchmark Testing

Benchmarks measure algorithm performance across input sizes, reporting wall-clock time and memory allocations.

### Running Benchmarks

```bash
go test -bench=. -benchmem ./internal/fibonacci/
go test -bench=BenchmarkFastDoubling -benchmem ./internal/fibonacci/
go test -bench='Benchmark(FastDoubling|Matrix|FFT)' -benchmem ./internal/fibonacci/
go test -bench=BenchmarkFibonacci -benchtime=5x ./internal/fibonacci/
go test -bench=BenchmarkCacheImpact -benchmem ./internal/fibonacci/
```

Benchmarks are organized as nested subtests (`Calculator/Size`), testing F(1M) and F(10M) across all three calculators. Each uses `b.ReportAllocs()` and `b.ResetTimer()` for accurate measurement.

### Profiling

```bash
go test -cpuprofile=cpu.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/
go tool pprof cpu.prof

go test -memprofile=mem.prof -bench=BenchmarkFastDoubling ./internal/fibonacci/
go tool pprof mem.prof

go test -trace=trace.out -bench=BenchmarkFastDoubling ./internal/fibonacci/
go tool trace trace.out
```

## Mock Generation

The project currently uses hand-written mocks and spies. There is no `mockgen`
wiring in the codebase: no `//go:generate mockgen ...` directives, no
`mocks/` directories, and no dependency on `go.uber.org/mock`. The previous
`make generate-mocks` / `make install-mockgen` Makefile targets were removed
because they produced no output.

If mockgen is later required, re-introduce the Makefile targets together
with the first `//go:generate mockgen` directive.

### Spy-Based Testing

In addition to generated mocks, the orchestration package uses hand-written spy implementations for focused integration tests:

```go
type SpyCalculator struct {
    capturedOpts fibonacci.Options
}

func (s *SpyCalculator) Calculate(ctx context.Context,
    progressChan chan<- progress.ProgressUpdate,
    calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
    s.capturedOpts = opts
    return big.NewInt(55), nil
}
```

This pattern in `orchestration_spy_test.go` verifies that configuration values (such as `StrassenThreshold`) propagate correctly through the orchestration layer.

## Coverage

The project targets a minimum total coverage of 80 % verified locally via `make coverage`; the suite currently sits well above that floor.

```bash
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out    # Coverage by function
go test -cover ./...                # Quick summary
make coverage                       # Via Makefile
```

The HTML report (`coverage.html`) highlights tested and untested code paths. Focus areas: algorithm implementations, output formatting, error handling paths, and configuration parsing.

> **Single source of the metric (A5-04)** — The coverage figure is *not* hard-coded in this document. The canonical sources are `make coverage` (HTML report) and `make coverage-check` (floor assertion). Do not quote a frozen percentage here; run the commands to obtain the current value.

### Coverage blind spots (A5-08)

Two categories of code are intentionally not reflected in the standard `coverage.out` totals:

1. **E2E subprocess paths** — The e2e tests (`test/e2e/`) build and execute the `fibcalc` binary as a subprocess (black-box). The `go test` coverage instrumentation does not see code run in a separate process, so these packages are reported as `[no statements]` and the CLI paths the e2e suite exercises are **not** counted in the module total. Measuring them would require building the binary with `go build -cover` (Go 1.20+) and aggregating the per-run profiles emitted to `GOCOVERDIR`. This is **not** wired up today — it is a known, documented limitation.
2. **GMP backend** — `internal/fibonacci/calculator_gmp.go` is guarded by `//go:build gmp` and requires CGO plus `libgmp`. It is **not** built by default, so it reports **0 %** on hosts without GMP. Validate it on a Linux runner with `libgmp-dev` installed and the `gmp` build tag enabled.

### generate-golden is sparsely covered (A5-09)

`cmd/generate-golden` is the **dev-time** oracle that regenerates the golden corpus; it is outside the production execution path. Its `main` is deliberately left uncovered (~29 %). **Exclude it from any per-package coverage floor** — the floor applies to the **module total** only (see `make coverage-check`, A5-10).

## End-to-End Testing

Files: `test/e2e/cli_e2e_test.go`, `test/e2e/extended_e2e_test.go`

E2E tests build the actual binary into a temporary directory and execute it as a subprocess, verifying complete program behavior including flag parsing, output formatting, and exit codes. Tests set `NO_COLOR=1` for deterministic output.

```go
func TestCLI_E2E(t *testing.T) {
    tmpDir := t.TempDir()
    binPath := filepath.Join(tmpDir, "fibcalc")
    cmd := exec.Command("go", "build", "-o", binPath, "./cmd/fibcalc")
    cmd.Dir = "../.."
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to build fibcalc: %v", err)
    }

    tests := []struct {
        name     string
        args     []string
        wantOut  string
        wantCode int
    }{
        {"Basic Calculation", []string{"-n", "10", "-c"}, "F(10) = 55", 0},
        {"Help", []string{"--help"}, "usage", 0},
    }
    // Execute binary for each test case, validate output and exit code
}
```

```bash
go test -v ./test/e2e/
```

## Local Validation Guardrails (A5-02 + A5-10)

There is **no remote CI** for this project (an assumed decision). Pre-commit validation rests entirely on **local guardrails** that the contributor is expected to run before each commit:

| Guardrail | Role |
|---|---|
| `scripts/check.ps1` / `scripts/check.sh` | One-shot pre-commit aggregator (lint + tests + coverage floor), Windows (PowerShell) and POSIX (sh) variants |
| `make coverage-check` | Fails the run if **total** module coverage drops below 80 % (A5-10) |
| `make test-win` | Full test run **without** `-race` (Windows / no-CGO hosts) |
| `make test` | Full test run **with** `-race`, requires CGO/gcc (run via WSL or a Linux/macOS host on Windows) |

Because no remote gate enforces these, discipline is the only safeguard: run the appropriate `scripts/check.*` (or the underlying `make` targets) locally before committing.

## Test Organization

| Package | Key Test Files | Testing Approach |
|---------|---------------|-----------------|
| `internal/fibonacci` | `fibonacci_test.go`, `fibonacci_golden_test.go`, `fibonacci_fuzz_test.go`, `fibonacci_property_test.go`, `fibonacci_strassen_test.go`, `fibonacci_edge_test.go`, `modular_test.go`, `fastdoubling_test.go`, `registry_test.go`, `strategy_test.go` | Unit, golden, fuzz, property-based, Strassen correctness, modular arithmetic, Fast Doubling state pooling, calculator registry, strategy selection |
| `internal/fibonacci/memory` | `arena_test.go`, `arena_fallback_test.go`, `budget_test.go`, `gc_control_test.go` | Bump arena allocation, heap-fallback pre-sizing, memory-budget pre-flight estimation, GC controller |
| `internal/fibonacci/threshold` | `manager_test.go`, `tuning_test.go` | Threshold manager (parallelism / FFT / Strassen decisions), `SetTuning` propagation |
| `internal/bigfft` | `fft_precision_test.go`, `fft_parallel_test.go`, `pool_test.go`, `fermat_test.go`, `bump_test.go`, `fft_cache_test.go` | Unit, precision, parallel correctness, pool recycling, Fermat arithmetic, bump allocator, FFT cache |
| `internal/cli` | `output_test.go`, `ui_test.go`, `goldens_test.go`, `presenter_test.go` | Unit, golden output, result presentation |
| `internal/tui` | `model_test.go`, `bridge_test.go`, `header_test.go`, `chart_test.go`, `metrics_test.go`, `sparkline_test.go`, `footer_test.go`, `logs_test.go`, `keymap_test.go`, `cli_flags_test.go` | Unit, sub-model testing, message handling |
| `internal/orchestration` | `orchestrator_test.go`, `orchestration_spy_test.go`, `calculator_selection_test.go` | Integration, spy-based config propagation, calculator selection |
| `internal/calibration` | `calibration_test.go`, `calibration_advanced_test.go`, `adaptive_test.go`, `microbench_test.go`, `profile_test.go`, `io_test.go` | Unit, advanced calibration, micro-benchmark validation, profile I/O |
| `internal/config` | `config_test.go`, `config_exhaustive_test.go`, `env_test.go` | Unit, exhaustive flag combinations, env vars |
| `internal/errors` | `errors_test.go`, `handler_test.go` | Unit, exit code mapping |
| `internal/metrics` | `memory_test.go` | Unit (memory snapshot collection) |
| `internal/app` | `app_test.go`, `version_test.go` | Unit, lifecycle |
| `test/e2e` | `cli_e2e_test.go`, `extended_e2e_test.go` | End-to-end binary testing |
| `cmd/fibcalc` | `main_test.go` | Entry point smoke test |
| `cmd/generate-golden` | `main_test.go` | Golden generator validation |

## Strassen Algorithm Testing

File: `internal/fibonacci/fibonacci_strassen_test.go`

Tests for the Strassen matrix multiplication optimization:

| Test | Description |
|------|-------------|
| `TestStrassenConfiguration` | Verifies `Options.StrassenThreshold` is correctly accepted and applied |
| `TestStrassenThresholdEffect` | Tests that different threshold values produce correct results |
| `TestStrassenOptionsPrecedence` | Tests that `Options.StrassenThreshold` overrides the global default set by `SetDefaultStrassenThreshold()` |

```bash
go test -v -run TestStrassen ./internal/fibonacci/
```

## Fermat Arithmetic Testing

File: `internal/bigfft/fermat_test.go`

Tests for the Fermat ring arithmetic used by the FFT subsystem:

- Squaring vs multiplication equivalence (`x*x == x^2`)
- Edge cases: zero, one, maximum word values
- Size boundary testing around `smallMulThreshold`
- Modular reduction correctness

```bash
go test -v -run TestFermat ./internal/bigfft/
```

## Concurrency Testing

Several tests specifically target concurrent behavior:

- **Race detector**: Local `make test` runs `-race` to detect data races (requires CGO)
- **Context cancellation**: `TestContextCancellation` verifies algorithms respond to `context.WithTimeout` within 50ms for N=100M. Cancellation is **coarse-grained** (checked between doubling steps and between the 3 FFT products); fine-grained cancellation *inside* a single giant FFT multiplication is deferred to the `FFTContext` migration — see [`docs/adr/0006-fft-recursion-cancellation.md`](adr/0006-fft-recursion-cancellation.md).
- **Concurrent GC control**: `TestGCController_ConcurrentBeginEnd_RestoresOriginal` (`internal/fibonacci/memory/gc_control_test.go`) verifies the package-level refcount (`gcGlobalMu`/`gcActiveDepth`/`gcSavedPercent`) keeps GC disabled while any sibling calculator runs and restores the real `GOGC` exactly once — see [`docs/adr/0005-gc-control-concurrent.md`](adr/0005-gc-control-concurrent.md).
- **Progress monotonicity**: `TestProgressReporter` validates that progress updates across goroutine boundaries never decrease
- **Parallel FFT tests**: `fft_parallel_test.go` validates thread safety of the FFT subsystem under concurrent load

## TUI Testing

The TUI package is tested using the Bubble Tea model-update-view pattern. Tests create models, send messages, and assert on state without a real terminal.

```go
func newTestModel(t *testing.T) Model {
    t.Helper()
    ctx := context.Background()
    cfg := config.AppConfig{N: 1000, Timeout: time.Minute}
    m := NewModel(ctx, nil, cfg, "v0.1.0")
    t.Cleanup(m.cancel)
    return m
}
```

Each sub-model (header, chart, metrics, logs, footer) has its own test file validating rendering and state transitions independently.

## Writing New Tests

### Guidelines

1. Follow the table-driven pattern with subtests and descriptive names
2. Use `t.Parallel()` for independent subtests
3. Add golden file entries for new algorithms (regenerate with `go run ./cmd/generate-golden/`)
4. Write fuzz tests for cross-validation between algorithms
5. Run with `-race` during development
6. Use `-short` to skip slow tests during rapid iteration
7. Test context cancellation for any long-running computation

### Adding a New Algorithm Test

When implementing a new `coreCalculator`:

1. Add the calculator to `knownFibResults` tests in `fibonacci_test.go`
2. Add it to the golden file test in `fibonacci_golden_test.go`
3. Add a cross-validation fuzz test against an existing algorithm
4. Add it to the Cassini's Identity property-based test
5. Add benchmark entries in `BenchmarkFibonacci`
6. Register it in `NewDefaultFactory()` in `registry.go`

## Cross-References

- [Architecture](architecture/README.md) for package structure and interface definitions
- [PERFORMANCE.md](PERFORMANCE.md) for benchmark reference data and profiling guidance
- [algorithms/](algorithms/) for detailed algorithm documentation
