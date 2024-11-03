// DoublingParallel-ChatGPTCanvas Optimized
// Ce programme est une version optimisée du calcul parallèle de la somme des nombres de Fibonacci.
// Il intègre des suggestions d'amélioration pour éviter le verrouillage inutile, utiliser sync.Pool pour la gestion des ressources,
// et adapter dynamiquement la taille des segments en fonction de la complexité du calcul.

package main

import (
	"fmt"
	"math/big"
	"math/bits"
	"runtime"
	"strings"
	"sync"
	"time"
)

// FibCalculator encapsule les variables big.Int réutilisables
type FibCalculator struct {
	a, b, c, temp *big.Int
}

// NewFibCalculator crée une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		a:    big.NewInt(0),
		b:    big.NewInt(1),
		c:    new(big.Int),
		temp: new(big.Int),
	}
}

// Calculate calcule le n-ième nombre de Fibonacci
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	if n > 1000000 {
		return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux")
	}

	fc.a.SetInt64(0)
	fc.b.SetInt64(1)

	if n < 2 {
		return big.NewInt(int64(n)), nil
	}

	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		fc.c.Lsh(fc.b, 1)
		fc.c.Sub(fc.c, fc.a)
		fc.c.Mul(fc.c, fc.a)

		fc.temp.Set(fc.b)

		fc.b.Mul(fc.b, fc.b)
		fc.a.Mul(fc.a, fc.a)
		fc.b.Add(fc.b, fc.a)

		if ((n >> i) & 1) == 0 {
			fc.a.Set(fc.c)
		} else {
			fc.a.Set(fc.b)
			fc.b.Add(fc.c, fc.b)
		}
	}

	return new(big.Int).Set(fc.a), nil
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, pool *sync.Pool, partialResult chan<- *big.Int) {
	calc := pool.Get().(*FibCalculator)
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
	pool.Put(calc) // Remet le calculateur dans le pool pour réutilisation
}

// formatBigIntSci formate un big.Int en notation scientifique
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
	pool := &sync.Pool{
		New: func() interface{} {
			return NewFibCalculator()
		},
	}
	taskChannel := make(chan [2]int, numWorkers*4)
	partialResult := make(chan *big.Int, numWorkers)
	var wg sync.WaitGroup

	// Déterminer dynamiquement la taille des segments en fonction de la complexité
	segmentSize := 1
	for segmentSize < n/(numWorkers*2) {
		segmentSize *= 2
	}

	// Initialiser les segments de travail et les envoyer au canal de tâches
	for i := 0; i < n; i += segmentSize {
		end := i + segmentSize - 1
		if end >= n {
			end = n - 1
		}
		taskChannel <- [2]int{i, end}
	}
	close(taskChannel)

	// Lancer les goroutines du pool pour traiter les tâches
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
