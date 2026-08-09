# Calibration System

> Interactive architecture map: **[agbruneau.github.io/Fibonacci/dashboard/](https://agbruneau.github.io/Fibonacci/dashboard/)** (knowledge graph, 1128 nodes / 4782 edges / 9 layers / 12-step tour, regenerated 2026-07-06 at commit 6e3ec29)

## Overview

The calibration system (`internal/calibration/`) determines optimal performance thresholds for the current hardware. Rather than relying on hard-coded constants, it benchmarks the system at runtime and selects the threshold values that yield the fastest execution for the active CPU, architecture, and core count.

It complements (does not replace) the **Dynamic Threshold Manager**
(`internal/fibonacci/threshold/`), which adjusts the FFT/parallel
thresholds *in-flight* during a calculation based on observed
per-iteration metrics. The two layers are kept separate because they
solve different problems : calibration produces a startup snapshot,
the manager reacts to per-run variance. Their relative value is
benchmarked via `BenchmarkFibonacciDTM` (`internal/fibonacci/dtm_bench_test.go`)
and analysed in [`docs/adr/0001-dtm-decision.md`](adr/0001-dtm-decision.md).

Three operational modes are supported:

| Mode | Flag | Latency | Description |
|------|------|---------|-------------|
| Full calibration | `--calibrate` | Seconds to minutes | Exhaustive threshold sweep with real Fibonacci calculations |
| Auto-calibration | `--auto-calibrate` | Instant to seconds | 3-tier fallback: cached profile, micro-benchmarks, full runner |
| Cached profile | `--calibration-profile` | Instant | Loads a previously saved JSON profile. The load itself is unconditional — `app.New` calls `LoadCachedCalibration` on every run; the flag only selects *which* file (default `~/.fibcalc_calibration.json`) |

## Quick Start

```bash
# Run full calibration (saves profile to ~/.fibcalc_calibration.json by default)
fibcalc --calibrate

# Full calibration saving to a custom profile path
fibcalc --calibrate --calibration-profile /path/to/profile.json

# Quick startup calibration with automatic fallback
fibcalc --auto-calibrate

# Use a specific profile file
fibcalc --calibration-profile /path/to/profile.json
```

After calibration completes, the optimal thresholds are applied to all subsequent calculations in the same invocation. The profile is saved to disk so future runs can skip benchmarking entirely.

## Calibrated Thresholds

The calibration system tunes three thresholds that control algorithm and concurrency dispatch in the `fibonacci.Options` struct:

| Threshold | Default | Unit | Description |
|-----------|---------|------|-------------|
| `ParallelThreshold` | 4096 | bits | Goroutine parallelism activation point for multiplication steps |
| `FFTThreshold` | 500,000 | bits | Crossover point from standard math/big to FFT multiplication |
| `StrassenThreshold` | 3072 | bits | Activation point for Strassen matrix multiplication |

These values interact with the 2-tier adaptive multiplication system described in [PERFORMANCE.md](PERFORMANCE.md):

```go
opts := fibonacci.Options{
    ParallelThreshold:  4096,
    FFTThreshold:       500_000,
    StrassenThreshold:  3072,
}
```

When operand bit sizes exceed a threshold, the corresponding optimization is activated. Setting a threshold to `0` means "use the package default" — `normalizeOptions()` rewrites `0` to `DefaultParallelThreshold`/`DefaultFFTThreshold`/`DefaultStrassenThreshold`. The genuine sequential sentinel is `-1` (FIB-02). Higher values delay activation to larger operand sizes.

## Calibration Modes

### Full Calibration

Entry point: `RunCalibration()` in `internal/calibration/calibration.go`.

Full calibration runs a real Fibonacci calculation (N=10,000,000, defined by `calibration.CalibrationN`) with the "fast" algorithm for every candidate threshold value. It measures wall-clock time for each run and selects the fastest.

The process:

1. `GenerateParallelThresholds()` produces a CPU-adaptive candidate list (see Adaptive Threshold Generation below).
2. For each candidate, the "fast" calculator runs `Calculate()` with that threshold.
3. Execution times are recorded in a `calibrationResult` slice.
4. The threshold that produced the shortest duration is selected.
5. FFT and Strassen thresholds are estimated via heuristics (`EstimateOptimalFFTThreshold()`, `EstimateOptimalStrassenThreshold()`).
6. Results are printed as a formatted table and the profile is saved to the path given by `--calibration-profile` (default: `~/.fibcalc_calibration.json`).

```
--- Calibration Summary ---
  Threshold      | Execution Time
  ──────────────┼─────────────────────────
  Sequential     | 3.842s
  512 bits       | 2.651s
  1024 bits      | 2.412s
  2048 bits      | 2.318s (Optimal)
  4096 bits      | 2.445s
```

### Auto-Calibration

Entry point: `AutoCalibrateWithProfile()` in `internal/calibration/calibration.go`.

Auto-calibration uses a 3-tier fallback strategy to minimize startup latency while still finding reasonable threshold values:

**Tier 1 -- Cached profile (instant)**

`LoadOrCreateProfile()` attempts to load a saved profile from disk. The cached thresholds are
applied immediately — no benchmarks executed — only when three conditions hold together
(`internal/calibration/calibration.go:AutoCalibrateWithProfile`): the profile exists and
`IsValid()` returns true; it is **not stale** (`IsStale(maxAge)`, see `FIBCALC_PROFILE_MAX_AGE`);
and its three `Optimal*Threshold` values are all `>= 0` (SEC-01 re-validation, because
`IsValid()` checks hardware only and never threshold ranges). A stale profile goes straight to
`CompleteStrategy`; out-of-range thresholds log `"Cached calibration profile has invalid
thresholds, re-calibrating"` and fall through to the escalation chain.

`IsValid()` checks **five** conditions, all of which must hold (`internal/calibration/profile.go:CalibrationProfile.IsValid`):

1. `ProfileVersion == CurrentProfileVersion` (3)
2. `NumCPU == runtime.NumCPU()`
3. `GOARCH == runtime.GOARCH`
4. `WordSize` matches the host's (32 or 64)
5. `CPUHeuristicKey == config.CurrentHardwareHeuristicKey()` — the `GOARCH`-plus-SIMD-class tag, e.g. `amd64-avx2`

**Tier 2 -- Quick micro-benchmarks (~100ms)**

If no valid cached profile exists, `QuickCalibrate()` from `microbench.go` runs rapid multiplication tests. If the resulting confidence score is >= 0.5, the thresholds are accepted and a profile is saved for future use.

**Tier 3 -- Full calibration runner**

If micro-benchmarks produce low confidence, `newCalibrationRunner()` executes targeted threshold searches:
- `findBestParallelThreshold()` with the "fast" calculator
- `findBestFFTThreshold()` with the "fast" calculator
- `findBestStrassenThreshold()` with the "matrix" calculator (if available)

Each method iterates over its candidate set (`GenerateQuickParallelThresholds()`, `GenerateFFTThresholds()`, `GenerateQuickStrassenThresholds()`). The profile is saved after successful calibration.

> **Architecture note:** the calibration flow is structured around the **Strategy pattern** — `CalibrationStrategy` (`strategy.go`) with concrete `FastStrategy` (`strategy_fast.go`, micro-benchmark tier) and `CompleteStrategy` (`strategy_complete.go`, full runner tier). `calibration.go` selects/escalates strategies, including the stale-profile branch (`IsStale` → `CompleteStrategy`). The `calibrationRunner` described under "Calibration Runner" is the execution helper invoked by `CompleteStrategy`, not a directly-called entry point.

### Cached Profile Loading

Entry point: `LoadCachedCalibration()` in `internal/calibration/calibration.go`.

This is the simplest mode. It loads an existing profile, validates it against the current hardware, and applies the thresholds to `config.AppConfig`. No benchmarks are executed. If the profile is missing or invalid, the function returns `false` and the caller falls back to default thresholds.

## Micro-Benchmarking Engine

File: `internal/calibration/microbench.go`

The micro-benchmarking engine provides rapid threshold estimation by testing raw multiplication performance rather than full Fibonacci calculations.

### Configuration

```go
const MicroBenchIterations = 3 // there is no per-test timeout constant

// MicroBenchTimeout is a var (not a const) sourced from
// config.DefaultThresholdTuning — the canonical value lives alongside the
// other dynamic-tuning knobs and can be re-pointed in tests.
var MicroBenchTimeout = config.DefaultThresholdTuning.MicroBenchTimeout // ~150ms default

var MicroBenchTestSizes = []int{500, 2000, 8000, 16000} // word counts
```

The test sizes are chosen to span the critical algorithm crossover ranges:

| Word Count | Approximate Bit Size | Region |
|------------|---------------------|--------|
| 500 | ~32K bits | Standard math/big territory |
| 2,000 | ~128K bits | Near parallel threshold |
| 8,000 | ~512K bits | Near FFT threshold |
| 16,000 | ~1M bits | FFT territory |

### Test Matrix

For each word size, four configurations are **enqueued** — but only two are
distinct workloads. `runSingleTest` opens with `_ = parallel`
(`internal/calibration/microbench.go:runSingleTest`) and never branches on the
flag, so rows 2 and 4 re-run rows 1 and 3 verbatim:

| # | Enqueued config | What actually runs |
|---|---|---|
| 1 | Standard math/big sequential | `new(big.Int).Mul(x, y)` |
| 2 | Standard math/big "parallel" | identical to 1 |
| 3 | FFT sequential | `bigfft.Mul(x, y)` |
| 4 | FFT "parallel" | identical to 3 |

The duplicated rows are not wasted budget: `findFFTCrossover` averages every
result sharing a `(wordSize, useFFT)` key, so rows 2 and 4 land in the same
average as rows 1 and 3 and act as a **second sample** of the same workload.
What they cannot support is a parallel-versus-sequential comparison.

The flag is kept deliberately (P1-07, see the `runSingleTest` doc comment) to
record the intent for future work; `analyzeResults` already treats any
"parallel crossover" as noise between two identical configurations and grants
it no confidence bonus — see [Confidence Scoring](#confidence-scoring) below.

Tests run in parallel with a semaphore limiting concurrency to `runtime.NumCPU()`. Each test generates deterministic `big.Int` operands via `generateTestNumber()`, performs a warm-up multiplication, then averages 3 timed iterations.

### Analysis

After all tests complete, the engine analyzes results:

- `findFFTCrossover()`: Identifies the smallest bit size where FFT multiplication is faster than standard `math/big`. Applies a 10% margin (multiplies the crossover by 9/10) to ensure FFT is clearly beneficial. Returns `0` when no crossover is observed — the conservative default (`FFTThreshold: 500000`) is owned by `analyzeResults()`, not by this function (FIB-03: a fallback is not a measurement).

- `findParallelCrossover()`: Compares the rows flagged `parallel` against those flagged sequential, keeping the smallest bit size where the former is at least 10% faster. Since `runSingleTest` ignores the flag (see [Test Matrix](#test-matrix)), both groups run the same code and any difference is timing noise — which is why `analyzeResults` discards the return value (`_ = mb.findParallelCrossover(bySize)`). Returns `0` on single-core systems and when no crossover is observed; the conservative default (`ParallelThreshold: 4096`) is owned by `analyzeResults()`.

### Confidence Scoring

The `ThresholdResults` struct includes a confidence score (0.0 to 1.0):

- Base confidence: 0.5 (conservative defaults assumed valid)
- 0.0 if no result at all was collected (timeout, or every test errored)
- +0.2 if an FFT crossover point was found
- Capped at 1.0

There is **no** parallel-crossover bonus: `runSingleTest` does not branch on the
`parallel` flag, so `findParallelCrossover`'s result is discarded (`_ = ...`).

A confidence of >= 0.5 is required for auto-calibration to accept micro-benchmark results.

## Calibration Profile

File: `internal/calibration/profile.go`

### Structure

```go
type CalibrationProfile struct {
    CPUModel                  string    `json:"cpu_model"`
    NumCPU                    int       `json:"num_cpu"`
    GOARCH                    string    `json:"goarch"`
    GOOS                      string    `json:"goos"`
    GoVersion                 string    `json:"go_version"`
    WordSize                  int       `json:"word_size"`
    CPUHeuristicKey           string    `json:"cpu_heuristic_key"`

    OptimalParallelThreshold  int       `json:"optimal_parallel_threshold"`
    OptimalFFTThreshold       int       `json:"optimal_fft_threshold"`
    OptimalStrassenThreshold  int       `json:"optimal_strassen_threshold"`

    CalibratedAt              time.Time `json:"calibrated_at"`
    CalibrationN              uint64    `json:"calibration_n"`
    CalibrationTime           string    `json:"calibration_time"`
    Confidence                float64   `json:"confidence"` // 0.0–1.0 calibration reliability
    ProfileVersion            int       `json:"profile_version"`
}
```

`NewProfile()` populates the hardware fields from `runtime`, sets `CPUHeuristicKey` from `config.CurrentHardwareHeuristicKey()` (the SIMD-class / arch tag that drives the default thresholds), and sets `ProfileVersion` to `CurrentProfileVersion` (currently **3**).

### Validation

`IsValid()` checks the following fields against the running system:

| Field | Comparison |
|-------|-----------|
| `ProfileVersion` | Must equal `CurrentProfileVersion` |
| `NumCPU` | Must equal `runtime.NumCPU()` |
| `GOARCH` | Must equal `runtime.GOARCH` |
| `WordSize` | Must equal system word size (32 or 64) |
| `CPUHeuristicKey` | Must equal `config.CurrentHardwareHeuristicKey()` (ex. `amd64-avx2`, `amd64-generic`) |

If any field differs, the profile is invalid and a fresh calibration is triggered. **v2**-format profiles (without `cpu_heuristic_key`) are no longer accepted after the version bump.

**Migration:** delete or rename `~/.fibcalc_calibration.json` when upgrading from an earlier binary, or re-run `--calibrate` / `--auto-calibrate` to regenerate a v3 profile.

`IsStale(maxAge time.Duration)` provides time-based invalidation. A profile older than `maxAge` is considered stale. This can be used to trigger periodic re-calibration.

### Persistence

File: `internal/calibration/profile.go` (save/load methods) and `internal/calibration/io.go` (output formatting).

- `SaveProfile(path)`: Serializes to JSON with `json.MarshalIndent` and writes with `0600` permissions. If `path` is empty, uses the default path.
- `loadProfile(path)`: Reads and deserializes. Returns an error if the file is missing or malformed.
- `LoadOrCreateProfile(path)`: Loads an existing valid profile, or returns `NewProfile(), false`. That fallback profile is **not empty** — `NewProfile()` populates nine fields from the running host (`CPUModel`, `NumCPU`, `GOARCH`, `GOOS`, `GoVersion`, `WordSize`, `CPUHeuristicKey`, `CalibratedAt`, `ProfileVersion` — `internal/calibration/profile.go:NewProfile`). Only the three `Optimal*Threshold` values and `Confidence` are left at zero.
- `GetDefaultProfilePath()`: Returns `~/.fibcalc_calibration.json` (falls back to the current directory if `$HOME` is unavailable).

Example profile on disk. `json.MarshalIndent` emits keys in struct-declaration order, so a
real file always has exactly this key sequence — `cpu_heuristic_key` sits with the hardware
block (before the thresholds), and `confidence` is always present:

```json
{
  "cpu_model": "amd64-12-cores",
  "num_cpu": 12,
  "goarch": "amd64",
  "goos": "linux",
  "go_version": "go1.26.0",
  "word_size": 64,
  "cpu_heuristic_key": "amd64-avx2",
  "optimal_parallel_threshold": 2048,
  "optimal_fft_threshold": 500000,
  "optimal_strassen_threshold": 256,
  "calibrated_at": "2025-03-15T10:30:00Z",
  "calibration_n": 10000000,
  "calibration_time": "45.2s",
  "confidence": 1,
  "profile_version": 3
}
```

`calibration_time` is populated only on the `--calibrate` path
(`persistCalibrationProfile`). On the `--auto-calibrate` path the persisted profile is rebuilt
by `saveCalibrationProfile`, which copies the thresholds, `CalibrationN` and `Confidence` but
not `CalibrationTime` — so an auto-calibrated file carries `"calibration_time": ""`.

## Adaptive Threshold Generation

File: `internal/calibration/adaptive.go`

The **benchmark-free estimates** (`EstimateOptimal*`, used when the thresholds are left at 0 and no valid profile loads) live in `internal/config/thresholds.go`. They read the core count and, on **amd64/386**, the **AVX2 / AVX-512** capabilities detected through `golang.org/x/sys/cpu` (`internal/config/hardware.go`). See also [PERFORMANCE.md](PERFORMANCE.md#hardware-heuristic-defaults).

### Parallel Threshold Candidates

`GenerateParallelThresholds()` produces a CPU-adaptive candidate list:

The baseline element is `-1` (the genuine sequential sentinel), not `0` — `0`
would be rewritten to the package default by `normalizeOptions()` and the
no-parallelism run would never be measured (FIB-02).

| Core Count | Candidates |
|-----------|------------|
| 1 | `[-1]` |
| 2-4 | `[-1, 512, 1024, 2048, 4096]` |
| 5-8 | `[-1, 256, 512, 1024, 2048, 4096, 8192]` |
| 9-16 | `[-1, 256, 512, 1024, 2048, 4096, 8192, 16384]` |
| 17+ | `[-1, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768]` |

`GenerateQuickParallelThresholds()` returns a reduced set for auto-calibration (Tier 3 fallback):

| Core Count | Candidates |
|-----------|------------|
| 1 | `[-1]` |
| 2-4 | `[-1, 2048, 4096]` |
| 5-8 | `[-1, 2048, 4096, 8192]` |
| 9+ | `[-1, 2048, 4096, 8192, 16384]` |

### FFT and Strassen Candidates

- `GenerateFFTThresholds()` (exported, `adaptive.go:GenerateFFTThresholds`): `[-1]` plus 200 000 → 1 000 000 in steps of 50 000 — 18 candidates. **This is the list the full runner actually sweeps** (`runner.go:findBestFFTThreshold`).
- `generateQuickFFTThresholds()` (unexported, `adaptive.go:generateQuickFFTThresholds`): `[-1, 750000, 1000000, 1500000]` — used by the quick micro-benchmark tier, not by the runner.
- `GenerateQuickStrassenThresholds()` (`adaptive.go:GenerateQuickStrassenThresholds`): `[192, 256, 384, 512]` — used by `runner.go:findBestStrassenThreshold`.

### Heuristic Estimation (No Benchmarks)

When benchmarks cannot run (e.g., timeout or missing calculator), heuristic functions provide reasonable defaults:

**`EstimateOptimalParallelThreshold()`**:

| Core Count | Estimated Threshold |
|-----------|-------------------|
| 1 | 0 |
| 2 | 8192 |
| 3-4 | 4096 |
| 5-8 | 2048 |
| 9-16 | 1024 |
| 17+ | 512 |

> A returned `0` does **not** disable parallelism by itself: `normalizeOptions`
> (`internal/fibonacci/options.go`) rewrites a zero `ParallelThreshold` to
> `DefaultParallelThreshold` (4096). What actually disables parallelism on a
> single-core host is the `runtime.GOMAXPROCS(0) > 1` gate in
> `internal/fibonacci/fastdoubling.go`.

> **SIMD adjustment** — on hosts with 8+ cores and a table value above 512,
> `estimateParallelThresholdForHeuristic()` (`internal/config/thresholds.go`)
> lowers the estimate for higher SIMD throughput: `max(512, base - 512)` with
> AVX-512, `max(512, base - 256)` with AVX2.

**`EstimateOptimalFFTThreshold()`**: on 64-bit systems, 500,000 bits (generic), 480,000 (AVX2) or 460,000 (AVX-512); 250,000 bits on 32-bit systems.

**`EstimateOptimalStrassenThreshold()`**: on systems with 4+ cores, 256 bits (generic), 240 (AVX2) or 224 (AVX-512); 3,072 bits otherwise.

## Calibration Runner

File: `internal/calibration/runner.go`

The `calibrationRunner` struct encapsulates the trial execution logic used by auto-calibration Tier 3:

```go
type calibrationRunner struct {
    ctx      context.Context
    perTrial time.Duration
}
```

`newCalibrationRunner()` derives a per-trial timeout from the overall timeout (`timeout / 6`, minimum 2 seconds). Each trial uses `context.WithTimeout` to prevent any single test from blocking.

Three search methods iterate over their respective candidate lists:

| Method | Calculator | Options Varied | Candidates Source |
|--------|-----------|----------------|-------------------|
| `findBestParallelThreshold()` | "fast" | `ParallelThreshold` | `GenerateQuickParallelThresholds()` |
| `findBestFFTThreshold()` | "fast" | `FFTThreshold` (with best parallel) | `GenerateFFTThresholds()` |
| `findBestStrassenThreshold()` | "matrix" | `StrassenThreshold` (with best parallel) | `GenerateQuickStrassenThresholds()` |

Each method returns the best threshold and its duration. If all trials fail (timeout or error), the default threshold is preserved.

## Package Structure

| File | Responsibility |
|------|---------------|
| `calibration.go` | Entry points: `RunCalibration()`, `AutoCalibrate()`, `AutoCalibrateWithProfile()`, `LoadCachedCalibration()` |
| `adaptive.go` | CPU-adaptive candidate-list generation only (`Generate*Thresholds`). It holds no estimation code: the `EstimateOptimal*` heuristics live in `internal/config/thresholds.go` and the former pass-through delegates were removed (audit 2026-06) |
| `microbench.go` | Quick micro-benchmarking engine (`QuickCalibrate()`, `MicroBenchmark`) |
| `profile.go` | `CalibrationProfile` data structure, validation, serialization |
| `io.go` | Result formatting and output (`printCalibrationResults()`, `printCalibrationOutput()`) |
| `runner.go` | `calibrationRunner` with `findBest*Threshold()` methods |
| `strategy.go` | `CalibrationStrategy` interface (Strategy pattern, see the Architecture note under Auto-Calibration) |
| `strategy_fast.go` | `FastStrategy` -- micro-benchmark tier |
| `strategy_complete.go` | `CompleteStrategy` -- full calibration runner tier |
| `doc.go` | Package documentation |

## Tuning Recommendations

| Hardware | ParallelThreshold | FFTThreshold | Notes |
|----------|-------------------|--------------|-------|
| Laptop (4 cores) | 2048-4096 | 500,000 | Conservative to avoid thermal throttling |
| Desktop (8 cores) | 1024-2048 | 500,000 | Good parallelism gains |
| Server (16+ cores) | 256-1024 | 250,000-500,000 | Maximum parallelism beneficial |
| Low memory (< 8 GB) | 4096+ | 1,000,000 | Higher FFT threshold reduces memory pressure |
| 32-bit system | 4096 | 250,000 | Smaller word size shifts crossover points |

For most users, running `fibcalc --auto-calibrate` once is sufficient. The saved profile will be reused on subsequent runs until the hardware configuration changes (e.g., different core count after a VM resize).

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FIBCALC_CALIBRATE` | Enable full calibration | `false` |
| `FIBCALC_AUTO_CALIBRATE` | Enable auto-calibration | `false` |
| `FIBCALC_CALIBRATION_PROFILE` | Path to calibration profile file | `~/.fibcalc_calibration.json` |
| `FIBCALC_PROFILE_MAX_AGE` | Freshness window; a profile older than this is `IsStale` and triggers re-calibration via `CompleteStrategy` | `168h` (7 d) |

These environment variables follow the `FIBCALC_*` convention and have lower priority than their corresponding CLI flags. See `internal/config/env.go` for the full list — except `FIBCALC_PROFILE_MAX_AGE`, which is defined and consumed by `internal/calibration/calibration.go` (`ProfileMaxAgeEnv` / `profileMaxAgeFromEnv`).

## Cross-References

- [PERFORMANCE.md](PERFORMANCE.md) -- Threshold impact on the 2-tier multiplication system
- [BUILD.md](BUILD.md) -- `make run-calibrate` target
- [Architecture](architecture/README.md) -- Calibration package placement (Business Logic layer, wired in at the app layer via `internal/app`)
- [algorithms/FFT.md](algorithms/FFT.md) -- FFT threshold context and algorithm details
