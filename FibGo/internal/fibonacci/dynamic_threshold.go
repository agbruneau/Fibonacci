// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file implements dynamic threshold adjustment during calculation.
package fibonacci

import (
	"sync"
	"time"
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

	// FFTSpeedupThreshold is the minimum speedup ratio to switch to FFT.
	// If FFT is expected to be at least this much faster, switch to it.
	FFTSpeedupThreshold = 1.2

	// ParallelSpeedupThreshold is the minimum speedup to enable parallelism.
	ParallelSpeedupThreshold = 1.1

	// HysteresisMargin prevents oscillating between modes.
	// Threshold must change by at least this factor to trigger adjustment.
	HysteresisMargin = 0.15
)

// DynamicThresholdManager adjusts FFT and parallel thresholds during calculation
// based on observed performance metrics.
type DynamicThresholdManager struct {
	mu sync.RWMutex

	// Current thresholds (can be adjusted during calculation)
	currentFFTThreshold      int
	currentParallelThreshold int

	// Original thresholds (for comparison and bounds)
	originalFFTThreshold      int
	originalParallelThreshold int

	// Collected metrics - implemented as a Ring Buffer for O(1) ops
	metrics      [MaxMetricsHistory]IterationMetric
	metricsCount int // Total metrics collected (ever)
	metricsHead  int // Index of the next slot to write to

	// Running sums for fast average calculation (O(1))
	// We track separate sums for different categories to avoid iterating.
	// Note: These sums are approximate as they cover the window, but simplifying updates is key.
	// Actually, accurate running sums require removing the overwritten value.
	// Given MaxMetricsHistory is small (20), iterating is fast enough, but let's optimize slightly.
	// We will stick to iterating the small ring buffer for simplicity in categorization (FFT vs Non-FFT),
	// but the structure itself is now a fixed array to avoid allocation.

	// Adjustment state
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
		currentFFTThreshold:       cfg.InitialFFTThreshold,
		currentParallelThreshold:  cfg.InitialParallelThreshold,
		originalFFTThreshold:      cfg.InitialFFTThreshold,
		originalParallelThreshold: cfg.InitialParallelThreshold,
		adjustmentInterval:        interval,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Metric Recording
// ─────────────────────────────────────────────────────────────────────────────

// RecordIteration records timing data for a completed iteration.
// This should be called after each doubling step in the algorithm.
func (m *DynamicThresholdManager) RecordIteration(bitLen int, duration time.Duration, usedFFT, usedParallel bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric := IterationMetric{
		BitLen:       bitLen,
		Duration:     duration,
		UsedFFT:      usedFFT,
		UsedParallel: usedParallel,
	}

	// Write to ring buffer
	m.metrics[m.metricsHead] = metric
	m.metricsHead = (m.metricsHead + 1) % MaxMetricsHistory
	m.metricsCount++
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
func (m *DynamicThresholdManager) ShouldAdjust() (newFFT, newParallel int, adjusted bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we should evaluate adjustments
	if m.iterationCount%m.adjustmentInterval != 0 {
		return m.currentFFTThreshold, m.currentParallelThreshold, false
	}

	if m.metricsCount < MinMetricsForAdjustment {
		return m.currentFFTThreshold, m.currentParallelThreshold, false
	}

	// Analyze recent metrics to determine if adjustments are beneficial
	newFFT = m.analyzeFFTThreshold()
	newParallel = m.analyzeParallelThreshold()

	// Check if changes are significant enough (hysteresis)
	fftChanged := m.significantChange(m.currentFFTThreshold, newFFT)
	parallelChanged := m.significantChange(m.currentParallelThreshold, newParallel)

	if fftChanged || parallelChanged {
		if fftChanged {
			m.currentFFTThreshold = newFFT
		}
		if parallelChanged {
			m.currentParallelThreshold = newParallel
		}
		m.lastAdjustment = time.Now()
		return m.currentFFTThreshold, m.currentParallelThreshold, true
	}

	return m.currentFFTThreshold, m.currentParallelThreshold, false
}

// getActiveMetrics returns a slice of valid metrics from the ring buffer.
func (m *DynamicThresholdManager) getActiveMetrics() []IterationMetric {
	count := m.metricsCount
	if count > MaxMetricsHistory {
		count = MaxMetricsHistory
	}

	// Create a temporary slice to make analysis easier without complex ring buffer arithmetic
	// Since MaxMetricsHistory is small (20), this copy is cheap and simplifies logic.
	result := make([]IterationMetric, count)

	if m.metricsCount <= MaxMetricsHistory {
		copy(result, m.metrics[:count])
	} else {
		// Buffer wrapped around. Order: [Head...End] + [0...Head-1]
		// Actually, since we just need the values and not strict temporal order for averages,
		// we can just copy the whole array.
		// However, to be strictly correct if we needed order:
		// tailLen := MaxMetricsHistory - m.metricsHead
		// copy(result, m.metrics[m.metricsHead:])
		// copy(result[tailLen:], m.metrics[:m.metricsHead])
		// For averages, order doesn't matter.
		copy(result, m.metrics[:])
	}
	return result
}

// analyzeFFTThreshold analyzes metrics to determine optimal FFT threshold.
func (m *DynamicThresholdManager) analyzeFFTThreshold() int {
	metrics := m.getActiveMetrics()
	if len(metrics) == 0 {
		return m.currentFFTThreshold
	}

	// Find the bit length where FFT started being used
	// and analyze if it was beneficial based on timing trends
	var fftMetrics, nonFFTMetrics []IterationMetric
	// Pre-allocate assuming 50/50 split to avoid frequent re-allocation
	fftMetrics = make([]IterationMetric, 0, len(metrics))
	nonFFTMetrics = make([]IterationMetric, 0, len(metrics))

	for _, metric := range metrics {
		if metric.UsedFFT {
			fftMetrics = append(fftMetrics, metric)
		} else {
			nonFFTMetrics = append(nonFFTMetrics, metric)
		}
	}

	// Not enough data to analyze
	if len(fftMetrics) == 0 || len(nonFFTMetrics) == 0 {
		return m.currentFFTThreshold
	}

	// Calculate average time per bit for FFT vs non-FFT
	avgFFTTimePerBit := m.avgTimePerBit(fftMetrics)
	avgNonFFTTimePerBit := m.avgTimePerBit(nonFFTMetrics)

	// If FFT is significantly faster per bit, lower the threshold
	if avgFFTTimePerBit > 0 && avgNonFFTTimePerBit > 0 {
		ratio := avgNonFFTTimePerBit / avgFFTTimePerBit
		if ratio > FFTSpeedupThreshold {
			// FFT is faster, lower threshold by 10%
			newThreshold := m.currentFFTThreshold * 9 / 10
			// Don't go below a reasonable minimum
			if newThreshold < 100000 {
				newThreshold = 100000
			}
			return newThreshold
		} else if ratio < 1.0/FFTSpeedupThreshold {
			// FFT is slower, raise threshold by 10%
			newThreshold := m.currentFFTThreshold * 11 / 10
			// Don't exceed original by too much
			if newThreshold > m.originalFFTThreshold*2 {
				newThreshold = m.originalFFTThreshold * 2
			}
			return newThreshold
		}
	}

	return m.currentFFTThreshold
}

// analyzeParallelThreshold analyzes metrics to determine optimal parallel threshold.
func (m *DynamicThresholdManager) analyzeParallelThreshold() int {
	metrics := m.getActiveMetrics()
	if len(metrics) == 0 {
		return m.currentParallelThreshold
	}

	// Analyze if parallelism was beneficial
	var parallelMetrics, sequentialMetrics []IterationMetric
	parallelMetrics = make([]IterationMetric, 0, len(metrics))
	sequentialMetrics = make([]IterationMetric, 0, len(metrics))

	for _, metric := range metrics {
		if metric.UsedParallel {
			parallelMetrics = append(parallelMetrics, metric)
		} else {
			sequentialMetrics = append(sequentialMetrics, metric)
		}
	}

	// Not enough data
	if len(parallelMetrics) == 0 || len(sequentialMetrics) == 0 {
		return m.currentParallelThreshold
	}

	// Compare performance at similar bit lengths
	avgParallelTimePerBit := m.avgTimePerBit(parallelMetrics)
	avgSequentialTimePerBit := m.avgTimePerBit(sequentialMetrics)

	if avgParallelTimePerBit > 0 && avgSequentialTimePerBit > 0 {
		ratio := avgSequentialTimePerBit / avgParallelTimePerBit
		if ratio > ParallelSpeedupThreshold {
			// Parallel is faster, lower threshold
			newThreshold := m.currentParallelThreshold * 8 / 10
			if newThreshold < 1024 {
				newThreshold = 1024
			}
			return newThreshold
		} else if ratio < 1.0/ParallelSpeedupThreshold {
			// Parallel is slower (overhead), raise threshold
			newThreshold := m.currentParallelThreshold * 12 / 10
			if newThreshold > m.originalParallelThreshold*4 {
				newThreshold = m.originalParallelThreshold * 4
			}
			return newThreshold
		}
	}

	return m.currentParallelThreshold
}

// avgTimePerBit calculates average time per bit across metrics.
func (m *DynamicThresholdManager) avgTimePerBit(metrics []IterationMetric) float64 {
	if len(metrics) == 0 {
		return 0
	}

	var totalTime time.Duration
	var totalBits int64
	for _, metric := range metrics {
		totalTime += metric.Duration
		totalBits += int64(metric.BitLen)
	}

	if totalBits == 0 {
		return 0
	}

	return float64(totalTime.Nanoseconds()) / float64(totalBits)
}

// significantChange checks if a threshold change is significant enough to apply.
func (m *DynamicThresholdManager) significantChange(oldVal, newVal int) bool {
	if oldVal == 0 {
		return newVal != 0
	}
	change := float64(newVal-oldVal) / float64(oldVal)
	if change < 0 {
		change = -change
	}
	return change > HysteresisMargin
}

// ─────────────────────────────────────────────────────────────────────────────
// Statistics and Reporting
// ─────────────────────────────────────────────────────────────────────────────

// GetStats returns current statistics about the manager.
func (m *DynamicThresholdManager) GetStats() ThresholdStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := m.metricsCount
	if count > MaxMetricsHistory {
		count = MaxMetricsHistory
	}

	return ThresholdStats{
		CurrentFFT:          m.currentFFTThreshold,
		CurrentParallel:     m.currentParallelThreshold,
		OriginalFFT:         m.originalFFTThreshold,
		OriginalParallel:    m.originalParallelThreshold,
		MetricsCollected:    count,
		IterationsProcessed: m.iterationCount,
	}
}

// Reset clears all collected metrics and restores original thresholds.
func (m *DynamicThresholdManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentFFTThreshold = m.originalFFTThreshold
	m.currentParallelThreshold = m.originalParallelThreshold
	// Ring buffer reset is simple
	m.metricsCount = 0
	m.metricsHead = 0
	m.iterationCount = 0
}
