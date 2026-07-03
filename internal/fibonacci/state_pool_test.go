package fibonacci

import (
	"sync"
	"testing"
)

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

	state := AcquireStateForN(0)
	if state == nil {
		t.Fatal("AcquireStateForN returned nil")
	}

	// Verify state is properly initialized
	if state.FK == nil || state.FK1 == nil {
		t.Error("State FK/FK1 should be initialized")
	}

	// Release should not panic
	ReleaseState(state)
}

// TestCalculationStatePool_ConcurrentAllocation verifies that the CalculationState
// and matrixState pools are safe for concurrent Get/Put operations. 100 goroutines
// perform simultaneous Acquire/Release cycles and all must complete without panics.
func TestCalculationStatePool_ConcurrentAllocation(t *testing.T) {
	t.Parallel()

	const goroutines = 100

	t.Run("CalculationState", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				state := AcquireStateForN(0)
				if state == nil {
					t.Error("AcquireStateForN returned nil")
					return
				}
				if state.FK == nil || state.FK1 == nil {
					t.Error("State FK/FK1 should be initialized")
				}
				ReleaseState(state)
			}()
		}

		wg.Wait()
	})

	t.Run("matrixState", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				state := acquireMatrixState()
				if state == nil {
					t.Error("acquireMatrixState returned nil")
					return
				}
				releaseMatrixState(state)
			}()
		}

		wg.Wait()
	})
}
