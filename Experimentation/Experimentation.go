package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Configuration holds all configurable parameters
type Configuration struct {
	M           int // Valeur maximale pour le calcul (m dans le code original)
	NumWorkers  int
	SegmentSize int
	Timeout     time.Duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() Configuration {
	return Configuration{
		M:           100000, // Valeur originale de m
		NumWorkers:  runtime.NumCPU(),
		SegmentSize: 1000,
		Timeout:     5 * time.Minute,
	}
}

// Metrics holds computation metrics
type Metrics struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalCalculations int64
	mutex             sync.Mutex
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// IncrementCalculations thread-safely increments the calculation counter
func (m *Metrics) IncrementCalculations(count int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.TotalCalculations += count
}

// FibCalculator encapsulates reusable big.Int variables
type FibCalculator struct {
	fk, fk1             *big.Int
	temp1, temp2, temp3 *big.Int
	mutex               sync.Mutex
}

// NewFibCalculator creates a new FibCalculator instance with initialized values
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		fk:    new(big.Int),
		fk1:   new(big.Int),
		temp1: new(big.Int),
		temp2: new(big.Int),
		temp3: new(big.Int),
	}
}

// Calculate computes the nth Fibonacci number using the doubling algorithm
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, errors.New("n doit être non-négatif")
	}
	if n > 500001 {
		return nil, errors.New("n est trop grand, risque de calculs extrêmement coûteux")
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if n <= 1 {
		return big.NewInt(int64(n)), nil
	}

	fc.fk.SetInt64(0)
	fc.fk1.SetInt64(1)

	for i := 63; i >= 0; i-- {
		fc.temp1.Set(fc.fk)
		fc.temp2.Set(fc.fk1)

		// F(2k) calculation
		fc.temp3.Mul(fc.temp2, big.NewInt(2))
		fc.temp3.Sub(fc.temp3, fc.temp1)
		fc.fk.Mul(fc.temp1, fc.temp3)

		// F(2k+1) calculation
		fc.fk1.Mul(fc.temp2, fc.temp2)
		fc.temp3.Mul(fc.temp1, fc.temp1)
		fc.fk1.Add(fc.fk1, fc.temp3)

		if (n & (1 << uint(i))) != 0 {
			fc.temp3.Set(fc.fk1)
			fc.fk1.Add(fc.fk1, fc.fk)
			fc.fk.Set(fc.temp3)
		}
	}

	return new(big.Int).Set(fc.fk), nil
}

// WorkerPool manages a pool of FibCalculators
type WorkerPool struct {
	calculators []*FibCalculator
	current     int
	mutex       sync.Mutex
}

// NewWorkerPool creates a new WorkerPool with the specified size
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
	}
}

// GetCalculator returns the next available calculator from the pool
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// Result represents a computation result with potential error
type Result struct {
	Value *big.Int
	Error error
}

// computeSegment calculates Fibonacci numbers for a segment
func computeSegment(ctx context.Context, start, end int, pool *WorkerPool, metrics *Metrics) Result {
	calc := pool.GetCalculator()
	partialSum := new(big.Int)
	segmentSize := end - start + 1

	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done():
			return Result{Error: ctx.Err()}
		default:
			fibValue, err := calc.Calculate(i)
			if err != nil {
				return Result{Error: errors.Wrapf(err, "computing Fibonacci(%d)", i)}
			}
			partialSum.Add(partialSum, fibValue)
		}
	}

	metrics.IncrementCalculations(int64(segmentSize))
	return Result{Value: partialSum}
}

// formatBigIntSci formats a big.Int in scientific notation
func formatBigIntSci(n *big.Int) string {
	numStr := n.String()
	numLen := len(numStr)

	if numLen <= 5 {
		return numStr
	}

	significand := numStr[:5]
	exponent := numLen - 1

	formattedNum := significand[:1] + "." + significand[1:]
	formattedNum = strings.TrimRight(strings.TrimRight(formattedNum, "0"), ".")

	return fmt.Sprintf("%se%d", formattedNum, exponent)
}

func main() {
	config := DefaultConfig()
	metrics := NewMetrics()

	// Calcul de n à partir de m comme dans le code original
	n := config.M - 1

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	pool := NewWorkerPool(config.NumWorkers)
	results := make(chan Result, config.NumWorkers)
	var wg sync.WaitGroup

	// Distribute work
	for start := 0; start < n; start += config.SegmentSize {
		end := start + config.SegmentSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			result := computeSegment(ctx, start, end, pool, metrics)
			results <- result
		}(start, end)
	}

	// Wait for completion in separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	sumFib := new(big.Int)
	hasErrors := false

	for result := range results {
		if result.Error != nil {
			log.Printf("Erreur durant le calcul: %v", result.Error)
			hasErrors = true
			continue
		}
		sumFib.Add(sumFib, result.Value)
	}

	if hasErrors {
		log.Printf("Des erreurs sont survenues pendant le calcul")
	}

	metrics.EndTime = time.Now()
	duration := metrics.EndTime.Sub(metrics.StartTime)
	avgTime := duration / time.Duration(metrics.TotalCalculations)

	// Output results
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Nombre de workers: %d\n", config.NumWorkers)
	fmt.Printf("  Taille des segments: %d\n", config.SegmentSize)
	fmt.Printf("  Valeur de m: %d\n", config.M)

	fmt.Printf("\nPerformance:\n")
	fmt.Printf("  Temps total d'exécution: %v\n", duration)
	fmt.Printf("  Nombre de calculs: %d\n", metrics.TotalCalculations)
	fmt.Printf("  Temps moyen par calcul: %v\n", avgTime)

	fmt.Printf("\nRésultat:\n")
	fmt.Printf("  Somme des Fibonacci(0..%d): %s\n", config.M, formatBigIntSci(sumFib))
}
