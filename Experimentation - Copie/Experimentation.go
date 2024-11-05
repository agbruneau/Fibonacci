package main

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"time"
)

// FibCalculator encapsule les variables big.Int réutilisables
type FibCalculator struct {
	// Pour stocker F(k) et F(k+1)
	fk, fk1 *big.Int
	// Variables temporaires pour les calculs
	temp1, temp2, temp3 *big.Int
	mutex               sync.Mutex
}

// NewFibCalculator crée une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		fk:    new(big.Int),
		fk1:   new(big.Int),
		temp1: new(big.Int),
		temp2: new(big.Int),
		temp3: new(big.Int),
	}
}

// Calculate calcule le n-ième nombre de Fibonacci avec la méthode Doubling correcte
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

	// Initialisation
	fc.fk.SetInt64(0)  // F(0)
	fc.fk1.SetInt64(1) // F(1)

	// Algorithme de doublement correct
	for i := 63; i >= 0; i-- { // Parcours des bits de n
		// Formules:
		// F(2k) = F(k)[2F(k+1) - F(k)]
		// F(2k+1) = F(k+1)^2 + F(k)^2

		// Sauvegarde de F(k) et F(k+1)
		fc.temp1.Set(fc.fk)  // temp1 = F(k)
		fc.temp2.Set(fc.fk1) // temp2 = F(k+1)

		// Calcul de F(2k)
		fc.temp3.Mul(fc.temp2, big.NewInt(2)) // temp3 = 2F(k+1)
		fc.temp3.Sub(fc.temp3, fc.temp1)      // temp3 = 2F(k+1) - F(k)
		fc.fk.Mul(fc.temp1, fc.temp3)         // F(k) = F(k)[2F(k+1) - F(k)]

		// Calcul de F(2k+1)
		fc.fk1.Mul(fc.temp2, fc.temp2)   // F(k+1) = F(k+1)^2
		fc.temp3.Mul(fc.temp1, fc.temp1) // temp3 = F(k)^2
		fc.fk1.Add(fc.fk1, fc.temp3)     // F(k+1) = F(k+1)^2 + F(k)^2

		// Si le bit correspondant de n est 1, on décale
		if (n & (1 << uint(i))) != 0 {
			fc.temp3.Set(fc.fk1)      // temp3 = F(2k+1)
			fc.fk1.Add(fc.fk1, fc.fk) // F(k+1) = F(2k+1) + F(2k)
			fc.fk.Set(fc.temp3)       // F(k) = F(2k+1)
		}
	}

	return new(big.Int).Set(fc.fk), nil
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

// Fonctions utilitaires inchangées
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
	m := 100000
	n := m - 1
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
	fmt.Printf("Somme des Fibonacci (%d): %s\n", m, formatBigIntSci(sumFib))
}
