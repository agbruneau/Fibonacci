package memory

import (
	"fmt"
	"math"
	"testing"
)

func TestEstimateMemoryUsage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		n        uint64
		minBytes uint64
		maxBytes uint64
	}{
		{1_000_000, 1_000_000, 50_000_000},
		{10_000_000, 10_000_000, 500_000_000},
		{1_000_000_000, 1_000_000_000, 50_000_000_000},
	}

	for _, tc := range cases {

		t.Run(fmt.Sprintf("N=%d", tc.n), func(t *testing.T) {
			t.Parallel()
			est := EstimateMemoryUsage(tc.n)
			if est.TotalBytes < tc.minBytes {
				t.Errorf("estimate %d too low, want >= %d", est.TotalBytes, tc.minBytes)
			}
			if est.TotalBytes > tc.maxBytes {
				t.Errorf("estimate %d too high, want <= %d", est.TotalBytes, tc.maxBytes)
			}
		})
	}
}

// TestEstimateMemoryUsage_SaturatesNoWrap verifies that an extreme (physically
// uncomputable) n does not wrap TotalBytes toward a small value, which would
// silently defeat the CanCalculate memory guard.
func TestEstimateMemoryUsage_SaturatesNoWrap(t *testing.T) {
	t.Parallel()

	est := EstimateMemoryUsage(math.MaxUint64)
	if est.TotalBytes < uint64(1)<<62 {
		t.Errorf("TotalBytes wrapped: got %d, want >= %d", est.TotalBytes, uint64(1)<<62)
	}
}

func TestParseMemoryLimit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  uint64
	}{
		{"8G", 8 * 1024 * 1024 * 1024},
		{"512M", 512 * 1024 * 1024},
		{"1024K", 1024 * 1024},
		{"1073741824", 1073741824},
	}

	for _, tc := range cases {

		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := ParseMemoryLimit(tc.input)
			if err != nil {
				t.Fatalf("ParseMemoryLimit(%q) error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseMemoryLimit(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseMemoryLimit_Errors(t *testing.T) {
	t.Parallel()

	cases := []string{"", "abc", "-5G", "0x10M"}
	for _, input := range cases {

		t.Run(input, func(t *testing.T) {
			t.Parallel()
			_, err := ParseMemoryLimit(input)
			if err == nil {
				t.Errorf("ParseMemoryLimit(%q) should return error", input)
			}
		})
	}
}
