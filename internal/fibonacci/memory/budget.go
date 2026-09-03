package memory

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MemoryEstimate holds the estimated memory usage for a calculation.
type MemoryEstimate struct {
	StateBytes     uint64 // big.Int state (5 temporaries)
	FFTBufferBytes uint64 // bump allocator + FFT buffers
	CacheBytes     uint64 // transform cache
	OverheadBytes  uint64 // GC + runtime overhead
	TotalBytes     uint64
}

// Multipliers over the byte size of F(n). They are an empirical envelope, not
// a derivation: see EstimateMemoryUsage for the measurements they are fitted
// to and for why a closed form is out of reach here.
const (
	// stateMultiplier covers, for the three calculators the default --algo all
	// runs concurrently: the CalculationArena (10 words per word of F(n) since
	// ADR-0009 R4), the five big.Int slots carved from it, and the deep copy
	// ReleaseStateWithResult makes of the result.
	stateMultiplier = 36
	// fftBufferMultiplier covers the bump scratch (EstimateBumpCapacity sizes
	// it at roughly 4x the operand) plus the PolValues alive across one
	// doubling step: two forward transforms, three products, three inverses,
	// each about twice its operand.
	fftBufferMultiplier = 39
	// cacheMultiplier matches fibonacci.FFTCacheMaxBytesFactor, the MaxBytes
	// bound configureFFTCache installs on the transform cache (audit M-08).
	// Before that bound the cache was capped in entries only, so this line of
	// the estimate was unrelated to what the cache could actually hold.
	cacheMultiplier = 48
	// overheadMultiplier is dominated by sync.Pool pre-warming, which is what
	// the old estimate missed entirely: PreWarmPools puts up to six buffers in
	// each of four pools, and every buffer is rounded UP to the next
	// power-of-four size class, so it can reach four times the size actually
	// requested. GC headroom and runtime structures are in here too.
	//
	// The four multipliers are a DESCRIPTIVE split of a single calibrated
	// total: it is the sum that is fitted to the measurements below, not each
	// term independently. Moving weight between them (as the M-08 cache bound
	// did, 4 -> 48) must keep the sum constant or the envelope test fails.
	overheadMultiplier = 57

	// baselineBytes is the floor that applies as soon as the FFT machinery is
	// engaged at all. It is not proportional to n: the smallest pool size
	// classes, the pre-warmed buffer counts and the Go runtime's own
	// structures cost the same at F(100k) as at F(1M). Without it the
	// F(1M) --algo all point (23 MB measured) came out at 22 MB, i.e. under.
	baselineBytes = 10 << 20

	// baselineMinN mirrors fibonacci.MaxFibUint64. Below it Calculate returns
	// through calculateSmall without warming a pool or building an arena, so
	// none of the fixed cost above is paid. The constant is duplicated rather
	// than imported because internal/fibonacci imports this package.
	baselineMinN = 93
)

// EstimateMemoryUsage estimates the memory needed to compute F(n).
//
// The estimate is a SAFETY BOUND, deliberately biased high: its only consumers
// are --memory-limit (config.ValidateMemoryBudget) and the calculator-boundary
// guard (fibonacci.CanCalculate), and both exist to refuse work that would not
// fit. Under-estimating defeats the whole point; over-estimating refuses a
// calculation that would have fitted.
//
// It used to under-estimate by 5x to 12x (audit H-03). The old model counted
// five big.Int for the state, three for FFT buffers and two for the cache — it
// modeled neither the x10 arena over-sizing, nor sync.Pool pre-warming (the
// largest single term), nor the fact that --algo all runs three calculators at
// once. Measured MemStats.Sys deltas against the old estimate:
//
//	n      old est   fast    fft   matrix    all
//	100k     0 MB     0       -       -        6
//	1M       1 MB     9      18      13       23
//	10M     12 MB    62      67     141      101
//	50M     62 MB   536       -       -        -
//	100M   124 MB   617     460       -        -
//
// A closed form is out of reach from this package: the dominant term is
// bigfft's pool pre-warming, whose power-of-four size classes make the true
// cost a step function that can jump 4x, and internal/bigfft cannot be
// imported here (internal/config imports this package, and the architecture
// gate forbids config -> bigfft). So the multipliers above are fitted to the
// worst algorithm at each measured n, which leaves the estimate between 1.0x
// and 2.5x the measured figure across the whole range — never under. The
// 2.5x appears at F(100M), which sits just past a size-class step.
//
// See docs/audits/mem-baseline-2026-09.txt for the raw measurements.
func EstimateMemoryUsage(n uint64) MemoryEstimate {
	bitsPerFib := float64(n) * 0.69424
	// bitsPerFib is non-negative (n is uint64) and at most ~1.3e19/64, so the
	// uint64 conversion is exact and cannot overflow — no intermediate int.
	wordsPerFib := uint64(bitsPerFib/64) + 1
	bytesPerFib := satMul(wordsPerFib, 8)

	// Saturating arithmetic: a silent uint64 wrap of total toward a small
	// value would let CanCalculate pass for a physically uncomputable n.
	// Defense-in-depth — for realistic n the result is bit-identical.
	stateBytes := satMul(bytesPerFib, stateMultiplier)
	fftBytes := satMul(bytesPerFib, fftBufferMultiplier)
	cacheBytes := satMul(bytesPerFib, cacheMultiplier)
	overheadBytes := satMul(bytesPerFib, overheadMultiplier)
	if n > baselineMinN {
		overheadBytes = satAdd(overheadBytes, baselineBytes)
	}

	total := satAdd(satAdd(satAdd(stateBytes, fftBytes), cacheBytes), overheadBytes)
	return MemoryEstimate{
		StateBytes:     stateBytes,
		FFTBufferBytes: fftBytes,
		CacheBytes:     cacheBytes,
		OverheadBytes:  overheadBytes,
		TotalBytes:     total,
	}
}

// ParseMemoryLimit parses a human-readable memory limit (e.g., "8G", "512M").
func ParseMemoryLimit(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty memory limit")
	}

	multiplier := uint64(1)
	suffix := s[len(s)-1]
	switch suffix {
	case 'K', 'k':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory limit %q: %w", s, err)
	}

	return satMul(val, multiplier), nil
}

// FormatMemoryEstimate returns a human-readable string of the estimate.
func FormatMemoryEstimate(est MemoryEstimate) string {
	return fmt.Sprintf("State: %s, FFT: %s, Cache: %s, Overhead: %s, Total: %s",
		formatBytesInternal(est.StateBytes),
		formatBytesInternal(est.FFTBufferBytes),
		formatBytesInternal(est.CacheBytes),
		formatBytesInternal(est.OverheadBytes),
		formatBytesInternal(est.TotalBytes))
}

func formatBytesInternal(b uint64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(b)/(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// satMul multiplies a and b, saturating to math.MaxUint64 on overflow instead
// of wrapping. The wrap would otherwise turn a huge estimate into a small one.
func satMul(a, b uint64) uint64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > math.MaxUint64/b {
		return math.MaxUint64
	}
	return a * b
}

// satAdd adds a and b, saturating to math.MaxUint64 on overflow instead of
// wrapping.
func satAdd(a, b uint64) uint64 {
	if a > math.MaxUint64-b {
		return math.MaxUint64
	}
	return a + b
}
