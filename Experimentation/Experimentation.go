package main

import (
	"fmt"
	"math/big"
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

// Calculate calcule le n-ième nombre de Fibonacci en utilisant l'algorithme du Fast Doubling
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	if n > 250000001 {
		return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux")
	}

	// Réinitialisation des valeurs a et b pour chaque calcul
	fc.a.SetInt64(0)
	fc.b.SetInt64(1)

	// Si n est inférieur à 2, le résultat est trivial (0 ou 1)
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}

	// Algorithme du Fast Doubling pour calculer Fib(n)
	return fc.fastDoubling(n), nil
}

// fastDoubling implémente l'algorithme du Fast Doubling
func (fc *FibCalculator) fastDoubling(n int) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}
	if n == 1 {
		return big.NewInt(1)
	}

	fc.c = fc.fastDoubling(n / 2)

	fc.temp.Mul(fc.c, fc.c) // temp = c * c

	if n%2 == 0 {
		// Fib(2k) = Fib(k) * [2 * Fib(k+1) – Fib(k)]
		fibKPlus1 := fc.fastDoubling(n/2 + 1)
		fc.a.Mul(big.NewInt(2), fibKPlus1) // a = 2 * Fib(k+1)
		fc.a.Sub(fc.a, fc.c)               // a = 2 * Fib(k+1) – Fib(k)
		fc.a.Mul(fc.a, fc.c)               // a = Fib(k) * (2 * Fib(k+1) – Fib(k))
		return fc.a
	} else {
		// Fib(2k+1) = Fib(k+1)^2 + Fib(k)^2
		fibKPlus1 := fc.fastDoubling(n/2 + 1)
		fc.a.Mul(fibKPlus1, fibKPlus1) // a = Fib(k+1)^2
		fc.b.Mul(fc.c, fc.c)           // b = Fib(k)^2
		fc.a.Add(fc.a, fc.b)           // a = Fib(k+1)^2 + Fib(k)^2
		return fc.a
	}
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, partialResult chan<- *big.Int) {
	calc := NewFibCalculator()
	partialSum := new(big.Int)
	a := big.NewInt(0)
	b := big.NewInt(1)

	// Si start > 1, calculer Fib(start - 1) et Fib(start)
	if start > 1 {
		a, _ = calc.Calculate(start - 1)
		b, _ = calc.Calculate(start)
	} else if start == 1 {
		a.SetInt64(0)
		b.SetInt64(1)
	} else {
		a.SetInt64(0)
		b.SetInt64(1)
	}

	for i := start; i <= end; i++ {
		next := new(big.Int).Add(a, b)
		partialSum.Add(partialSum, a)
		a.Set(b)
		b.Set(next)
	}

	partialResult <- partialSum
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
	formattedNum = formattedNum.TrimRight(formattedNum, "0")
	formattedNum = formattedNum.TrimRight(formattedNum, ".")

	return fmt.Sprintf("%se%d", formattedNum, exponent)
}

func main() {
	n := 250000
	numWorkers := 4 // Vous pouvez ajuster ce nombre en fonction de votre machine
	segmentSize := n / numWorkers
	partialResult := make(chan *big.Int, numWorkers)
	var workers []int

	// Diviser les tâches en segments
	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 {
			end = n - 1
		}
		workers = append(workers, start, end)
	}

	startTime := time.Now()

	// Lancer les goroutines
	for i := 0; i < numWorkers*2; i += 2 {
		go calcFibonacci(workers[i], workers[i+1], partialResult)
	}

	sumFib := new(big.Int)
	for i := 0; i < numWorkers; i++ {
		partialSum := <-partialResult
		sumFib.Add(sumFib, partialSum)
	}

	executionTime := time.Since(startTime)
	fmt.Printf("Nombre de workers: %d\n", numWorkers)
	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Somme des Fibonacci jusqu'à %d: %s\n", n, formatBigIntSci(sumFib))
}
