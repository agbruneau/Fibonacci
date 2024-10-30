package main

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
)

// Matrix2x2 représente une matrice 2x2 pour le calcul de Fibonacci
type Matrix2x2 struct {
	a00, a01, a10, a11 *big.Int
}

// NewMatrix2x2 crée une nouvelle matrice 2x2
func NewMatrix2x2() *Matrix2x2 {
	return &Matrix2x2{
		a00: new(big.Int), a01: new(big.Int),
		a10: new(big.Int), a11: new(big.Int),
	}
}

// FibCalculator encapsule la logique de calcul de Fibonacci
type FibCalculator struct {
	cache     sync.Map
	matrix    *Matrix2x2
	tempMat   *Matrix2x2
	resultMat *Matrix2x2
	mutex     sync.Mutex
}

// NewFibCalculator crée une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	fc := &FibCalculator{
		matrix:    NewMatrix2x2(),
		tempMat:   NewMatrix2x2(),
		resultMat: NewMatrix2x2(),
	}
	return fc
}

// multiplyMatrix multiplie deux matrices 2x2
func (fc *FibCalculator) multiplyMatrix(m1, m2, result *Matrix2x2) {
	temp00 := new(big.Int).Mul(m1.a00, m2.a00)
	temp00.Add(temp00, new(big.Int).Mul(m1.a01, m2.a10))

	temp01 := new(big.Int).Mul(m1.a00, m2.a01)
	temp01.Add(temp01, new(big.Int).Mul(m1.a01, m2.a11))

	temp10 := new(big.Int).Mul(m1.a10, m2.a00)
	temp10.Add(temp10, new(big.Int).Mul(m1.a11, m2.a10))

	temp11 := new(big.Int).Mul(m1.a10, m2.a01)
	temp11.Add(temp11, new(big.Int).Mul(m1.a11, m2.a11))

	result.a00.Set(temp00)
	result.a01.Set(temp01)
	result.a10.Set(temp10)
	result.a11.Set(temp11)
}

// Calculate calcule le n-ième nombre de Fibonacci
func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être positif")
	}

	// Vérifier le contexte
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Vérifier le cache
	if cachedValue, ok := fc.cache.Load(n); ok {
		return new(big.Int).Set(cachedValue.(*big.Int)), nil
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Initialiser la matrice de base
	fc.matrix.a00.SetInt64(1)
	fc.matrix.a01.SetInt64(1)
	fc.matrix.a10.SetInt64(1)
	fc.matrix.a11.SetInt64(0)

	// Initialiser la matrice résultat comme matrice identité
	fc.resultMat.a00.SetInt64(1)
	fc.resultMat.a01.SetInt64(0)
	fc.resultMat.a10.SetInt64(0)
	fc.resultMat.a11.SetInt64(1)

	// Exponentiation rapide de matrice
	power := n - 1
	for power > 0 {
		if power&1 == 1 {
			fc.multiplyMatrix(fc.resultMat, fc.matrix, fc.tempMat)
			fc.resultMat, fc.tempMat = fc.tempMat, fc.resultMat
		}
		fc.multiplyMatrix(fc.matrix, fc.matrix, fc.tempMat)
		fc.matrix, fc.tempMat = fc.tempMat, fc.matrix
		power >>= 1

		// Vérifier le contexte périodiquement
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	result := new(big.Int).Set(fc.resultMat.a00)
	fc.cache.Store(n, result)
	return result, nil
}

// WorkerPool gère un pool de workers pour le calcul parallèle
type WorkerPool struct {
	calculators []*FibCalculator
	sem         *semaphore.Weighted
	current     uint64
}

// NewWorkerPool crée un nouveau pool de workers
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
		sem:         semaphore.NewWeighted(int64(size)),
	}
}

// GetCalculator obtient un calculateur du pool
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	current := atomic.AddUint64(&wp.current, 1)
	return wp.calculators[current%uint64(len(wp.calculators))]
}

// ProcessSegment traite un segment de calculs Fibonacci
func (wp *WorkerPool) ProcessSegment(ctx context.Context, start, end int, results chan<- *big.Int) error {
	if err := wp.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer wp.sem.Release(1)

	calc := wp.GetCalculator()
	sum := new(big.Int)

	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fibValue, err := calc.Calculate(ctx, i)
			if err != nil {
				return err
			}
			sum.Add(sum, fibValue)
		}
	}

	select {
	case results <- sum:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	n := 250000
	numCPU := runtime.NumCPU()
	pool := NewWorkerPool(numCPU)

	// Calculer la taille optimale des segments
	segmentSize := 1000
	if n > 10000 {
		segmentSize = n / (numCPU * 4)
	}

	results := make(chan *big.Int, numCPU)
	var wg sync.WaitGroup
	var errCount uint64

	startTime := time.Now()

	// Distribuer le travail
	for start := 0; start < n; start += segmentSize {
		end := start + segmentSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			if err := pool.ProcessSegment(ctx, s, e, results); err != nil {
				fmt.Printf("Erreur segment %d-%d: %v\n", s, e, err)
				atomic.AddUint64(&errCount, 1)
			}
		}(start, end)
	}

	// Goroutine pour fermer le canal des résultats
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collecter les résultats
	finalSum := new(big.Int)
	for partialSum := range results {
		finalSum.Add(finalSum, partialSum)
	}

	duration := time.Since(startTime)
	fmt.Printf("Temps total: %v\n", duration)
	fmt.Printf("Erreurs: %d\n", atomic.LoadUint64(&errCount))
	fmt.Printf("Résultat: %s\n", formatBigInt(finalSum))
	fmt.Printf("Performance moyenne: %v par calcul\n", duration/time.Duration(n))
}

// formatBigInt formate un grand nombre en notation scientifique
func formatBigInt(n *big.Int) string {
	str := n.String()
	length := len(str)
	if length <= 10 {
		return str
	}
	return fmt.Sprintf("%s.%se%d", str[0:1], str[1:10], length-1)
}
