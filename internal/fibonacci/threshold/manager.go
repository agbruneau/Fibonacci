// This file implements dynamic threshold adjustment during calculation.

package threshold

import (
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/agbru/fibcalc/internal/config"
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

// Tuning knobs sourced from config.DefaultThresholdTuning. These were
// constants prior to audit R4.2; they are now declared as vars so the
// canonical value lives in one place (internal/config/threshold_tuning.go)
// without breaking the existing `threshold.FFTSpeedupThreshold` etc.
// callers in tests and docs.
var (
	// FFTSpeedupThreshold is the minimum speedup ratio (baseline / FFT)
	// at which the dynamic-threshold manager will lower the FFT
	// activation threshold. See config.ThresholdTuningProfile for the
	// rationale behind the default value.
	FFTSpeedupThreshold = config.DefaultThresholdTuning.FFTSpeedupThreshold

	// ParallelSpeedupThreshold is the analogous ratio for the parallel
	// multiplication path. See config.ThresholdTuningProfile.
	ParallelSpeedupThreshold = config.DefaultThresholdTuning.ParallelSpeedupThreshold

	// HysteresisMargin is the minimum relative change required before a
	// new threshold is committed; damps oscillation. See
	// config.ThresholdTuningProfile.
	HysteresisMargin = config.DefaultThresholdTuning.HysteresisMargin
)

// DynamicThresholdManager adjusts FFT and parallel thresholds during calculation
// based on observed performance metrics.
//
// The manager is the orchestrator: it owns the thresholds and the lifecycle
// state, delegates sample storage to MetricsBuffer, and delegates the
// analytical work (speedup, hysteresis, multiplicative adjustment) to
// ThresholdAnalyzer.
type DynamicThresholdManager struct {
	mu     sync.RWMutex
	logger zerolog.Logger

	// Current thresholds (can be adjusted during calculation).
	currentFFTThreshold      int
	currentParallelThreshold int

	// Original thresholds (for comparison and bounds).
	originalFFTThreshold      int
	originalParallelThreshold int

	// Metrics storage and analysis.
	buffer   MetricsBuffer
	analyzer ThresholdAnalyzer

	// Adjustment state.
	iterationCount     int
	adjustmentInterval int
	lastAdjustment     time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// Constructor and Configuration
// ─────────────────────────────────────────────────────────────────────────────

// NewDynamicThresholdManager creates a new manager with the given initial thresholds.
func NewDynamicThresholdManager(fftThreshold, parallelThreshold int) *DynamicThresholdManager {
	return &DynamicThresholdManager{
		logger:                    zerolog.Nop(),
		currentFFTThreshold:       fftThreshold,
		currentParallelThreshold:  parallelThreshold,
		originalFFTThreshold:      fftThreshold,
		originalParallelThreshold: parallelThreshold,
		adjustmentInterval:        DynamicAdjustmentInterval,
	}
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

	return &DynamicThresholdManager{
		logger:                    zerolog.Nop(),
		currentFFTThreshold:       cfg.InitialFFTThreshold,
		currentParallelThreshold:  cfg.InitialParallelThreshold,
		originalFFTThreshold:      cfg.InitialFFTThreshold,
		originalParallelThreshold: cfg.InitialParallelThreshold,
		adjustmentInterval:        interval,
	}
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
// No mutex is taken: this is called from the single doubling-loop
// goroutine. Reader-side getters (GetThresholds/GetStats) take RLock.
func (m *DynamicThresholdManager) RecordIteration(bitLen int, duration time.Duration, usedFFT, usedParallel bool) {
	m.buffer.Record(bitLen, duration, usedFFT, usedParallel)
	m.iterationCount++
}

// ─────────────────────────────────────────────────────────────────────────────
// Threshold Access
// ─────────────────────────────────────────────────────────────────────────────

// GetThresholds returns the current FFT and parallel thresholds.
func (m *DynamicThresholdManager) GetThresholds() (fft, parallel int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentFFTThreshold, m.currentParallelThreshold
}

// GetFFTThreshold returns the current FFT threshold.
func (m *DynamicThresholdManager) GetFFTThreshold() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentFFTThreshold
}

// GetParallelThreshold returns the current parallel threshold.
func (m *DynamicThresholdManager) GetParallelThreshold() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentParallelThreshold
}

// ─────────────────────────────────────────────────────────────────────────────
// Adjustment Logic
// ─────────────────────────────────────────────────────────────────────────────

// ShouldAdjust checks if thresholds should be adjusted based on collected metrics.
// Returns the new thresholds and whether an adjustment was made.
// No mutex needed: called from single goroutine in the doubling loop.
func (m *DynamicThresholdManager) ShouldAdjust() (newFFT, newParallel int, adjusted bool) {
	if m.iterationCount%m.adjustmentInterval != 0 {
		return m.currentFFTThreshold, m.currentParallelThreshold, false
	}

	if m.buffer.Count() < MinMetricsForAdjustment {
		return m.currentFFTThreshold, m.currentParallelThreshold, false
	}

	newFFT = m.analyzeFFTThreshold()
	newParallel = m.analyzeParallelThreshold()

	fftChanged := m.analyzer.SignificantChange(m.currentFFTThreshold, newFFT)
	parallelChanged := m.analyzer.SignificantChange(m.currentParallelThreshold, newParallel)

	if !fftChanged && !parallelChanged {
		return m.currentFFTThreshold, m.currentParallelThreshold, false
	}

	oldFFT := m.currentFFTThreshold
	oldParallel := m.currentParallelThreshold
	if fftChanged {
		m.currentFFTThreshold = newFFT
	}
	if parallelChanged {
		m.currentParallelThreshold = newParallel
	}
	m.lastAdjustment = time.Now()
	m.logger.Debug().
		Int("iteration", m.iterationCount).
		Bool("fft_changed", fftChanged).
		Int("fft_old", oldFFT).
		Int("fft_new", m.currentFFTThreshold).
		Bool("parallel_changed", parallelChanged).
		Int("parallel_old", oldParallel).
		Int("parallel_new", m.currentParallelThreshold).
		Msg("thresholds adjusted")
	return m.currentFFTThreshold, m.currentParallelThreshold, true
}

// analyzeFFTThreshold delegates to the analyzer with FFT-specific parameters.
func (m *DynamicThresholdManager) analyzeFFTThreshold() int {
	return m.analyzer.Analyze(m.buffer.RecentMetrics(), AnalysisParams{
		Predicate:         func(metric IterationMetric) bool { return metric.UsedFFT },
		SpeedupThreshold:  FFTSpeedupThreshold,
		LowerNumerator:    9,
		RaiseNumerator:    11,
		MinThreshold:      config.DefaultThresholdTuning.MinFFTThreshold,
		MaxCapMultiplier:  2,
		CurrentThreshold:  m.currentFFTThreshold,
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
		MinThreshold:      config.DefaultThresholdTuning.MinParallelThreshold,
		MaxCapMultiplier:  4,
		CurrentThreshold:  m.currentParallelThreshold,
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
	m.mu.RLock()
	defer m.mu.RUnlock()

	return ThresholdStats{
		CurrentFFT:          m.currentFFTThreshold,
		CurrentParallel:     m.currentParallelThreshold,
		OriginalFFT:         m.originalFFTThreshold,
		OriginalParallel:    m.originalParallelThreshold,
		MetricsCollected:    m.buffer.Count(),
		IterationsProcessed: m.iterationCount,
	}
}

// Reset clears all collected metrics and restores original thresholds.
func (m *DynamicThresholdManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentFFTThreshold = m.originalFFTThreshold
	m.currentParallelThreshold = m.originalParallelThreshold
	m.buffer.Reset()
	m.iterationCount = 0
}
