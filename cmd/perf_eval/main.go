package main

import (
	"context"
	"fmt"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
)

func main() {
	calc := fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{})
	ctx := context.Background()

	// Test parameters
	n := uint64(10000)
	iterations := 100

	fmt.Println("=== Évaluation du Gain de Performance du Cache ===")
	fmt.Printf("Configuration: F(%d), %d itérations\n\n", n, iterations)

	// --- Test WITHOUT cache ---
	optsNoCache := fibonacci.Options{
		ParallelThreshold: fibonacci.DefaultParallelThreshold,
		FFTThreshold:      fibonacci.DefaultFFTThreshold,
		Cache:             nil,
	}

	start := time.Now()
	for i := 0; i < iterations; i++ {
		calc.Calculate(ctx, nil, 0, n, optsNoCache)
	}
	durationNoCache := time.Since(start)

	// --- Test WITH cache ---
	cache := fibonacci.NewFibonacciCache(100)
	optsWithCache := fibonacci.Options{
		ParallelThreshold: fibonacci.DefaultParallelThreshold,
		FFTThreshold:      fibonacci.DefaultFFTThreshold,
		Cache:             cache,
	}

	start = time.Now()
	for i := 0; i < iterations; i++ {
		calc.Calculate(ctx, nil, 0, n, optsWithCache)
	}
	durationWithCache := time.Since(start)

	// --- Results ---
	fmt.Println("--- Résultats ---")
	fmt.Printf("Sans cache:  %v (%.2f ms/op)\n", durationNoCache, float64(durationNoCache.Microseconds())/float64(iterations)/1000)
	fmt.Printf("Avec cache:  %v (%.2f ms/op)\n", durationWithCache, float64(durationWithCache.Microseconds())/float64(iterations)/1000)

	speedup := float64(durationNoCache) / float64(durationWithCache)
	improvement := (1 - float64(durationWithCache)/float64(durationNoCache)) * 100

	fmt.Printf("\n--- Amélioration ---\n")
	fmt.Printf("Speedup: %.1fx plus rapide\n", speedup)
	fmt.Printf("Gain de performance: %.1f%%\n", improvement)

	hits, misses, _ := cache.Stats()
	fmt.Printf("\nCache Stats: %d hits, %d misses, hit rate: %.1f%%\n", hits, misses, cache.HitRate()*100)
}
