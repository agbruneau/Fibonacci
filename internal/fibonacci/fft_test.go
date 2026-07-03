package fibonacci

import (
	"context"
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
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      10000, // Low threshold to trigger FFT
		}

		// Capture operands before the step mutates state.
		fk := new(big.Int).Set(state.FK)
		fk1 := new(big.Int).Set(state.FK1)

		err := executeDoublingStepFFT(context.Background(), state, opts, false)
		if err != nil {
			t.Errorf("executeDoublingStepFFT returned unexpected error: %v", err)
		}
		assertDoublingProducts(t, state, fk, fk1)
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
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      10000,
		}

		fk := new(big.Int).Set(state.FK)
		fk1 := new(big.Int).Set(state.FK1)

		err := executeDoublingStepFFT(context.Background(), state, opts, true)
		if err != nil {
			t.Errorf("executeDoublingStepFFT returned unexpected error: %v", err)
		}
		assertDoublingProducts(t, state, fk, fk1)
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
		}

		opts := Options{
			ParallelThreshold: 4096,
			FFTThreshold:      10000,
		}

		fk := new(big.Int).Set(state.FK)
		fk1 := new(big.Int).Set(state.FK1)

		err := executeDoublingStepFFT(context.Background(), state, opts, false)
		if err != nil {
			t.Errorf("executeDoublingStepFFT returned unexpected error: %v", err)
		}
		assertDoublingProducts(t, state, fk, fk1)
	})
}

// assertDoublingProducts verifies the three pointwise products computed by a
// doubling step against fk/fk1 captured before the call:
//
//	T3 = FK * FK1
//	T1 = FK1²
//	T2 = FK²
func assertDoublingProducts(t *testing.T, state *CalculationState, fk, fk1 *big.Int) {
	t.Helper()
	if want := new(big.Int).Mul(fk, fk1); state.T3.Cmp(want) != 0 {
		t.Errorf("T3 = %s, want FK*FK1 = %s", state.T3.String(), want.String())
	}
	if want := new(big.Int).Mul(fk1, fk1); state.T1.Cmp(want) != 0 {
		t.Errorf("T1 = %s, want FK1² = %s", state.T1.String(), want.String())
	}
	if want := new(big.Int).Mul(fk, fk); state.T2.Cmp(want) != 0 {
		t.Errorf("T2 = %s, want FK² = %s", state.T2.String(), want.String())
	}
}

// TestSmartMultiply_InPlace_BufferReuse verifies that smartMultiply reuses
// the destination buffer when it has sufficient capacity.
func TestSmartMultiply_InPlace_BufferReuse(t *testing.T) {
	t.Parallel()

	x := new(big.Int).SetInt64(123456789)
	y := new(big.Int).SetInt64(987654321)
	expected := new(big.Int).Mul(x, y)

	// Pre-allocate z with sufficient capacity
	z := new(big.Int)
	z.SetBits(make([]big.Word, 0, len(expected.Bits())+10))

	result, err := smartMultiply(z, x, y, 0)
	if err != nil {
		t.Fatalf("smartMultiply error: %v", err)
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("smartMultiply = %s, want %s", result.String(), expected.String())
	}
}

// TestSmartMultiply_NilZ_AllPaths verifies that smartMultiply handles nil z
// across both the FFT and non-FFT code paths.
func TestSmartMultiply_NilZ_AllPaths(t *testing.T) {
	t.Parallel()

	x := new(big.Int).SetInt64(123456789)
	y := new(big.Int).SetInt64(987654321)
	expected := new(big.Int).Mul(x, y)

	result, err := smartMultiply(nil, x, y, 0)
	if err != nil {
		t.Fatalf("smartMultiply error: %v", err)
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("smartMultiply = %s, want %s", result.String(), expected.String())
	}
}

// TestSmartSquare_InPlace_BufferReuse verifies that smartSquare reuses
// the destination buffer when it has sufficient capacity.
func TestSmartSquare_InPlace_BufferReuse(t *testing.T) {
	t.Parallel()

	x := new(big.Int).SetInt64(123456789)
	expected := new(big.Int).Mul(x, x)

	z := new(big.Int)
	z.SetBits(make([]big.Word, 0, len(expected.Bits())+10))

	result, err := smartSquare(z, x, 0)
	if err != nil {
		t.Fatalf("smartSquare error: %v", err)
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("smartSquare = %s, want %s", result.String(), expected.String())
	}
}
