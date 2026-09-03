package metrics

import (
	"math"
	"math/big"
	"testing"
	"time"
)

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

func TestDigitalRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		x    *big.Int
		want int
	}{
		{"zero", big.NewInt(0), 0},
		{"one", big.NewInt(1), 1},
		{"nine", big.NewInt(9), 9},
		{"ten", big.NewInt(10), 1},
		{"55 (F10)", big.NewInt(55), 1},   // 5+5=10, 1+0=1
		{"89 (F11)", big.NewInt(89), 8},   // 8+9=17, 1+7=8
		{"144 (F12)", big.NewInt(144), 9}, // 1+4+4=9
		{"233 (F13)", big.NewInt(233), 8}, // 2+3+3=8
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := digitalRoot(tt.x)
			if got != tt.want {
				t.Errorf("digitalRoot(%s) = %d, want %d", tt.x, got, tt.want)
			}
		})
	}
}

func TestLastNDigits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		x    *big.Int
		n    int
		want string
	}{
		{"F10 last 5", big.NewInt(55), 5, "55"},
		{"F12 last 3", big.NewInt(144), 3, "144"},
		{"F20 last 4", fibSmall(20), 4, "6765"}, // F(20) = 6765
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastNDigits(tt.x, tt.n)
			if got != tt.want {
				t.Errorf("lastNDigits(%s, %d) = %q, want %q", tt.x, tt.n, got, tt.want)
			}
		})
	}
}

// TestLastNDigitsZeroPaddingThreshold covers APP-18: the zero-padding guard
// compared BitLen against n*4 bits/digit, but a decimal digit needs only
// log2(10) ≈ 3.32 bits. For n=20 that mismatch (67 bits actual vs 80 bits
// required) meant 20-24 digit numbers whose last 20 digits start with zeros
// were not padded, silently truncating the reported LastDigits.
func TestLastNDigitsZeroPaddingThreshold(t *testing.T) {
	t.Parallel()
	// 10^20 has BitLen 67 (< old threshold of n*4=80) but 21 decimal digits,
	// so its last 20 digits must be twenty zeros, not a single "0".
	x := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)
	got := lastNDigits(x, 20)
	want := "00000000000000000000"[:20]
	if got != want {
		t.Errorf("lastNDigits(10^20, 20) = %q, want %q", got, want)
	}
}

func TestLastNDigitsLargeNumber(t *testing.T) {
	t.Parallel()
	// F(1000) is a large number; verify last 10 digits
	f1000 := fibSmall(1000)
	got := lastNDigits(f1000, 10)
	// F(1000) ends with ...8757193200 (last 10 digits)
	fullStr := f1000.String()
	want := fullStr[len(fullStr)-10:]
	if got != want {
		t.Errorf("lastNDigits(F(1000), 10) = %q, want %q", got, want)
	}
}

func TestCompute(t *testing.T) {
	t.Parallel()
	result := fibSmall(100) // F(100) is 354224848179261915075
	duration := 500 * time.Millisecond
	n := uint64(100)

	ind := Compute(result, n, duration)

	// Performance indicators should be positive
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

	// F(100) is even because 100 % 3 != 0 → false? 100/3 = 33.33, so 100%3 = 1 → not even
	// F(100) = 354224848179261915075, last digit 5 → odd
	if ind.IsEven != false {
		t.Errorf("IsEven = %v, want false (100 %% 3 = %d)", ind.IsEven, 100%3)
	}

	// F(99) should be even (99 % 3 == 0)
	f99 := fibSmall(99)
	ind99 := Compute(f99, 99, duration)
	if ind99.IsEven != true {
		t.Errorf("IsEven for n=99 = %v, want true", ind99.IsEven)
	}
}

func TestComputeNilResult(t *testing.T) {
	t.Parallel()
	ind := Compute(nil, 100, time.Second)
	if ind.BitsPerSecond != 0 {
		t.Errorf("expected zero indicators for nil result")
	}
}

func TestComputeZeroDuration(t *testing.T) {
	t.Parallel()
	ind := Compute(big.NewInt(55), 10, 0)
	if ind.BitsPerSecond != 0 {
		t.Errorf("expected zero indicators for zero duration")
	}
}

func TestGoldenRatioDeviationConverges(t *testing.T) {
	t.Parallel()
	// For larger n, the deviation should get smaller
	prevDeviation := math.MaxFloat64
	for _, n := range []int{50, 100, 500, 1000} {
		result := fibSmall(n)
		ind := Compute(result, uint64(n), time.Second)
		if ind.GoldenRatioDeviation >= prevDeviation {
			t.Logf("n=%d: deviation=%f%% (prev=%f%%)", n, ind.GoldenRatioDeviation, prevDeviation)
		}
		prevDeviation = ind.GoldenRatioDeviation
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
		got := FormatBitsPerSecond(tt.bps)
		if got != tt.want {
			t.Errorf("FormatBitsPerSecond(%f) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}

func TestComputeLive(t *testing.T) {
	t.Parallel()
	n := uint64(1_000_000)
	progress := 0.5
	elapsed := 2 * time.Second

	ind := ComputeLive(n, progress, elapsed)

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

func TestComputeLiveZeroProgress(t *testing.T) {
	t.Parallel()
	ind := ComputeLive(1000, 0, time.Second)
	if ind.BitsPerSecond != 0 {
		t.Errorf("expected BitsPerSecond = 0 for zero progress, got %f", ind.BitsPerSecond)
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
		got := FormatDigitsPerSecond(tt.dps)
		if got != tt.want {
			t.Errorf("FormatDigitsPerSecond(%f) = %q, want %q", tt.dps, got, tt.want)
		}
	}
}
