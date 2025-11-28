package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

// TestStrassenConfiguration verifies that the StrassenThreshold option can be passed
// via the Options struct. This test primarily ensures that the API change is correctly
// reflected in the exposed types and that the configuration is accepted.
// It also verifies that the calculation remains correct with custom thresholds.
func TestStrassenConfiguration(t *testing.T) {
	// 1. Verify that Options struct has the new field.
	// This part is implicit: if the code compiles, the field exists.
	opts := Options{
		ParallelThreshold: 4096,
		FFTThreshold:      1000000,
		StrassenThreshold: 1024, // Custom threshold
	}

	// 2. Run a calculation with the custom option.
	// We use a small n, so Strassen vs Classic doesn't matter for correctness,
	// but we want to ensure it doesn't crash or error out.
	calc := &MatrixExponentiation{}
	reporter := func(p float64) {}

	// F(10) = 55
	n := uint64(10)
	expected := big.NewInt(55)

	result, err := calc.CalculateCore(context.Background(), reporter, n, opts)
	if err != nil {
		t.Fatalf("CalculateCore failed with custom StrassenThreshold: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("Expected F(%d) = %s, got %s", n, expected, result)
	}
}

// TestStrassenThresholdEffect attempts to verify that different thresholds are respected.
// Since we cannot easily introspect the internal execution path, we run with extreme values.
// - Threshold = 0 (Force Default -> 256)
// - Threshold = Huge (Force Classic)
// Correctness should be maintained in both cases.
func TestStrassenThresholdEffect(t *testing.T) {
	calc := &MatrixExponentiation{}
	reporter := func(p float64) {}
	n := uint64(100) // F(100) is large enough to potentially trigger different paths if thresholds are low

	// Case 1: High threshold (should use Classic)
	optsClassic := Options{StrassenThreshold: 1000000}
	res1, err := calc.CalculateCore(context.Background(), reporter, n, optsClassic)
	if err != nil {
		t.Fatalf("Classic path failed: %v", err)
	}

	// Case 2: Low threshold (should use Strassen if implemented correctly)
	// Note: 2x2 matrix elements for F(100) are ~70 bits.
	// Default Strassen is 256 bits.
	// If we set threshold to 10 bits, it should force Strassen.
	optsStrassen := Options{StrassenThreshold: 10}
	res2, err := calc.CalculateCore(context.Background(), reporter, n, optsStrassen)
	if err != nil {
		t.Fatalf("Strassen path failed: %v", err)
	}

	if res1.Cmp(res2) != 0 {
		t.Errorf("Results mismatch between Classic and Strassen paths")
	}
}
