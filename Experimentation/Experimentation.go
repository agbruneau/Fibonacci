package main

import (
	"fmt"
	"math/big"
	"math/bits"
	"os"
	"runtime"
	"sync"
	"time"
)

var memo sync.Map

// fibDoubling calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
func fibDoubling(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}
	result := fibDoublingHelperIterative(n)
	return result, nil
}

// fibDoublingHelperIterative est une fonction itérative qui utilise la méthode de doublage pour calculer les nombres de Fibonacci
func fibDoublingHelperIterative(n int) *big.Int {
	a := big.NewInt(0)
	b := big.NewInt(1)
	c := new(big.Int)

	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		c.Lsh(b, 1)
		c.Sub(c, a)
		c.Mul(c, a)
		b.Mul(a, a)
		b.Add(b, b.Mul(b, b))

		if ((n >> i) & 1) == 0 {
			a.Set(c)
			b.Set(b)
		} else {
			a.Set(b)
			b.Add(c, b)
		}
	}

	return a
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, partialResult chan<- *big.Int, wg *sync.WaitGroup) {
	defer wg.Done()

	partialSum := big.NewInt(0)
	for i := start; i <= end; i++ {
		fibValue, _ := fibDoubling(i)
		partialSum.Add(partialSum, fibValue)
	}

	partialResult <- partialSum
}

func main() {
	n := 100000000
	numWorkers := runtime.NumCPU()
	segmentSize := n / numWorkers
	remaining := n % numWorkers

	partialResult := make(chan *big.Int, numWorkers*2)
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 {
			end += remaining
		}

		wg.Add(1)
		go calcFibonacci(start, end, partialResult, &wg)
	}

	go func() {
		wg.Wait()
		close(partialResult)
	}()

	sumFib := big.NewInt(0)
	numCalculations := 0
	for partial := range partialResult {
		sumFib.Add(sumFib, partial)
		numCalculations++
	}

	executionTime := time.Since(startTime)
	avgTimePerCalculation := executionTime / time.Duration(numCalculations)

	file, err := os.Create("fibonacci_result.txt")
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("Somme des Fib(%d) = %s\n", n, sumFib.String()))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du résultat dans le fichier:", err)
		return
	}

	_, err = file.WriteString(fmt.Sprintf("Nombre de calculs: %d\n", numCalculations))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du nombre de calculs dans le fichier:", err)
		return
	}

	_, err = file.WriteString(fmt.Sprintf("Temps moyen par calcul: %s\n", avgTimePerCalculation))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du temps moyen par calcul dans le fichier:", err)
		return
	}

	_, err = file.WriteString(fmt.Sprintf("Temps d'exécution: %s\n", executionTime))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du temps d'exécution dans le fichier:", err)
		return
	}

	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Nombre de calculs: %d\n", numCalculations)
	fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	fmt.Println("Résultat, nombre de calculs, temps moyen par calcul et temps d'exécution écrits dans 'fibonacci_result.txt'.")
}
