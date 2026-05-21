package fibonacci

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkFibonacciDTM compares Fibonacci computation with and without the
// dynamic threshold manager active. Audit-PRD E7-R1 / Sprint S3-T1: the result
// drives the ADR-0001 decision on whether to retain DynamicThresholdManager
// or remove it as duplicative of the startup calibration.
//
// To produce the two files used by ADR-0001:
//
//	go test -bench='BenchmarkFibonacciDTM/Off' -benchmem -run='^$' \
//	    -count=5 -benchtime=1x ./internal/fibonacci/ \
//	    > docs/audits/bench-dtm-off.txt
//	go test -bench='BenchmarkFibonacciDTM/On' -benchmem -run='^$' \
//	    -count=5 -benchtime=1x ./internal/fibonacci/ \
//	    > docs/audits/bench-dtm-on.txt
//	benchstat docs/audits/bench-dtm-off.txt docs/audits/bench-dtm-on.txt
func BenchmarkFibonacciDTM(b *testing.B) {
	sizes := []struct {
		name string
		n    uint64
	}{
		{"1M", 1_000_000},
		{"10M", 10_000_000},
	}

	for _, sz := range sizes {
		// DTM off: vanilla fast-doubling path. The reference.
		b.Run(fmt.Sprintf("Off/%s", sz.name), func(b *testing.B) {
			calc := MustNewCalculator(&FastDoublingCalculator{})
			ctx := context.Background()
			opts := Options{
				ParallelThreshold:       DefaultParallelThreshold,
				FFTThreshold:            DefaultFFTThreshold,
				StrassenThreshold:       DefaultStrassenThreshold,
				EnableDynamicThresholds: false,
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = calc.Calculate(ctx, nil, 0, sz.n, opts)
			}
		})

		// DTM on: in-flight threshold adjustment. Should outperform Off by
		// > 5% to justify the ~283 LOC + invariant-A18 maintenance cost.
		b.Run(fmt.Sprintf("On/%s", sz.name), func(b *testing.B) {
			calc := MustNewCalculator(&FastDoublingCalculator{})
			ctx := context.Background()
			opts := Options{
				ParallelThreshold:       DefaultParallelThreshold,
				FFTThreshold:            DefaultFFTThreshold,
				StrassenThreshold:       DefaultStrassenThreshold,
				EnableDynamicThresholds: true,
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = calc.Calculate(ctx, nil, 0, sz.n, opts)
			}
		})
	}
}
