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

// TestParseMemoryLimit_OverflowSaturates verifies that a value*multiplier
// product exceeding uint64 range saturates to math.MaxUint64 instead of
// silently wrapping to a small value, which would defeat the CanCalculate
// memory guard (memLimitBytes == 0 means "no limit").
func TestParseMemoryLimit_OverflowSaturates(t *testing.T) {
	t.Parallel()

	// 2^54 * 1024 (K multiplier) = 2^64, which wraps to 0 under raw uint64
	// multiplication.
	got, err := ParseMemoryLimit("18014398509481984K")
	if err != nil {
		t.Fatalf("ParseMemoryLimit() error: %v", err)
	}
	if got != math.MaxUint64 {
		t.Errorf("ParseMemoryLimit() = %d, want saturated %d (must not wrap to a small value)", got, uint64(math.MaxUint64))
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

// TestFormatMemoryEstimate pins the exact rendering (field order and units)
// so a formatting regression cannot slip through unnoticed.
func TestFormatMemoryEstimate(t *testing.T) {
	t.Parallel()

	est := MemoryEstimate{
		StateBytes:     2 * 1024 * 1024 * 1024, // GB branch
		FFTBufferBytes: 3 * 1024 * 1024,        // MB branch
		CacheBytes:     512 * 1024,             // KB branch
		OverheadBytes:  100,                    // B branch
		TotalBytes:     1536,                   // fractional KB
	}
	got := FormatMemoryEstimate(est)
	want := "State: 2.0 GB, FFT: 3.0 MB, Cache: 512.0 KB, Overhead: 100 B, Total: 1.5 KB"
	if got != want {
		t.Errorf("FormatMemoryEstimate() = %q, want %q", got, want)
	}
}

func TestFormatBytesInternal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		b    uint64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		// Just below the GB threshold must stay in MB (rounds to 1024.0).
		{1024*1024*1024 - 1, "1024.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{5 * 1024 * 1024 * 1024, "5.0 GB"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d", tc.b), func(t *testing.T) {
			t.Parallel()
			if got := formatBytesInternal(tc.b); got != tc.want {
				t.Errorf("formatBytesInternal(%d) = %q, want %q", tc.b, got, tc.want)
			}
		})
	}
}

// TestSatMul covers the zero and overflow branches, which EstimateMemoryUsage
// never reaches (its operands are always non-zero and below the saturation
// point for the multipliers it uses). An unguarded a*b would wrap 2^32 * 2^32
// to 0 — exactly the small-value wrap satMul exists to prevent.
func TestSatMul(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b uint64
		want uint64
	}{
		{"zero_left", 0, 5, 0},
		{"zero_right", 5, 0, 0},
		{"normal", 3, 4, 12},
		{"max_times_one_no_saturation", math.MaxUint64, 1, math.MaxUint64},
		{"saturates_max_times_two", math.MaxUint64, 2, math.MaxUint64},
		{"saturates_wrap_to_zero_case", 1 << 32, 1 << 32, math.MaxUint64},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := satMul(tc.a, tc.b); got != tc.want {
				t.Errorf("satMul(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// TestEstimateMemoryUsageEnvelope pins audit H-03. The estimate is a safety
// bound for --memory-limit, so the property that matters is directional: it
// must never fall below what a calculation really costs, or the flag validates
// runs that go on to exhaust the machine. The old model did exactly that,
// under-estimating by 5x to 12x at every measured point.
//
// The measured figures are MemStats.Sys deltas for the worst of the three
// algorithms at each n, one process per point
// (docs/audits/mem-baseline-2026-09.txt). They are machine-specific, so the
// upper bound is deliberately loose; the lower bound is the real assertion.
func TestEstimateMemoryUsageEnvelope(t *testing.T) {
	t.Parallel()

	const mb = 1 << 20
	// worstMeasuredMB is the largest Sys delta observed across fast / fft /
	// matrix / all at that index.
	tests := []struct {
		n               uint64
		worstMeasuredMB uint64
	}{
		{100_000, 6},
		{1_000_000, 23},
		{5_000_000, 34},
		{10_000_000, 141},
		{50_000_000, 536},
		{100_000_000, 617},
	}

	for _, tt := range tests {
		est := EstimateMemoryUsage(tt.n).TotalBytes
		measured := tt.worstMeasuredMB * mb

		if est < measured {
			t.Errorf("n=%d: estimate %d MB is BELOW the measured %d MB; --memory-limit would validate a run it cannot cover",
				tt.n, est/mb, tt.worstMeasuredMB)
		}
		// 4x leaves room for a slower machine or a different Go version while
		// still failing if a future edit inflates the model out of usefulness.
		if est > 4*measured {
			t.Errorf("n=%d: estimate %d MB is more than 4x the measured %d MB; the bound is too coarse to be useful",
				tt.n, est/mb, tt.worstMeasuredMB)
		}
	}
}

// TestEstimateMemoryUsageMonotonic guards the shape of the model: a larger
// index can never be cheaper, and the components must sum to the total.
func TestEstimateMemoryUsageMonotonic(t *testing.T) {
	t.Parallel()

	var prev uint64
	for _, n := range []uint64{0, 1, 93, 94, 1000, 100_000, 1_000_000, 10_000_000, 100_000_000} {
		est := EstimateMemoryUsage(n)
		if est.TotalBytes < prev {
			t.Errorf("n=%d: total %d is below the previous index's %d", n, est.TotalBytes, prev)
		}
		prev = est.TotalBytes

		sum := est.StateBytes + est.FFTBufferBytes + est.CacheBytes + est.OverheadBytes
		if sum != est.TotalBytes {
			t.Errorf("n=%d: components sum to %d but TotalBytes is %d", n, sum, est.TotalBytes)
		}
	}
}

// TestEstimateMemoryUsageBaselineOnlyAboveSmallN checks that the fixed floor is
// not charged to indices that never build an arena or warm a pool: at or below
// MaxFibUint64, Calculate returns through calculateSmall.
func TestEstimateMemoryUsageBaselineOnlyAboveSmallN(t *testing.T) {
	t.Parallel()

	if got := EstimateMemoryUsage(baselineMinN).TotalBytes; got >= baselineBytes {
		t.Errorf("n=%d takes the small-N path and must not be charged the %d-byte floor, got %d",
			baselineMinN, baselineBytes, got)
	}
	if got := EstimateMemoryUsage(baselineMinN + 1).TotalBytes; got < baselineBytes {
		t.Errorf("n=%d engages the FFT machinery and must carry the floor, got %d", baselineMinN+1, got)
	}
}
