package bigfft

import (
	"math/big"
	"runtime"
	"testing"
)

// drainPools performs enough GC cycles to clear sync.Pool contents,
// ensuring a clean starting state for benchmarks.
func drainPools() {
	runtime.GC()
	runtime.GC()
}

// benchmarkFFTMul runs a series of FFT multiplications at the given operand
// bit size and reports allocations. This exercises the pool acquire/release
// path that pool warming is designed to speed up.
func benchmarkFFTMul(b *testing.B, bits int64) {
	b.Helper()
	x := new(big.Int).Exp(big.NewInt(2), big.NewInt(bits), nil)
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(bits), nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		z := new(big.Int)
		_, _ = MulTo(z, x, y)
	}
}

// BenchmarkPoolWithWarming measures FFT multiplication performance when pools
// have been pre-warmed with appropriately-sized buffers. The pre-warming step
// is included in setup (not timed) so the benchmark reflects steady-state
// performance after warming.
func BenchmarkPoolWithWarming(b *testing.B) {
	cases := []struct {
		name string
		n    uint64 // Fibonacci index used for pool warming estimation
		bits int64  // Operand bit size for the multiplication
	}{
		{"N=100k", 100_000, 100_000},
		{"N=500k", 500_000, 500_000},
		{"N=1M", 1_000_000, 1_000_000},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			drainPools()
			PreWarmPools(tc.n)
			benchmarkFFTMul(b, tc.bits)
		})
	}
}

// BenchmarkPoolWithoutWarming measures FFT multiplication performance with
// cold (empty) pools. Comparing this against BenchmarkPoolWithWarming shows
// the allocation reduction that pre-warming provides.
func BenchmarkPoolWithoutWarming(b *testing.B) {
	cases := []struct {
		name string
		bits int64
	}{
		{"N=100k", 100_000},
		{"N=500k", 500_000},
		{"N=1M", 1_000_000},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			drainPools()
			benchmarkFFTMul(b, tc.bits)
		})
	}
}
