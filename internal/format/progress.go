package format

// ProgressState encapsulates the aggregated progress of concurrent calculations.
// It maintains the individual progress of each calculator and computes the
// average, which is essential for providing a consolidated progress view when
// multiple algorithms are running in parallel.
//
// ProgressState is a primitive aggregation type. Higher-level concerns such as
// ETA estimation live in the orchestration layer, on top of this type.
type ProgressState struct {
	progresses     []float64
	numCalculators int
}

// NewProgressState creates and initializes a new ProgressState.
// It sets up the internal storage for tracking the progress of a specified
// number of calculators.
//
// Parameters:
//   - numCalculators: The number of calculators to track.
//
// Returns:
//   - *ProgressState: A pointer to the new progress state object.
func NewProgressState(numCalculators int) *ProgressState {
	return &ProgressState{
		progresses:     make([]float64, numCalculators),
		numCalculators: numCalculators,
	}
}

// Update records a new progress value for a specific calculator.
// Note: This type is NOT thread-safe. It is designed to be accessed
// from a single goroutine (the select loop in DisplayProgress).
//
// Parameters:
//   - index: The index of the calculator (0 to numCalculators-1).
//   - value: The progress value (0.0 to 1.0).
func (ps *ProgressState) Update(index int, value float64) {
	if index >= 0 && index < len(ps.progresses) {
		ps.progresses[index] = value
	}
}

// CalculateAverage computes the average progress across all tracked calculators.
// This is used to display a single, consolidated progress bar to the user,
// representing the overall progress of the application.
//
// Returns:
//   - float64: The average progress (0.0 to 1.0).
func (ps *ProgressState) CalculateAverage() float64 {
	var totalProgress float64
	for _, p := range ps.progresses {
		totalProgress += p
	}
	if ps.numCalculators == 0 {
		return 0.0
	}
	return totalProgress / float64(ps.numCalculators)
}

// NumCalculators returns the number of calculators being tracked.
func (ps *ProgressState) NumCalculators() int {
	return ps.numCalculators
}
