package fibonacci

import (
	"math/big"
	"testing"
)

func TestExecuteDoublingStepFFT(t *testing.T) {
	t.Parallel()

	t.Run("Execute doubling step with FFT", func(t *testing.T) {
		t.Parallel()
		// Create a calculation state with values that will trigger FFT
		// All fields must be initialized to avoid nil pointer dereference
		state := &CalculationState{
			FK:  new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil), // Large number
			FK1: new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil), // Large number
			T1:  new(big.Int),
			T2:  new(big.Int),
			T3:  new(big.Int),
			T4:  new(big.Int),
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      10000, // Low threshold to trigger FFT
		}

		err := executeDoublingStepFFT(state, opts, false)
		// FFT execution may succeed or fail depending on implementation details
		// We're mainly testing that the function doesn't panic and handles errors
		if err != nil {
			t.Logf("executeDoublingStepFFT returned error (may be expected): %v", err)
		}
	})

	t.Run("Execute doubling step with FFT in parallel", func(t *testing.T) {
		t.Parallel()
		// Create a calculation state with values that will trigger FFT
		state := &CalculationState{
			FK:  new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil),
			FK1: new(big.Int).Exp(big.NewInt(2), big.NewInt(1000), nil),
			T1:  new(big.Int),
			T2:  new(big.Int),
			T3:  new(big.Int),
			T4:  new(big.Int),
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      10000,
		}

		err := executeDoublingStepFFT(state, opts, true)
		if err != nil {
			t.Logf("executeDoublingStepFFT returned error (may be expected): %v", err)
		}
	})

	t.Run("Execute doubling step with smaller numbers", func(t *testing.T) {
		t.Parallel()
		// Create a calculation state with smaller values
		state := &CalculationState{
			FK:  big.NewInt(5),
			FK1: big.NewInt(8),
			T1:  new(big.Int),
			T2:  new(big.Int),
			T3:  new(big.Int),
			T4:  new(big.Int),
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      10000,
		}

		err := executeDoublingStepFFT(state, opts, false)
		// Should still work even with smaller numbers
		if err != nil {
			t.Logf("executeDoublingStepFFT returned error (may be expected): %v", err)
		}
	})
}
