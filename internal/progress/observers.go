// This file contains concrete observer implementations for the Observer pattern.

package progress

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
