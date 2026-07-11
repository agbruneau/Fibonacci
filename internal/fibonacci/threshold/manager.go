// This file implements dynamic threshold adjustment during calculation.

package threshold

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// ─────────────────────────────────────────────────────────────────────────────
// Dynamic Threshold Configuration
// ─────────────────────────────────────────────────────────────────────────────

const (
	// DynamicAdjustmentInterval is the number of iterations between threshold checks.
	DynamicAdjustmentInterval = 5

	// MinMetricsForAdjustment is the minimum number of metrics needed before adjusting.
	MinMetricsForAdjustment = 3

	// MaxMetricsHistory is the maximum number of metrics to keep for analysis.
	MaxMetricsHistory = 20
)

// Tuning knobs owned by the threshold package. Mirrors
// internal/config.DefaultThresholdTuning, but kept here so this leaf
// package does not import config (which would close a cycle via
// config → fibonacci/memory). The config layer reads these defaults
// and may override them via SetTuning before any manager is constructed.
// INVARIANT (A2-04): these package-level tuning knobs are NOT synchronized.
// They are written ONLY by SetTuning, itself called once at wiring time (by
// internal/app) BEFORE any DynamicThresholdManager is constructed and before
// any calculation reads them — single-writer-before-use. Calling SetTuning
// concurrently with an active calculation would be a data race. The manager's
// per-instance fields use atomic.Int64/atomic.Pointer; these package defaults
// stay in clear by this deliberate protocol (no atomic migration this pass).
var (
	// FFTSpeedupThreshold is the minimum speedup ratio (baseline / FFT)
	// at which the dynamic-threshold manager will lower the FFT
	// activation threshold.
	FFTSpeedupThreshold = 1.2

	// ParallelSpeedupThreshold is the analogous ratio for the parallel
	// multiplication path.
	ParallelSpeedupThreshold = 1.1

	// HysteresisMargin is the minimum relative change required before a
	// new threshold is committed; damps oscillation.
	HysteresisMargin = 0.15
)

// Floor values used by the analyzer to bound downward adjustments.
// Mirrored from internal/config.DefaultThresholdTuning; see the SetTuning
// note for override semantics.
var (
	minFFTThresholdFloor      = 100_000
	minParallelThresholdFloor = 1024
)

// Tuning is a value object carrying tuning knobs that the config layer
// can inject without creating an upward import from this package to
// internal/config.
type Tuning struct {
	FFTSpeedupThreshold      float64
	ParallelSpeedupThreshold float64
	HysteresisMargin         float64
	MinFFTThreshold          int
	MinParallelThreshold     int
}

// SetTuning installs new defaults for the package-level tuning knobs.
// Intended to be called once at startup from the wiring layer (e.g. by
// internal/app) so production code can stay on
// internal/config.DefaultThresholdTuning as its single source of truth
// while the threshold package itself stays free of upward imports.
func SetTuning(t Tuning) {
	if t.FFTSpeedupThreshold > 0 {
		FFTSpeedupThreshold = t.FFTSpeedupThreshold
	}
	if t.ParallelSpeedupThreshold > 0 {
		ParallelSpeedupThreshold = t.ParallelSpeedupThreshold
	}
	if t.HysteresisMargin > 0 {
		HysteresisMargin = t.HysteresisMargin
	}
	if t.MinFFTThreshold > 0 {
		minFFTThresholdFloor = t.MinFFTThreshold
	}
	if t.MinParallelThreshold > 0 {
		minParallelThresholdFloor = t.MinParallelThreshold
	}
}

// DynamicThresholdManager adjusts FFT and parallel thresholds during calculation
// based on observed performance metrics.
//
// The manager is the orchestrator: it owns the thresholds and the lifecycle
// state, delegates sample storage to MetricsBuffer, and delegates the
// analytical work (speedup, hysteresis, multiplicative adjustment) to
// ThresholdAnalyzer.
//
// Threshold and counter state is held in atomics; mu serializes Reset's
// multi-field update AND every access to the MetricsBuffer, which is
// deliberately not goroutine-safe (its doc delegates synchronization to
// this manager). Record/Count/RecentMetrics all go through mu — the buffer
// write path races with concurrent getStats readers otherwise (data race
// found by go test -race on TestConcurrentAccess, 2026-06-10).
type DynamicThresholdManager struct {
	mu     sync.Mutex // serializes Reset AND all MetricsBuffer access; other fields use atomics
	logger zerolog.Logger

	// Current thresholds (can be adjusted during calculation).
	currentFFTThreshold      atomic.Int64
	currentParallelThreshold atomic.Int64

	// Original thresholds (immutable after construction).
	originalFFTThreshold      int
	originalParallelThreshold int

	// Metrics storage and analysis.
	buffer   MetricsBuffer
	analyzer ThresholdAnalyzer

	// Adjustment state.
	iterationCount     atomic.Int64
	adjustmentInterval int // immutable after construction
	lastAdjustment     atomic.Pointer[time.Time]
}

