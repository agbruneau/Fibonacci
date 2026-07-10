// This file contains concrete observer implementations for the Observer pattern.

package progress

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

// ─────────────────────────────────────────────────────────────────────────────
// Channel Observer (Backward Compatibility)
// ─────────────────────────────────────────────────────────────────────────────

// ChannelObserver adapts the Observer pattern to channel-based communication.
// This maintains backward compatibility with existing UI code that expects
// progress updates via channels.
type ChannelObserver struct {
	channel chan<- ProgressUpdate
}

// NewChannelObserver creates an observer that sends updates to a channel.
// The channel should have sufficient buffer capacity to avoid blocking.
//
// Parameters:
//   - ch: The channel to send progress updates to. If nil, updates are discarded.
//
// Returns:
//   - *ChannelObserver: A new observer that forwards to the channel.
func NewChannelObserver(ch chan<- ProgressUpdate) *ChannelObserver {
	return &ChannelObserver{channel: ch}
}

// Update implements ProgressObserver by sending to the channel.
// Uses non-blocking send to avoid deadlocks when the channel is full.
//
// Parameters:
//   - calcIndex: The calculator instance identifier.
//   - progress: The normalized progress value (0.0 to 1.0).
func (o *ChannelObserver) Update(calcIndex int, progress float64) {
	if o.channel == nil {
		return
	}

	// Clamp progress to valid range
	if progress > 1.0 {
		progress = 1.0
	}

	update := ProgressUpdate{CalculatorIndex: calcIndex, Value: progress}

	// Non-blocking send to avoid deadlocks
	select {
	case o.channel <- update:
	default:
		// Channel full, drop update (UI will catch up on next update)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Logging Observer
// ─────────────────────────────────────────────────────────────────────────────

// LoggingObserver logs progress updates using zerolog.
// It throttles logging based on a threshold to avoid log spam.
type LoggingObserver struct {
	logger    zerolog.Logger
	threshold float64         // Minimum progress change to log
	lastLog   map[int]float64 // Last logged progress per calculator
	mu        sync.Mutex
}

// NewLoggingObserver creates an observer that logs progress.
// It only logs when progress changes by at least the threshold amount.
//
// Parameters:
//   - logger: The zerolog logger to use.
//   - threshold: Minimum progress change to trigger a log (e.g., 0.1 for 10%).
//
// Returns:
//   - *LoggingObserver: A new observer that logs to zerolog.
func NewLoggingObserver(logger zerolog.Logger, threshold float64) *LoggingObserver {
	if threshold <= 0 {
		threshold = 0.1 // Default to 10%
	}
	return &LoggingObserver{
		logger:    logger,
		threshold: threshold,
		lastLog:   make(map[int]float64),
	}
}

// Update implements ProgressObserver by logging significant progress changes.
//
// Parameters:
//   - calcIndex: The calculator instance identifier.
//   - progress: The normalized progress value (0.0 to 1.0).
func (o *LoggingObserver) Update(calcIndex int, progress float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	lastProgress := o.lastLog[calcIndex]

	// Log at boundaries or significant changes
	shouldLog := progress >= 1.0 ||
		lastProgress == 0 && progress > 0 ||
		progress-lastProgress >= o.threshold

	if shouldLog {
		o.logger.Debug().
			Int("calculator", calcIndex).
			Float64("progress", progress).
			Str("percent", fmt.Sprintf("%.1f%%", progress*100)).
			Msg("calculation progress")
		o.lastLog[calcIndex] = progress
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// No-Op Observer (Null Object Pattern)
// ─────────────────────────────────────────────────────────────────────────────

// NoOpObserver is a null object that discards all progress updates.
// Useful for testing or when progress tracking is not needed.
type NoOpObserver struct{}

// NewNoOpObserver creates a no-op observer that discards updates.
//
// Returns:
//   - *NoOpObserver: A new no-op observer.
func NewNoOpObserver() *NoOpObserver {
	return &NoOpObserver{}
}

// Update implements ProgressObserver by doing nothing.
func (o *NoOpObserver) Update(calcIndex int, progress float64) {
	// Intentionally empty - Null Object pattern
}
