// Package cli provides progress tracking with ETA estimation.
package cli

import (
	"fmt"
	"time"
)

// ProgressWithETA extends ProgressState with time estimation capabilities.
// It tracks progress updates and calculates the estimated time remaining
// based on the rate of progress.
type ProgressWithETA struct {
	*ProgressState
	startTime    time.Time
	lastUpdate   time.Time
	lastProgress float64
	progressRate float64 // smoothed progress rate (progress per second)
}

// NewProgressWithETA creates a new progress tracker with ETA calculation.
//
// Parameters:
//   - numCalculators: The number of calculators being tracked.
//
// Returns:
//   - *ProgressWithETA: A new progress tracker with ETA support.
func NewProgressWithETA(numCalculators int) *ProgressWithETA {
	now := time.Now()
	return &ProgressWithETA{
		ProgressState: NewProgressState(numCalculators),
		startTime:     now,
		lastUpdate:    now,
		lastProgress:  0,
		progressRate:  0,
	}
}

// UpdateWithETA updates progress for a specific calculator and calculates ETA.
// It uses exponential smoothing for the progress rate to provide stable
// estimates even with variable progress updates.
//
// Parameters:
//   - index: The index of the calculator (0 to numCalculators-1).
//   - value: The new progress value (0.0 to 1.0).
//
// Returns:
//   - progress: The current average progress (0.0 to 1.0).
//   - eta: The estimated time remaining, or 0 if calculation started recently.
func (p *ProgressWithETA) UpdateWithETA(index int, value float64) (progress float64, eta time.Duration) {
	p.Update(index, value)
	progress = p.CalculateAverage()

	now := time.Now()
	elapsed := now.Sub(p.startTime)

	// Need some elapsed time and progress to make meaningful estimates
	if elapsed < 100*time.Millisecond || progress <= 0.001 {
		p.lastUpdate = now
		p.lastProgress = progress
		return progress, 0
	}

	// Calculate instantaneous rate if enough time has passed
	timeSinceUpdate := now.Sub(p.lastUpdate).Seconds()
	if timeSinceUpdate > 0.05 { // At least 50ms between updates
		progressDelta := progress - p.lastProgress
		if progressDelta > 0 {
			instantRate := progressDelta / timeSinceUpdate

			// Exponential smoothing: 70% old rate, 30% new rate
			if p.progressRate > 0 {
				p.progressRate = 0.7*p.progressRate + 0.3*instantRate
			} else {
				// First meaningful rate calculation - use simple estimation
				p.progressRate = progress / elapsed.Seconds()
			}
		}

		p.lastUpdate = now
		p.lastProgress = progress
	}

	// Calculate ETA based on smoothed rate
	if p.progressRate > 0 && progress < 1.0 {
		remaining := 1.0 - progress
		etaSeconds := remaining / p.progressRate
		eta = time.Duration(etaSeconds * float64(time.Second))

		// Cap ETA at reasonable values
		if eta > 24*time.Hour {
			eta = 24 * time.Hour
		}
	}

	return progress, eta
}

// GetETA calculates the current ETA without updating progress.
// Useful for getting an estimate between progress updates.
//
// Returns:
//   - eta: The estimated time remaining based on current progress rate.
func (p *ProgressWithETA) GetETA() time.Duration {
	progress := p.CalculateAverage()
	if p.progressRate <= 0 || progress >= 1.0 {
		return 0
	}

	remaining := 1.0 - progress
	etaSeconds := remaining / p.progressRate
	eta := time.Duration(etaSeconds * float64(time.Second))

	if eta > 24*time.Hour {
		eta = 24 * time.Hour
	}

	return eta
}

// FormatETA formats a duration into a human-readable ETA string.
// It adapts the format based on the magnitude of the duration.
//
// Parameters:
//   - eta: The duration to format.
//
// Returns:
//   - string: A formatted string like "< 1s", "2m30s", "1h15m".
func FormatETA(eta time.Duration) string {
	if eta <= 0 {
		return "calculating..."
	}

	if eta < time.Second {
		return "< 1s"
	}

	if eta < time.Minute {
		return fmt.Sprintf("%ds", int(eta.Seconds()))
	}

	if eta < time.Hour {
		minutes := int(eta.Minutes())
		seconds := int(eta.Seconds()) % 60
		if seconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(eta.Hours())
	minutes := int(eta.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// FormatProgressBarWithETA generates a formatted progress string with ETA.
// It combines the progress percentage, visual bar, and time estimate.
//
// Parameters:
//   - progress: The normalized progress value (0.0 to 1.0).
//   - eta: The estimated time remaining.
//   - width: The width of the progress bar in characters.
//
// Returns:
//   - string: A formatted string like "45.00% [████░░░░] ETA: 2m30s".
func FormatProgressBarWithETA(progress float64, eta time.Duration, width int) string {
	bar := progressBar(progress, width)
	etaStr := FormatETA(eta)
	return fmt.Sprintf("%6.2f%% [%s] ETA: %s", progress*100, bar, etaStr)
}
