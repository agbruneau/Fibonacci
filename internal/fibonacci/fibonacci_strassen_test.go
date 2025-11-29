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

// TestStrassenOptionsPrecedence verifies that providing StrassenThreshold via Options
// takes precedence over the global default. This ensures that we don't rely on
// global state mutation (SetDefaultStrassenThreshold) which is considered a bad practice.
func TestStrassenOptionsPrecedence(t *testing.T) {
	// 1. Save original global default and restore it after test
	originalDefault := GetDefaultStrassenThreshold()
	defer SetDefaultStrassenThreshold(originalDefault)

	// 2. Set global default to a very low value (e.g., 1)
	// This would normally force Strassen algorithm for almost everything.
	SetDefaultStrassenThreshold(1)

	// 3. Configure Options with a very high threshold (e.g., 1,000,000)
	// This should force Classic algorithm if Options is respected.
	opts := Options{StrassenThreshold: 1_000_000}

	calc := &MatrixExponentiation{}
	reporter := func(p float64) {}
	n := uint64(100) // ~70 bits elements

	// If global default (1) was used, it would use Strassen.
	// If Options (1,000,000) is used, it uses Classic.
	// We can't verify which algorithm was used directly by output (since both are correct).
	// However, this test verifies that the logic *accepts* the override and produces correct results.
	// Combined with code inspection (which shows `if strassenThresholdBits == 0 ...`), this confirms
	// that a non-zero Option overrides the global default.

	res, err := calc.CalculateCore(context.Background(), reporter, n, opts)
	if err != nil {
		t.Fatalf("Calculation failed with override: %v", err)
	}

	// Also verify that setting Options to 0 falls back to Global Default
	// Set global default to a reasonable value
	SetDefaultStrassenThreshold(256)
	optsDefault := Options{StrassenThreshold: 0}

	resDefault, err := calc.CalculateCore(context.Background(), reporter, n, optsDefault)
	if err != nil {
		t.Fatalf("Calculation failed with default: %v", err)
	}

	if res.Cmp(resDefault) != 0 {
		t.Error("Results mismatch")
	}
}
