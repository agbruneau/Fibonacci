package main

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"time"
)

// FibCalculator encapsule les variables big.Int réutilisables avec la méthode Doubling
type FibCalculator struct {
	// f[k+1] f[k]
	// f[k]   f[k-1]
	a, b, c, d   *big.Int
	temp1, temp2 *big.Int
	mutex        sync.Mutex
}

// NewFibCalculator crée une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		a:     big.NewInt(1), // f[k+1]
		b:     big.NewInt(1), // f[k]
		c:     big.NewInt(1), // f[k]
		d:     big.NewInt(0), // f[k-1]
		temp1: new(big.Int),
		temp2: new(big.Int),
	}
}

// Calculate calcule le n-ième nombre de Fibonacci avec la méthode Doubling
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	if n > 1000000 {
		return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux")
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Cas de base
	if n <= 1 {
		return big.NewInt(int64(n)), nil
	}

	// Réinitialisation des matrices
	fc.a.SetInt64(1) // f[1]
	fc.b.SetInt64(1) // f[1]
	fc.c.SetInt64(1) // f[1]
	fc.d.SetInt64(0) // f[0]

	// Algorithme de doublement
	k := n
	for k > 0 {
		if k%2 == 1 {
			// Multiplication des matrices
			// [a b] = [a b] × [1 1]
			// [c d]   [c d]   [1 0]
			fc.temp1.Set(fc.a)
			fc.temp2.Set(fc.c)

			fc.a.Mul(fc.a, fc.a).Add(fc.a, fc.temp1.Mul(fc.temp1, fc.b))
			fc.b.Mul(fc.b, fc.temp2).Add(fc.b, fc.temp1)
			fc.c.Mul(fc.c, fc.a).Add(fc.c, fc.temp2.Mul(fc.temp2, fc.b))
			fc.d.Mul(fc.d, fc.temp2).Add(fc.d, fc.temp1)
		}

		k /= 2

		if k > 0 {
			// Carré de la matrice
			// [a b]² = [a² + b² ab]
			// [c d]    [ac + bd cd]
			fc.temp1.Mul(fc.a, fc.b)
			fc.temp2.Mul(fc.c, fc.d)

			fc.b.Mul(fc.b, fc.a.Add(fc.a, fc.c))
			fc.c.Set(fc.temp1)
			fc.d.Set(fc.temp2)
		}
	}

	return new(big.Int).Set(fc.b), nil
}

// WorkerPool reste inchangé
type WorkerPool struct {
	calculators []*FibCalculator
	current     int
	mutex       sync.Mutex
}

func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
	}
}

func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// Le reste des fonctions utilitaires reste identique
func calcFibonacci(start, end int, pool *WorkerPool, partialResult chan<- *big.Int) {
	calc := pool.GetCalculator()
	partialSum := new(big.Int)

	for i := start; i <= end; i++ {
		fibValue, err := calc.Calculate(i)
		if err != nil {
			fmt.Printf("Erreur lors du calcul de Fib(%d): %v\n", i, err)
			continue
		}
		partialSum.Add(partialSum, fibValue)
	}

	partialResult <- partialSum
}

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
	n := 100000
	n = n - 1
	numWorkers := runtime.NumCPU()
	segmentSize := n / (numWorkers * 2)
	pool := NewWorkerPool(numWorkers)
	taskChannel := make(chan [2]int, numWorkers*4)
	partialResult := make(chan *big.Int, numWorkers)
	var wg sync.WaitGroup

	for i := 0; i < n; i += segmentSize {
		end := i + segmentSize - 1
		if end >= n {
			end = n - 1
		}
		taskChannel <- [2]int{i, end}
	}
	close(taskChannel)

	startTime := time.Now()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for segment := range taskChannel {
				calcFibonacci(segment[0], segment[1], pool, partialResult)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(partialResult)
	}()

	sumFib := new(big.Int)
	count := 0

	for partial := range partialResult {
		sumFib.Add(sumFib, partial)
		count++
	}

	executionTime := time.Since(startTime)
	avgTimePerCalculation := executionTime / time.Duration(count)

	fmt.Printf("Nombre de workers: %d\n", numWorkers)
	fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Somme des Fibonacci: %s\n", formatBigIntSci(sumFib))
}
