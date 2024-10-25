package main

import (
	"fmt"
	"math/big"
	"math/bits"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// FibCalculator encapsule les variables big.Int réutilisables
type FibCalculator struct {
	a, b, c, temp *big.Int
	mutex         sync.Mutex
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

// Calculate calcule le n-ième nombre de Fibonacci de manière thread-safe
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	if n > 100000000 {
		return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux")
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Réinitialisation des valeurs
	fc.a.SetInt64(0)
	fc.b.SetInt64(1)

	if n < 2 {
		return big.NewInt(int64(n)), nil
	}

	// Algorithme principal
	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		// c = a*(2*b - a)
		fc.c.Lsh(fc.b, 1)    // c = 2*b
		fc.c.Sub(fc.c, fc.a) // c = 2*b - a
		fc.c.Mul(fc.c, fc.a) // c = a*(2*b - a)

		// Sauvegarde temporaire de b
		fc.temp.Set(fc.b)

		// b = a² + b²
		fc.b.Mul(fc.b, fc.b) // b = b²
		fc.a.Mul(fc.a, fc.a) // a = a²
		fc.b.Add(fc.b, fc.a) // b = a² + b²

		if ((n >> i) & 1) == 0 {
			fc.a.Set(fc.c)
			fc.b.Set(fc.b)
		} else {
			fc.a.Set(fc.b)
			fc.b.Add(fc.c, fc.b)
		}
	}

	return new(big.Int).Set(fc.a), nil
}

// WorkerPool gère un pool de FibCalculator pour le calcul parallèle
type WorkerPool struct {
	calculators []*FibCalculator
	current     int
	mutex       sync.Mutex
}

// NewWorkerPool crée un nouveau pool de calculateurs
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
	}
}

// GetCalculator retourne un calculateur du pool de manière thread-safe
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, pool *WorkerPool, partialResult chan<- *big.Int, wg *sync.WaitGroup) {
	defer wg.Done()

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

// formatBigIntSci formate un big.Int en notation scientifique
func formatBigIntSci(n *big.Int) string {
	// Convertir en string
	numStr := n.String()
	numLen := len(numStr)

	if numLen <= 5 {
		return numStr
	}

	// Prendre les 5 premiers chiffres et calculer l'exposant
	significand := numStr[:5]
	exponent := numLen - 1 // -1 car on déplace la virgule après le premier chiffre

	// Insérer un point décimal après le premier chiffre
	formattedNum := significand[:1] + "." + significand[1:]

	// Supprimer les zéros à la fin de la partie décimale
	formattedNum = strings.TrimRight(strings.TrimRight(formattedNum, "0"), ".")

	return fmt.Sprintf("%se%d", formattedNum, exponent)
}
func main() {
	n := 100 // 100000000
	n = n - 1
	numWorkers := runtime.NumCPU()
	segmentSize := n / numWorkers
	remaining := n % numWorkers

	pool := NewWorkerPool(numWorkers)
	partialResult := make(chan *big.Int, numWorkers)
	var wg sync.WaitGroup

	startTime := time.Now()

	// Démarre les travailleurs
	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 {
			end += remaining
		}

		wg.Add(1)
		go calcFibonacci(start, end, pool, partialResult, &wg)
	}

	go func() {
		wg.Wait()
		close(partialResult)
	}()

	sumFib := new(big.Int)
	numCalculations := 0

	for partial := range partialResult {
		sumFib.Add(sumFib, partial)
		numCalculations++
	}

	executionTime := time.Since(startTime)
	avgTimePerCalculation := executionTime / time.Duration(numCalculations)

	// Écriture des résultats dans un fichier
	file, err := os.Create("fibonacci_result.txt")
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close()

	// Correction AGB
	n = n + 1
	sumFib.Add(sumFib, big.NewInt(1))

	// Écriture simplifiée et corrigée dans le fichier
	writeLines := []string{
		fmt.Sprintf("Nombre de calculs: %d", numCalculations),
		fmt.Sprintf("Temps moyen par calcul: %s", avgTimePerCalculation),
		fmt.Sprintf("Temps d'exécution: %s", executionTime),
		// fmt.Sprintf("Somme des Fib(%d) = %s", n, sumFib.String()),
		fmt.Sprintf("Somme des Fib(%d) = %s\n", n, formatBigIntSci(sumFib)),
	}

	for _, line := range writeLines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			fmt.Printf("Erreur lors de l'écriture dans le fichier: %v\n", err)
			return
		}
	}

	// Affichage console
	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Nombre de calculs: %d\n", numCalculations)
	fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	fmt.Println("Résultats écrits dans 'fibonacci_result.txt'")
}