// ─────────────────────────────────────────────────────────────────────────────
// Constructor and Configuration
// ─────────────────────────────────────────────────────────────────────────────

// NewDynamicThresholdManagerFromConfig creates a manager from configuration.
func NewDynamicThresholdManagerFromConfig(cfg DynamicThresholdConfig) *DynamicThresholdManager {
	if !cfg.Enabled {
		return nil
	}

	interval := cfg.AdjustmentInterval
	if interval <= 0 {
		interval = DynamicAdjustmentInterval
	}

	m := &DynamicThresholdManager{
		logger:                    zerolog.Nop(),
		originalFFTThreshold:      cfg.InitialFFTThreshold,
		originalParallelThreshold: cfg.InitialParallelThreshold,
		adjustmentInterval:        interval,
	}
	m.currentFFTThreshold.Store(int64(cfg.InitialFFTThreshold))
	m.currentParallelThreshold.Store(int64(cfg.InitialParallelThreshold))
	return m
}

// setLogger configures the logger for threshold adjustment events.
func (m *DynamicThresholdManager) setLogger(l zerolog.Logger) {
	m.logger = l
}

// ─────────────────────────────────────────────────────────────────────────────
// Metric Recording
// ─────────────────────────────────────────────────────────────────────────────

// RecordIteration records timing data for a completed iteration.
// This should be called after each doubling step in the algorithm.
//
// The buffer write must hold mu: MetricsBuffer is not goroutine-safe and a
// concurrent getStats/ShouldAdjust reader would race with this writer (the
// uncontended lock costs ~tens of ns per doubling step — invisible next to
// a multiplication step).
func (m *DynamicThresholdManager) RecordIteration(bitLen int, duration time.Duration, usedFFT, usedParallel bool) {
	m.mu.Lock()
	m.buffer.Record(bitLen, duration, usedFFT, usedParallel)
	m.mu.Unlock()
	m.iterationCount.Add(1)
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Access
// ─────────────────────────────────────────────────────────────────────────────

// getThresholds returns the current FFT and parallel thresholds.
func (m *DynamicThresholdManager) getThresholds() (fft, parallel int) {
	return int(m.currentFFTThreshold.Load()), int(m.currentParallelThreshold.Load())
}

// getFFTThreshold returns the current FFT threshold.
func (m *DynamicThresholdManager) getFFTThreshold() int {
	return int(m.currentFFTThreshold.Load())
}

// getParallelThreshold returns the current parallel threshold.
func (m *DynamicThresholdManager) getParallelThreshold() int {
	return int(m.currentParallelThreshold.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// Adjustment Logic
// ─────────────────────────────────────────────────────────────────────────────

// ShouldAdjust checks if thresholds should be adjusted based on collected metrics.
// Returns the new thresholds and whether an adjustment was made.
//
// Concurrency: all mutable state is held in atomics. ShouldAdjust may run
// concurrently with reader-side accessors; readers observe either the
// pre-adjustment or post-adjustment value (never a torn write). Concurrent
// callers of ShouldAdjust itself remain unsupported by design — the
// adjustment is intended to be driven by the single doubling-loop
// goroutine that owns the manager.
func (m *DynamicThresholdManager) ShouldAdjust() (newFFT, newParallel int, adjusted bool) {
	currentFFT := int(m.currentFFTThreshold.Load())
	currentParallel := int(m.currentParallelThreshold.Load())
	iterationCount := m.iterationCount.Load()

	if iterationCount%int64(m.adjustmentInterval) != 0 {
		return currentFFT, currentParallel, false
	}

	m.mu.Lock()
	count := m.buffer.Count()
	m.mu.Unlock()
	if count < MinMetricsForAdjustment {
		return currentFFT, currentParallel, false
	}

	// One snapshot shared by both analyzers (previously each call snapshotted
	// independently — a redundant lock + buffer copy per adjustment).
	metrics := m.snapshotMetrics()
	newFFT = m.analyzeFFTThresholdFrom(metrics)
	newParallel = m.analyzeParallelThresholdFrom(metrics)

	fftChanged := m.analyzer.SignificantChange(currentFFT, newFFT)
	parallelChanged := m.analyzer.SignificantChange(currentParallel, newParallel)

	if !fftChanged && !parallelChanged {
		return currentFFT, currentParallel, false
	}

	oldFFT := currentFFT
	oldParallel := currentParallel
	if fftChanged {
		m.currentFFTThreshold.Store(int64(newFFT))
		currentFFT = newFFT
	}
	if parallelChanged {
		m.currentParallelThreshold.Store(int64(newParallel))
		currentParallel = newParallel
	}
	now := time.Now()
	m.lastAdjustment.Store(&now)
	m.logger.Debug().
		Int64("iteration", iterationCount).
		Bool("fft_changed", fftChanged).
		Int("fft_old", oldFFT).
		Int("fft_new", currentFFT).
		Bool("parallel_changed", parallelChanged).
		Int("parallel_old", oldParallel).
		Int("parallel_new", currentParallel).
		Msg("thresholds adjusted")
	return currentFFT, currentParallel, true
}

// snapshotMetrics copies the buffer's retained samples under mu. The copy
// (RecentMetrics already allocates one) lets the analyzers run lock-free.
func (m *DynamicThresholdManager) snapshotMetrics() []IterationMetric {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buffer.RecentMetrics()
}

// analyzeFFTThresholdFrom analyzes the FFT threshold from a caller-provided
// metrics snapshot, applying FFT-specific parameters.
func (m *DynamicThresholdManager) analyzeFFTThresholdFrom(metrics []IterationMetric) int {
	return m.analyzer.Analyze(metrics, AnalysisParams{
		Predicate:         func(metric IterationMetric) bool { return metric.UsedFFT },
		SpeedupThreshold:  FFTSpeedupThreshold,
		LowerNumerator:    9,
		RaiseNumerator:    11,
		MinThreshold:      minFFTThresholdFloor,
		MaxCapMultiplier:  2,
		CurrentThreshold:  int(m.currentFFTThreshold.Load()),
		OriginalThreshold: m.originalFFTThreshold,
	})
}

// analyzeParallelThresholdFrom analyzes the parallel threshold from a
// caller-provided metrics snapshot, applying parallel-specific parameters.
func (m *DynamicThresholdManager) analyzeParallelThresholdFrom(metrics []IterationMetric) int {
	return m.analyzer.Analyze(metrics, AnalysisParams{
		Predicate:         func(metric IterationMetric) bool { return metric.UsedParallel },
		SpeedupThreshold:  ParallelSpeedupThreshold,
		LowerNumerator:    8,
		RaiseNumerator:    12,
		MinThreshold:      minParallelThresholdFloor,
		MaxCapMultiplier:  4,
		CurrentThreshold:  int(m.currentParallelThreshold.Load()),
		OriginalThreshold: m.originalParallelThreshold,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Statistics and Reporting
// ─────────────────────────────────────────────────────────────────────────────

// getStats returns current statistics about the manager.
func (m *DynamicThresholdManager) getStats() thresholdStats {
	m.mu.Lock()
	collected := m.buffer.Count()
	m.mu.Unlock()
	return thresholdStats{
		CurrentFFT:          int(m.currentFFTThreshold.Load()),
		CurrentParallel:     int(m.currentParallelThreshold.Load()),
		OriginalFFT:         m.originalFFTThreshold,
		OriginalParallel:    m.originalParallelThreshold,
		MetricsCollected:    collected,
		IterationsProcessed: int(m.iterationCount.Load()),
	}
}

// reset clears all collected metrics and restores original thresholds.
func (m *DynamicThresholdManager) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentFFTThreshold.Store(int64(m.originalFFTThreshold))
	m.currentParallelThreshold.Store(int64(m.originalParallelThreshold))
	m.buffer.Reset()
	m.iterationCount.Store(0)
}
