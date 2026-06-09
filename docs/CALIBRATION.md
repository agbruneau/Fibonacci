# Calibration System

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 747 nodes / 8 layers / 13-step tour)

## Overview

The calibration system (`internal/calibration/`) determines optimal performance thresholds for the current hardware. Rather than relying on hard-coded constants, it benchmarks the system at runtime and selects the threshold values that yield the fastest execution for the active CPU, architecture, and core count.

It complements (does not replace) the **Dynamic Threshold Manager**
(`internal/fibonacci/threshold/`), which adjusts the FFT/parallel
thresholds *in-flight* during a calculation based on observed
per-iteration metrics. The two layers are kept separate because they
solve different problems : calibration produces a startup snapshot,
the manager reacts to per-run variance. Their relative value is
benchmarked in [`docs/audits/bench-dtm-{on,off}.txt`](audits/) and
analysed in [`docs/adr/0001-dtm-decision.md`](adr/0001-dtm-decision.md).

Three operational modes are supported:

| Mode | Flag | Latency | Description |
|------|------|---------|-------------|
| Full calibration | `--calibrate` | Seconds to minutes | Exhaustive threshold sweep with real Fibonacci calculations |
| Auto-calibration | `--auto-calibrate` | Instant to seconds | 3-tier fallback: cached profile, micro-benchmarks, full runner |
| Cached profile | `--calibration-profile` | Instant | Loads a previously saved JSON profile |

## Quick Start

