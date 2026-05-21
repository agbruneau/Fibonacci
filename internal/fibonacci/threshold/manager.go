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
// All mutable state is held in atomics; the legacy mu RWMutex is preserved
// only as a write-side barrier for Reset (multi-field consistency). Readers
// access fields via atomic loads and do not take a lock.
type DynamicThresholdManager struct {
	mu     sync.Mutex // serializes Reset's multi-field update; readers use atomics
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

// NewDynamicThresholdManager creates a new manager with the given initial thresholds.
func NewDynamicThresholdManager(fftThreshold, parallelThreshold int) *DynamicThresholdManager {
	m := &DynamicThresholdManager{
		logger:                    zerolog.Nop(),
		originalFFTThreshold:      fftThreshold,
		originalParallelThreshold: parallelThreshold,
		adjustmentInterval:        DynamicAdjustmentInterval,
	}
	m.currentFFTThreshold.Store(int64(fftThreshold))
	m.currentParallelThreshold.Store(int64(parallelThreshold))
	return m
}

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

// SetLogger configures the logger for threshold adjustment events.
func (m *DynamicThresholdManager) SetLogger(l zerolog.Logger) {
	m.logger = l
}

// ─────────────────────────────────────────────────────────────────────────────
// Metric Recording
// ─────────────────────────────────────────────────────────────────────────────

// RecordIteration records timing data for a completed iteration.
// This should be called after each doubling step in the algorithm.
//
// All mutable state is atomic. The buffer.Record method is internally
// safe for the single-writer/many-reader access pattern documented in
// MetricsBuffer.
func (m *DynamicThresholdManager) RecordIteration(bitLen int, duration time.Duration, usedFFT, usedParallel bool) {
	m.buffer.Record(bitLen, duration, usedFFT, usedParallel)
	m.iterationCount.Add(1)
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Access
// ─────────────────────────────────────────────────────────────────────────────

// GetThresholds returns the current FFT and parallel thresholds.
func (m *DynamicThresholdManager) GetThresholds() (fft, parallel int) {
	return int(m.currentFFTThreshold.Load()), int(m.currentParallelThreshold.Load())
}

// GetFFTThreshold returns the current FFT threshold.
func (m *DynamicThresholdManager) GetFFTThreshold() int {
	return int(m.currentFFTThreshold.Load())
}

// GetParallelThreshold returns the current parallel threshold.
func (m *DynamicThresholdManager) GetParallelThreshold() int {
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

	if m.buffer.Count() < MinMetricsForAdjustment {
		return currentFFT, currentParallel, false
	}

	newFFT = m.analyzeFFTThreshold()
	newParallel = m.analyzeParallelThreshold()

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

// analyzeFFTThreshold delegates to the analyzer with FFT-specific parameters.
func (m *DynamicThresholdManager) analyzeFFTThreshold() int {
	return m.analyzer.Analyze(m.buffer.RecentMetrics(), AnalysisParams{
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

// analyzeParallelThreshold delegates to the analyzer with parallel-specific parameters.
func (m *DynamicThresholdManager) analyzeParallelThreshold() int {
	return m.analyzer.Analyze(m.buffer.RecentMetrics(), AnalysisParams{
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

// avgTimePerBit is a thin wrapper preserved for backward-compatible test
// access; new code should use ThresholdAnalyzer.AvgTimePerBit directly.
func (m *DynamicThresholdManager) avgTimePerBit(metrics []IterationMetric) float64 {
	return m.analyzer.AvgTimePerBit(metrics)
}

// significantChange is a thin wrapper preserved for backward-compatible test
// access; new code should use ThresholdAnalyzer.SignificantChange directly.
func (m *DynamicThresholdManager) significantChange(oldVal, newVal int) bool {
	return m.analyzer.SignificantChange(oldVal, newVal)
}

// ─────────────────────────────────────────────────────────────────────────────
// Statistics and Reporting
// ─────────────────────────────────────────────────────────────────────────────

// GetStats returns current statistics about the manager.
func (m *DynamicThresholdManager) GetStats() ThresholdStats {
	return ThresholdStats{
		CurrentFFT:          int(m.currentFFTThreshold.Load()),
		CurrentParallel:     int(m.currentParallelThreshold.Load()),
		OriginalFFT:         m.originalFFTThreshold,
		OriginalParallel:    m.originalParallelThreshold,
		MetricsCollected:    m.buffer.Count(),
		IterationsProcessed: int(m.iterationCount.Load()),
	}
}

// Reset clears all collected metrics and restores original thresholds.
func (m *DynamicThresholdManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentFFTThreshold.Store(int64(m.originalFFTThreshold))
	m.currentParallelThreshold.Store(int64(m.originalParallelThreshold))
	m.buffer.Reset()
	m.iterationCount.Store(0)
}
