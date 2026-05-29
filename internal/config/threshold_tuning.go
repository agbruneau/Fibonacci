package config

import (
	"time"

	"github.com/agbruneau/FibGo/internal/fibonacci/memory"
)

// ThresholdTuningProfile centralises the magic numbers that govern the
// dynamic-threshold and micro-benchmark subsystems. Concentrating them here
// gives a single audit point when tuning the heuristics; previously the
// constants were scattered across the threshold/, memory/ and calibration/
// packages.
//
// Field origins are documented inline. Values must be kept in sync with the
// constants they replaced — see audit R4.2.
type ThresholdTuningProfile struct {
	// FFTSpeedupThreshold is the minimum measured speedup ratio
	// (baseline / FFT) for the dynamic-threshold manager to consider FFT
	// beneficial and lower its activation threshold. Values below the
	// reciprocal raise it instead.
	//
	// Source: internal/fibonacci/threshold/manager.go (pre-R4.2 const).
	// Rationale: 1.2 = 20 % minimum win to amortise FFT setup overhead.
	FFTSpeedupThreshold float64

	// ParallelSpeedupThreshold mirrors FFTSpeedupThreshold for the
	// goroutine-parallel multiplication path. The threshold is lower than
	// FFT because parallel dispatch is cheaper than the FFT transform.
	//
	// Source: internal/fibonacci/threshold/manager.go (pre-R4.2 const).
	// Rationale: 1.1 = 10 % minimum win — enough to cover sync.WaitGroup
	// and goroutine scheduling overhead on commodity hardware.
	ParallelSpeedupThreshold float64

	// HysteresisMargin is the minimum |Δthreshold| / |threshold|
	// required before the manager commits a new value. It damps
	// oscillation between modes when measurements straddle the speedup
	// cut-off.
	//
	// Source: internal/fibonacci/threshold/manager.go (pre-R4.2 const).
	// Rationale: 0.15 chosen empirically — large enough to suppress
	// jitter, small enough to track real workload shifts within a few
	// adjustment intervals.
	HysteresisMargin float64

	// MinFFTThreshold is the floor (in bits) below which the dynamic
	// manager refuses to drop the FFT activation threshold, even when
	// FFT looks faster on the most recent samples. Below this size the
	// schoolbook / Karatsuba path of math/big always wins.
	//
	// Source: internal/fibonacci/threshold/manager.go,
	// AnalysisParams.MinThreshold for the FFT analysis (pre-R4.2 literal).
	MinFFTThreshold int

	// MinParallelThreshold is the analogous floor for the parallel
	// multiplication path. Below ~16 machine words (≈1024 bits) the
	// dispatch overhead dwarfs any work the worker can do.
	//
	// Source: internal/fibonacci/threshold/manager.go,
	// AnalysisParams.MinThreshold for the parallel analysis
	// (pre-R4.2 literal).
	MinParallelThreshold int

	// MemoryLimitMultiplier is the factor applied to the runtime's
	// reported Sys footprint to derive a soft GOMEMLIMIT while GC is
	// disabled. Acts as an OOM safety net for very large F(N).
	//
	// Source: internal/fibonacci/memory/gc_control.go (pre-R4.2 literal).
	// Rationale: 3.0 leaves headroom for the doubling loop's working set
	// (a + b + temp + arena) without leaking to a hard limit on hosts
	// where Sys already includes a generous Go heap.
	//
	// The canonical value lives in
	// memory.DefaultMemoryLimitMultiplier so the gc_control package can
	// consume it without taking a dependency on internal/config (config
	// already depends on memory via validator.go, and the reverse import
	// would be a cycle).
	MemoryLimitMultiplier float64

	// MicroBenchTimeout caps the wall-clock duration of the entire
	// quick-calibration micro-benchmark suite. The 150 ms budget is
	// chosen to keep `fibcalc --quick-calibrate` responsive while still
	// allowing the four (size × algorithm) configurations to each
	// collect a few iterations on commodity hardware.
	//
	// Source: internal/calibration/microbench.go (pre-R4.2 const).
	MicroBenchTimeout time.Duration
}

// DefaultThresholdTuning is the production tuning profile. Its values
// reproduce the constants that lived in internal/fibonacci/threshold/,
// internal/fibonacci/memory/ and internal/calibration/ before audit R4.2.
// Behaviour is therefore byte-identical to the pre-centralisation code.
var DefaultThresholdTuning = ThresholdTuningProfile{
	FFTSpeedupThreshold:      1.2,
	ParallelSpeedupThreshold: 1.1,
	HysteresisMargin:         0.15,
	MinFFTThreshold:          100_000,
	MinParallelThreshold:     1024,
	MemoryLimitMultiplier:    memory.DefaultMemoryLimitMultiplier,
	MicroBenchTimeout:        150 * time.Millisecond,
}
