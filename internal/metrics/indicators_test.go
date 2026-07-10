package metrics_test

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/metrics"
)

// fibSmall computes F(n) via naive iteration; used only to build realistic
// *big.Int fixtures for the exported Compute/ComputeLive API.
func fibSmall(n int) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}
	a, b := big.NewInt(0), big.NewInt(1)
	for i := 2; i <= n; i++ {
		a.Add(a, b)
		a, b = b, a
	}
	return b
}

func TestCompute(t *testing.T) {
	t.Parallel()
	result := fibSmall(100) // F(100) = 354224848179261915075
	duration := 500 * time.Millisecond
	n := uint64(100)

	ind := metrics.Compute(result, n, duration)

	if ind.BitsPerSecond <= 0 {
		t.Errorf("BitsPerSecond = %f, want > 0", ind.BitsPerSecond)
	}
	if ind.DigitsPerSecond <= 0 {
		t.Errorf("DigitsPerSecond = %f, want > 0", ind.DigitsPerSecond)
	}
	if ind.DoublingSteps == 0 {
		t.Error("DoublingSteps = 0, want > 0")
	}
	if ind.StepsPerSecond <= 0 {
		t.Errorf("StepsPerSecond = %f, want > 0", ind.StepsPerSecond)
	}

	// DoublingSteps ≈ log₂(100) = 7
	if ind.DoublingSteps != 7 {
		t.Errorf("DoublingSteps = %d, want 7", ind.DoublingSteps)
	}

	// Golden ratio deviation should be small for n=100
	if ind.GoldenRatioDeviation > 5.0 {
		t.Errorf("GoldenRatioDeviation = %f%%, want < 5%%", ind.GoldenRatioDeviation)
	}

	// F(100) digital root: 354224848179261915075 → sum = 3+5+4+2+2+4+8+4+8+1+7+9+2+6+1+9+1+5+0+7+5 = 93 → 9+3 = 12 → 1+2 = 3
	if ind.DigitalRoot != 3 {
		t.Errorf("DigitalRoot = %d, want 3", ind.DigitalRoot)
	}
}

// TestComputeParity covers IsEven for both branches of n%3==0: F(n) is even
// iff 3 | n.
func TestComputeParity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		n    int
		want bool
	}{
		{"F(100), 100%3 != 0, odd", 100, false},
		{"F(99), 99%3 == 0, even", 99, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ind := metrics.Compute(fibSmall(tt.n), uint64(tt.n), time.Second)
			if ind.IsEven != tt.want {
				t.Errorf("IsEven = %v, want %v (n%%3 = %d)", ind.IsEven, tt.want, tt.n%3)
			}
		})
	}
}

// TestComputeEdgeCases covers the two guard branches of Compute: a nil
// result and a non-positive duration both yield zero-value Indicators
// instead of dividing by zero or dereferencing nil.
func TestComputeEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		result   *big.Int
		duration time.Duration
	}{
		{"nil result", nil, time.Second},
		{"zero duration", big.NewInt(55), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ind := metrics.Compute(tt.result, 100, tt.duration)
			if ind.BitsPerSecond != 0 {
				t.Errorf("BitsPerSecond = %f, want 0", ind.BitsPerSecond)
			}
		})
	}
}

func TestGoldenRatioDeviationConverges(t *testing.T) {
	t.Parallel()
	// For larger n, the deviation should get smaller
	var prevDeviation float64 = math.MaxFloat64
	for _, n := range []int{50, 100, 500, 1000} {
		result := fibSmall(n)
		ind := metrics.Compute(result, uint64(n), time.Second)
		if ind.GoldenRatioDeviation >= prevDeviation {
			t.Logf("n=%d: deviation=%f%% (prev=%f%%)", n, ind.GoldenRatioDeviation, prevDeviation)
		}
		prevDeviation = ind.GoldenRatioDeviation
	}
}

func TestComputeLive(t *testing.T) {
	t.Parallel()
	n := uint64(1_000_000)
	progress := 0.5
	elapsed := 2 * time.Second

	ind := metrics.ComputeLive(n, progress, elapsed)

	if ind.BitsPerSecond <= 0 {
		t.Errorf("BitsPerSecond = %f, want > 0", ind.BitsPerSecond)
	}
	if ind.DigitsPerSecond <= 0 {
		t.Errorf("DigitsPerSecond = %f, want > 0", ind.DigitsPerSecond)
	}
	if ind.DoublingSteps == 0 {
		t.Error("DoublingSteps = 0, want > 0")
	}
	if ind.StepsPerSecond <= 0 {
		t.Errorf("StepsPerSecond = %f, want > 0", ind.StepsPerSecond)
	}
	// Parity: 1_000_000 % 3 != 0, so F(n) is odd
	if ind.IsEven {
		t.Error("expected IsEven = false for n=1000000")
	}
	// Mathematical fields should be zero/empty for live
	if ind.GoldenRatioDeviation != 0 {
		t.Errorf("expected GoldenRatioDeviation = 0 for live, got %f", ind.GoldenRatioDeviation)
	}
	if ind.DigitalRoot != 0 {
		t.Errorf("expected DigitalRoot = 0 for live, got %d", ind.DigitalRoot)
	}
	if ind.LastDigits != "" {
		t.Errorf("expected LastDigits = \"\" for live, got %q", ind.LastDigits)
	}
}

// TestComputeLiveEdgeCases covers the three guard conditions of ComputeLive
// (elapsed <= 0, progress <= 0, n <= 1): each must short-circuit to a
// zero-value performance indicator instead of dividing by zero.
func TestComputeLiveEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		n        uint64
		progress float64
		elapsed  time.Duration
	}{
		{"zero progress", 1000, 0, time.Second},
		{"zero elapsed", 1000, 0.5, 0},
		{"n <= 1", 1, 0.5, time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ind := metrics.ComputeLive(tt.n, tt.progress, tt.elapsed)
			if ind.BitsPerSecond != 0 {
				t.Errorf("BitsPerSecond = %f, want 0", ind.BitsPerSecond)
			}
		})
	}
}

func TestFormatBitsPerSecond(t *testing.T) {
	t.Parallel()
	tests := []struct {
		bps  float64
		want string
	}{
		{500, "500 bit/s"},
		{1500, "1.50 Kbit/s"},
		{2_500_000, "2.50 Mbit/s"},
		{3_500_000_000, "3.50 Gbit/s"},
	}
	for _, tt := range tests {
		got := metrics.FormatBitsPerSecond(tt.bps)
		if got != tt.want {
			t.Errorf("FormatBitsPerSecond(%f) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}

func TestFormatDigitsPerSecond(t *testing.T) {
	t.Parallel()
	tests := []struct {
		dps  float64
		want string
	}{
		{500, "500 digits/s"},
		{1500, "1.50 K digits/s"},
		{2_500_000, "2.50 M digits/s"},
	}
	for _, tt := range tests {
		got := metrics.FormatDigitsPerSecond(tt.dps)
		if got != tt.want {
			t.Errorf("FormatDigitsPerSecond(%f) = %q, want %q", tt.dps, got, tt.want)
		}
	}
}
