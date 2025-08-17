package fibonacci_test

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/user/fibbench/internal/fibonacci"
)

// Valeurs connues de la suite de Fibonacci pour validation.
var knownFib = []struct {
	n     int
	value string
}{
	{0, "0"},
	{1, "1"},
	{10, "55"},
	{50, "12586269025"},
	{100, "354224848179261915075"},
}

// TestAlgorithmCorrectness vérifie l'exactitude de tous les algorithmes enregistrés.
func TestAlgorithmCorrectness(t *testing.T) {
	algos := fibonacci.ListAlgorithms()
	pool := fibonacci.NewIntPool()

	for _, algo := range algos {
		t.Run(string(algo.Key), func(t *testing.T) {
			for _, tt := range knownFib {
				t.Run(fmt.Sprintf("N=%d", tt.n), func(t *testing.T) {
					ctx := context.Background()
					// Test sans canal de progression.
					result, err := algo.Impl.Calculate(ctx, nil, tt.n, pool)

					if err != nil {
						t.Fatalf("Calculate(N=%d) failed: %v", tt.n, err)
					}

					expected := new(big.Int)
					expected.SetString(tt.value, 10)

					if result.Cmp(expected) != 0 {
						t.Errorf("Calculate(N=%d) got %s, want %s", tt.n, result.String(), expected.String())
					}
				})
			}
		})
	}
}

// TestNegativeN vérifie la gestion des entrées négatives.
func TestNegativeN(t *testing.T) {
	algos := fibonacci.ListAlgorithms()
	pool := fibonacci.NewIntPool()

	for _, algo := range algos {
		t.Run(string(algo.Key), func(t *testing.T) {
			_, err := algo.Impl.Calculate(context.Background(), nil, -1, pool)
			if err == nil {
				t.Error("Expected error for N=-1, got nil")
			}
		})
	}
}

// TestContextCancellation vérifie la réactivité à l'annulation du contexte.
func TestContextCancellation(t *testing.T) {
	largeN := 50000000 // Très grand N
	algos := fibonacci.ListAlgorithms()
	pool := fibonacci.NewIntPool()

	for _, algo := range algos {
		t.Run(string(algo.Key), func(t *testing.T) {
			// Timeout très court.
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			start := time.Now()
			_, err := algo.Impl.Calculate(ctx, nil, largeN, pool)
			duration := time.Since(start)

			if err != context.DeadlineExceeded {
				t.Errorf("Expected context.DeadlineExceeded, got %v", err)
			}

			// Vérification de la réactivité (doit être rapide).
			if duration > 150*time.Millisecond {
				t.Errorf("Cancellation took too long: %v", duration)
			}
		})
	}
}

// TestProgressReporting vérifie que la progression est croissante et atteint 100%.
func TestProgressReporting(t *testing.T) {
	N := 10000
	algo, _ := fibonacci.Get("iterative") // Utilisation de l'itératif pour garantir de nombreuses mises à jour.

	pool := fibonacci.NewIntPool()
	progressCh := make(chan float64, 200)

	var wg sync.WaitGroup
	wg.Add(1)

	var progressUpdates []float64
	go func() {
		defer wg.Done()
		for p := range progressCh {
			progressUpdates = append(progressUpdates, p)
		}
	}()

	_, err := algo.Impl.Calculate(context.Background(), progressCh, N, pool)
	close(progressCh)
	wg.Wait()

	if err != nil {
		t.Fatalf("Calculation failed: %v", err)
	}

	if len(progressUpdates) == 0 {
		t.Fatal("Expected progress updates, got none")
	}

	// Vérification que la progression atteint 100%.
	lastUpdate := progressUpdates[len(progressUpdates)-1]
	if lastUpdate != 100.0 {
		t.Errorf("Expected last progress update to be 100.0, got %f", lastUpdate)
	}

	// Vérification de la monotonicité.
	for i := 1; i < len(progressUpdates); i++ {
		if progressUpdates[i] < progressUpdates[i-1] {
			t.Errorf("Progress decreased at index %d: %f < %f", i, progressUpdates[i], progressUpdates[i-1])
			break
		}
	}
}