```bash
# Run full calibration (saves profile to ~/.fibcalc_calibration.json)
fibcalc --calibrate

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

When operand bit sizes exceed a threshold, the corresponding optimization is activated. Setting a threshold to `0` disables parallelism (sequential execution); higher values delay activation to larger operand sizes.

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
6. Results are printed as a formatted table and the profile is saved to `~/.fibcalc_calibration.json`.

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

`LoadOrCreateProfile()` attempts to load a saved profile from disk. If the profile exists and `IsValid()` returns true (matching CPU count, architecture, and word size), the cached thresholds are applied immediately. No benchmarks are executed.

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
const (
    MicroBenchIterations     = 3
    MicroBenchPerTestTimeout = 30 * time.Millisecond // per individual test
)

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

For each word size, four configurations are tested:

1. Standard math/big sequential
2. Standard math/big parallel
3. FFT sequential
4. FFT parallel

Tests run in parallel with a semaphore limiting concurrency to `runtime.NumCPU()`. Each test generates deterministic `big.Int` operands via `generateTestNumber()`, performs a warm-up multiplication, then averages 3 timed iterations.

### Analysis

After all tests complete, the engine analyzes results:

- `findFFTCrossover()`: Identifies the smallest bit size where FFT multiplication is faster than standard `math/big`. Applies a 10% margin (multiplies the crossover by 9/10) to ensure FFT is clearly beneficial. Falls back to 1,000,000 bits if no crossover is found.

- `findParallelCrossover()`: Identifies the smallest bit size where parallel multiplication is at least 10% faster than sequential. Returns 0 on single-core systems. Falls back to 4,096 bits if no crossover is found.

### Confidence Scoring

The `ThresholdResults` struct includes a confidence score (0.0 to 1.0):

- Base confidence: 0.5 (conservative defaults assumed valid)
- +0.2 if an FFT crossover point was found
- +0.2 if a parallel crossover point was found
- Capped at 1.0

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

`NewProfile()` populates hardware fields from `runtime`, définit `CPUHeuristicKey` via `config.CurrentHardwareHeuristicKey()` (classe SIMD / arch pour les seuils par défaut), et fixe `ProfileVersion` à `CurrentProfileVersion` (actuellement **3**).

### Validation

`IsValid()` vérifie les conditions suivantes par rapport au système courant :

| Field | Comparison |
|-------|-----------|
| `ProfileVersion` | Must equal `CurrentProfileVersion` |
| `NumCPU` | Must equal `runtime.NumCPU()` |
| `GOARCH` | Must equal `runtime.GOARCH` |
| `WordSize` | Must equal system word size (32 or 64) |
| `CPUHeuristicKey` | Must equal `config.CurrentHardwareHeuristicKey()` (ex. `amd64-avx2`, `amd64-generic`) |

Si un champ diffère, le profil est invalide et une nouvelle calibration est déclenchée. Les profils au format **v2** (sans `cpu_heuristic_key`) ne sont plus acceptés après montée de version.

**Migration :** supprimez ou renommez `~/.fibcalc_calibration.json` si vous passez d’une version antérieure du binaire, ou relancez `--calibrate` / `--auto-calibrate` pour régénérer un profil v3.

`IsStale(maxAge time.Duration)` provides time-based invalidation. A profile older than `maxAge` is considered stale. This can be used to trigger periodic re-calibration.

### Persistence

File: `internal/calibration/profile.go` (save/load methods) and `internal/calibration/io.go` (output formatting).

- `SaveProfile(path)`: Serializes to JSON with `json.MarshalIndent` and writes with `0600` permissions. If `path` is empty, uses the default path.
- `loadProfile(path)`: Reads and deserializes. Returns an error if the file is missing or malformed.
- `LoadOrCreateProfile(path)`: Loads an existing valid profile or returns a new empty profile with `false`.
- `GetDefaultProfilePath()`: Returns `~/.fibcalc_calibration.json` (falls back to the current directory if `$HOME` is unavailable).

Example profile on disk:

```json
{
  "cpu_model": "amd64-12-cores",
  "num_cpu": 12,
  "goarch": "amd64",
  "goos": "linux",
  "go_version": "go1.26.0",
  "word_size": 64,
  "optimal_parallel_threshold": 2048,
  "optimal_fft_threshold": 500000,
  "optimal_strassen_threshold": 256,
  "calibrated_at": "2025-03-15T10:30:00Z",
  "calibration_n": 10000000,
  "calibration_time": "45.2s",
  "cpu_heuristic_key": "amd64-avx2",
  "profile_version": 3
}
```

## Adaptive Threshold Generation

File: `internal/calibration/adaptive.go`

Les **estimations sans benchmark** (`EstimateOptimal*`, utilisées quand les seuils restent à 0 et qu’aucun profil valide n’est chargé) sont définies dans `internal/config/thresholds.go` et tiennent compte du nombre de cœurs et, sur **amd64/386**, des capacités **AVX2 / AVX-512** détectées via `golang.org/x/sys/cpu` (`internal/config/hardware.go`). Voir aussi [PERFORMANCE.md](PERFORMANCE.md#hardware-heuristic-defaults).

### Parallel Threshold Candidates

`GenerateParallelThresholds()` produces a CPU-adaptive candidate list:

| Core Count | Candidates |
|-----------|------------|
| 1 | `[0]` |
| 2-4 | `[0, 512, 1024, 2048, 4096]` |
| 5-8 | `[0, 256, 512, 1024, 2048, 4096, 8192]` |
| 9-16 | `[0, 256, 512, 1024, 2048, 4096, 8192, 16384]` |
| 17+ | `[0, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768]` |

`GenerateQuickParallelThresholds()` returns a reduced set for auto-calibration (Tier 3 fallback):

| Core Count | Candidates |
|-----------|------------|
| 1 | `[0]` |
| 2-4 | `[0, 2048, 4096]` |
| 5-8 | `[0, 2048, 4096, 8192]` |
| 9+ | `[0, 2048, 4096, 8192, 16384]` |

### FFT and Strassen Candidates

- `GenerateQuickFFTThresholds()`: `[0, 750000, 1000000, 1500000]`
- `GenerateQuickStrassenThresholds()`: `[192, 256, 384, 512]`

### Heuristic Estimation (No Benchmarks)

When benchmarks cannot run (e.g., timeout or missing calculator), heuristic functions provide reasonable defaults:

**`EstimateOptimalParallelThreshold()`**:

| Core Count | Estimated Threshold |
|-----------|-------------------|
| 1 | 0 (disabled) |
| 2 | 8192 |
| 3-4 | 4096 |
| 5-8 | 2048 |
| 9-16 | 1024 |
| 17+ | 512 |

**`EstimateOptimalFFTThreshold()`**: 500,000 bits on 64-bit systems, 250,000 bits on 32-bit systems.

**`EstimateOptimalStrassenThreshold()`**: 256 bits on systems with 4+ cores, 3,072 bits otherwise.

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
| `adaptive.go` | CPU-adaptive threshold generation and heuristic estimation |
| `microbench.go` | Quick micro-benchmarking engine (`QuickCalibrate()`, `MicroBenchmark`) |
| `profile.go` | `CalibrationProfile` data structure, validation, serialization |
| `io.go` | Result formatting and output (`printCalibrationResults()`, `printCalibrationOutput()`) |
| `runner.go` | `calibrationRunner` with `findBest*Threshold()` methods |
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

These environment variables follow the `FIBCALC_*` convention and have lower priority than their corresponding CLI flags. See `internal/config/env.go` for the full list.

## Cross-References

- [PERFORMANCE.md](PERFORMANCE.md) -- Threshold impact on the 3-tier multiplication system
- [BUILD.md](BUILD.md) -- `make run-calibrate` target
- [Architecture](architecture/README.md) -- Calibration package placement in the orchestration layer
- [algorithms/FFT.md](algorithms/FFT.md) -- FFT threshold context and algorithm details
