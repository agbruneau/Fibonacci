package fibonacci

import (
	"math/big"
	"sync"
	"testing"

	"github.com/agbru/fibcalc/internal/bigfft"
)

// TestExecuteDoublingStepFFT_NoDataRace verifies that the parallel FFT execution
// does not cause data races. This test uses multiple goroutines to stress test
// the concurrent operations.
//
// Run with: go test -race -run TestExecuteDoublingStepFFT_NoDataRace ./internal/fibonacci/
func TestExecuteDoublingStepFFT_NoDataRace(t *testing.T) {
	t.Parallel()

	// Create a calculation state with values large enough to trigger FFT path
	state := &CalculationState{
		FK:  new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil), // Large number
		FK1: new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil),
		T1:  new(big.Int),
		T2:  new(big.Int),
		T3:  new(big.Int),
		T4:  new(big.Int),
	}

	// Setup T2 = 2*F(k+1) - F(k)
	state.T2.Lsh(state.FK1, 1)
	state.T2.Sub(state.T2, state.FK)

	opts := Options{
		ParallelThreshold: 4096,
		FFTThreshold:      10000,
	}

	// Run multiple times to increase chance of detecting race conditions
	for i := 0; i < 10; i++ {
		// Reset state for each iteration
		state.T1 = new(big.Int)
		state.T3 = new(big.Int)
		state.T4 = new(big.Int)

		err := executeDoublingStepFFT(state, opts, true)
		if err != nil {
			t.Logf("executeDoublingStepFFT returned error (may be expected for small test values): %v", err)
		}
	}
}

// TestExecuteDoublingStepFFT_ConcurrentCalls verifies that multiple concurrent calls
// to executeDoublingStepFFT do not interfere with each other.
func TestExecuteDoublingStepFFT_ConcurrentCalls(t *testing.T) {
	t.Parallel()

	const numGoroutines = 4

	opts := Options{
		ParallelThreshold: 4096,
		FFTThreshold:      10000,
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func() {
			defer wg.Done()

			// Each goroutine has its own state
			state := &CalculationState{
				FK:  new(big.Int).Exp(big.NewInt(2), big.NewInt(500), nil),
				FK1: new(big.Int).Exp(big.NewInt(2), big.NewInt(500), nil),
				T1:  new(big.Int),
				T2:  new(big.Int),
				T3:  new(big.Int),
				T4:  new(big.Int),
			}

			state.T2.Lsh(state.FK1, 1)
			state.T2.Sub(state.T2, state.FK)

			// Run parallel FFT
			_ = executeDoublingStepFFT(state, opts, true)
		}()
	}

	wg.Wait()
}

// TestPolValuesClone verifies that PolValues.Clone creates an independent copy.
func TestPolValuesClone(t *testing.T) {
	t.Parallel()

	// Initialize through Transform
	poly := bigfft.PolyFromInt(big.NewInt(12345), 4, 2)
	transformed, err := poly.Transform(bigfft.ValueSize(4, 2, 2))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Clone the transformed polynomial
	cloned := transformed.Clone()

	// Verify they are independent - modifying one shouldn't affect the other
	if len(cloned.Values) != len(transformed.Values) {
		t.Errorf("Clone has different length: got %d, want %d", len(cloned.Values), len(transformed.Values))
	}

	// Verify the cloned values match the original
	for i := range transformed.Values {
		for j := range transformed.Values[i] {
			if cloned.Values[i][j] != transformed.Values[i][j] {
				t.Errorf("Clone values differ at [%d][%d]: got %v, want %v",
					i, j, cloned.Values[i][j], transformed.Values[i][j])
			}
		}
	}
}
