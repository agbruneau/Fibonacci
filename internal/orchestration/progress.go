package orchestration

import (
	"time"

	"github.com/agbru/fibcalc/internal/format"
	"github.com/agbru/fibcalc/internal/progress"
)

// ProgressAggregator manages multi-calculator progress aggregation with ETA
// estimation. It combines the per-calculator progress state (delegated to
// format.ProgressState) with time-based smoothing to produce a stable ETA.
//
// Both CLI and TUI use this type so that aggregation, smoothing, and ETA
// computation are not duplicated across UI layers.
//
// ProgressAggregator is NOT thread-safe. It is designed to be accessed from a
// single consumer goroutine that reads from the progress channel.
type ProgressAggregator struct {
	state          *format.ProgressState
	numCalculators int

	// ETA computation state.
	startTime    time.Time
	lastUpdate   time.Time
	lastProgress float64
	progressRate float64 // smoothed progress rate (progress per second)
}

// NewProgressAggregator creates a new aggregator for the given number
// of calculators. Returns nil if numCalculators <= 0.
func NewProgressAggregator(numCalculators int) *ProgressAggregator {
	if numCalculators <= 0 {
		return nil
	}
	now := time.Now()
	return &ProgressAggregator{
		state:          format.NewProgressState(numCalculators),
		numCalculators: numCalculators,
		startTime:      now,
		lastUpdate:     now,
	}
}

// AggregatedProgress holds the result of processing a single progress update.
type AggregatedProgress struct {
	// CalculatorIndex is the index of the calculator that sent the update.
	CalculatorIndex int
	// Value is the raw progress value from the update (0.0 to 1.0).
	Value float64
	// AverageProgress is the aggregated average across all calculators.
	AverageProgress float64
	// ETA is the estimated time remaining based on smoothed progress rate.
	ETA time.Duration
}

// Update processes a single progress update and returns the aggregated result.
// It refreshes the smoothed progress rate using exponential smoothing so the
// ETA stays stable even when updates arrive at irregular intervals.
func (a *ProgressAggregator) Update(update progress.ProgressUpdate) AggregatedProgress {
	a.state.Update(update.CalculatorIndex, update.Value)
	avg := a.state.CalculateAverage()
	eta := a.recomputeETA(avg)

	return AggregatedProgress{
		CalculatorIndex: update.CalculatorIndex,
		Value:           update.Value,
		AverageProgress: avg,
		ETA:             eta,
	}
}

// recomputeETA refreshes the smoothed progress rate and returns the resulting
// ETA. It mirrors the previous behaviour of format.ProgressWithETA: requires
// some elapsed time and progress before producing a meaningful estimate, uses
// 70/30 exponential smoothing, and caps the ETA at 24h.
func (a *ProgressAggregator) recomputeETA(progress float64) time.Duration {
	now := time.Now()
	elapsed := now.Sub(a.startTime)

	// Need some elapsed time and progress to make meaningful estimates.
	if elapsed < 100*time.Millisecond || progress <= 0.001 {
		a.lastUpdate = now
		a.lastProgress = progress
		return 0
	}

	// Calculate instantaneous rate if enough time has passed.
	timeSinceUpdate := now.Sub(a.lastUpdate).Seconds()
	if timeSinceUpdate > 0.05 { // At least 50ms between updates.
		progressDelta := progress - a.lastProgress
		if progressDelta > 0 {
			instantRate := progressDelta / timeSinceUpdate

			// Exponential smoothing: 70% old rate, 30% new rate.
			if a.progressRate > 0 {
				a.progressRate = 0.7*a.progressRate + 0.3*instantRate
			} else {
				// First meaningful rate calculation - use simple estimation.
				a.progressRate = progress / elapsed.Seconds()
			}
		}

		a.lastUpdate = now
		a.lastProgress = progress
	}

	return capETA(a.progressRate, progress)
}

// CalculateAverage returns the current average progress without updating.
// Useful for periodic refresh between updates (e.g., CLI ticker).
func (a *ProgressAggregator) CalculateAverage() float64 {
	return a.state.CalculateAverage()
}

// GetETA returns the current ETA estimate without updating progress.
// Useful for periodic refresh between updates (e.g., CLI ticker).
func (a *ProgressAggregator) GetETA() time.Duration {
	return capETA(a.progressRate, a.state.CalculateAverage())
}

// NumCalculators returns the number of calculators being tracked.
func (a *ProgressAggregator) NumCalculators() int {
	return a.numCalculators
}

// IsMultiCalculator returns true if tracking more than one calculator.
func (a *ProgressAggregator) IsMultiCalculator() bool {
	return a.numCalculators > 1
}

// DrainChannel reads all updates from the channel without processing.
// Use this when numCalculators <= 0 and updates should be discarded.
func DrainChannel(progressChan <-chan progress.ProgressUpdate) {
	for range progressChan {
	}
}

// capETA converts a smoothed rate + progress into a remaining duration, capped
// at 24h. Returns 0 when the estimate is not yet meaningful.
func capETA(rate, progress float64) time.Duration {
	if rate <= 0 || progress >= 1.0 {
		return 0
	}
	remaining := 1.0 - progress
	eta := time.Duration((remaining / rate) * float64(time.Second))
	if eta > 24*time.Hour {
		eta = 24 * time.Hour
	}
	return eta
}
