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

// EstimateMemoryUsage estimates the memory needed to compute F(n).
func EstimateMemoryUsage(n uint64) MemoryEstimate {
	bitsPerFib := float64(n) * 0.69424
	wordsPerFib := int(bitsPerFib/64) + 1
	// wordsPerFib is derived from a positive uint64 (n) scaled down by 64,
	// so it is always non-negative and fits comfortably in uint64. #nosec G115
	bytesPerFib := satMul(uint64(wordsPerFib), 8)

	// Saturating arithmetic: a silent uint64 wrap of total toward a small
	// value would let CanCalculate pass for a physically uncomputable n.
	// Defense-in-depth — for realistic n the result is bit-identical.
	stateBytes := satMul(bytesPerFib, 5) // 5 big.Int in CalculationState
	fftBytes := satMul(bytesPerFib, 3)   // bump allocator estimate
	cacheBytes := satMul(bytesPerFib, 2) // transform cache estimate
	overheadBytes := stateBytes          // GC + runtime ~1x

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
	if len(s) == 0 {
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

	return val * multiplier, nil
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
