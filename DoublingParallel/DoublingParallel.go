package main

import (
	"errors"
	"flag"
	"fmt"
	"math/big"
	"math/bits"
	"os"
	"sync"
	"time"
)

var memo sync.Map

// fibDoublingMemo calcule le nième nombre de Fibonacci en utilisant la méthode du doublement avec mémoïsation
func fibDoublingMemo(n int) (*big.Int, error) {
	if n < 0 {
		return nil, errors.New("n doit être un entier positif")
	}
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}

	// Vérifier dans la mémoïsation
	if val, exists := memo.Load(n); exists {
		return val.(*big.Int), nil
	}

	a := big.NewInt(0)
	b := big.NewInt(1)
	c := new(big.Int)
	d := new(big.Int)

	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		// c = a * (2*b - a)
		t1 := new(big.Int).Lsh(b, 1)  // 2*b
		t2 := new(big.Int).Sub(t1, a) // 2*b - a
		c.Mul(a, t2)                  // a * (2*b - a)

		// d = a^2 + b^2
		t3 := new(big.Int).Mul(a, a) // a^2
		t4 := new(big.Int).Mul(b, b) // b^2
		d.Add(t3, t4)                // a^2 + b^2

		if ((n >> i) & 1) == 0 {
			a.Set(c)
			b.Set(d)
		} else {
			a.Set(d)
			b.Add(c, d)
		}
	}

	// Stocker dans la mémoïsation
	memo.Store(n, a)

	return a, nil
}

// calcFibonacciSegment calcule une plage de termes de Fibonacci et envoie les résultats sur un canal
func calcFibonacciSegment(start, end int, results chan<- struct {
	index int
	fib   *big.Int
}, wg *sync.WaitGroup) {
	defer wg.Done()

	for j := start; j <= end; j++ {
		fib, err := fibDoublingMemo(j)
		if err != nil {
			fmt.Printf("Erreur au calcul de F(%d): %v\n", j, err)
			continue
		}
		results <- struct {
			index int
			fib   *big.Int
		}{index: j, fib: fib}
	}
}

func main() {
	// Définir les flags de la ligne de commande
	nPtr := flag.Int("n", 100000, "Nombre de termes de Fibonacci à générer")
	workersPtr := flag.Int("workers", 4, "Nombre de goroutines à utiliser")
	outputPtr := flag.String("output", "fibonacci_list.txt", "Fichier de sortie pour la liste de Fibonacci")
	flag.Parse()

	n := *nPtr
	numWorkers := *workersPtr
	outputFile := *outputPtr

	if n < 0 {
		fmt.Println("Erreur: n doit être un entier positif.")
		return
	}

	startTime := time.Now()

	// Initialisation des structures de données
	results := make(chan struct {
		index int
		fib   *big.Int
	}, n)
	var wg sync.WaitGroup

	// Calcul des segments
	segmentSize := n / numWorkers
	remaining := n % numWorkers

	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 {
			end += remaining
		}
		if end > n {
			end = n
		}
		wg.Add(1)
		go calcFibonacciSegment(start, end, results, &wg)
	}

	// Fermeture du canal une fois tous les calculs terminés
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collecter les résultats
	fibList := make([]*big.Int, n+1)
	for res := range results {
		fibList[res.index] = res.fib
	}

	// Écrire les résultats dans le fichier
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close()

	for i, fib := range fibList {
		_, err := file.WriteString(fmt.Sprintf("F(%d) = %s\n", i, fib.String()))
		if err != nil {
			fmt.Println("Erreur lors de l'écriture dans le fichier:", err)
			return
		}
	}

	executionTime := time.Since(startTime)
	fmt.Printf("Génération des %d termes de Fibonacci terminée en %s.\n", n, executionTime)
	fmt.Printf("La liste de Fibonacci a été sauvegardée dans '%s'.\n", outputFile)
}
