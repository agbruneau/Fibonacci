package memory

import (
	"fmt"
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
	bytesPerFib := uint64(wordsPerFib) * 8

	stateBytes := bytesPerFib * 5 // 5 big.Int in CalculationState
	fftBytes := bytesPerFib * 3   // bump allocator estimate
	cacheBytes := bytesPerFib * 2 // transform cache estimate
	overheadBytes := stateBytes   // GC + runtime ~1x

	total := stateBytes + fftBytes + cacheBytes + overheadBytes
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
