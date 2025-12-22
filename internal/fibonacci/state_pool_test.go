package fibonacci

import "testing"

// TestReleaseState_NilSafe verifies that ReleaseState handles nil input safely.
// This is a critical fix to prevent panics when ReleaseState is called with nil.
func TestReleaseState_NilSafe(t *testing.T) {
	t.Parallel()

	// Should not panic when called with nil
	ReleaseState(nil)
}

// TestAcquireAndReleaseState_RoundTrip tests the normal acquire/release cycle.
func TestAcquireAndReleaseState_RoundTrip(t *testing.T) {
	t.Parallel()

	state := AcquireState()
	if state == nil {
		t.Fatal("AcquireState returned nil")
	}

	// Verify state is properly initialized
	if state.FK == nil || state.FK1 == nil {
		t.Error("State FK/FK1 should be initialized")
	}

	// Release should not panic
	ReleaseState(state)
}
