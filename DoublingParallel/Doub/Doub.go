package main

import (
	"errors"
	"flag"
	"fmt"
	"math/big"
	"math/bits"
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

func main() {
	// Définir les flags de la ligne de commande
	nPtr := flag.Int("n", 100000, "Nombre de termes de Fibonacci à générer")
	flag.Parse()

	n := *nPtr

	if n < 0 {
		fmt.Println("Erreur: n doit être un entier positif.")
		return
	}

	startTime := time.Now()

	// Initialisation des structures de données
	fibList := make([]*big.Int, n+1)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Calcul des termes de Fibonacci
	for j := 0; j <= n; j++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			fib, err := fibDoublingMemo(j)
			if err != nil {
				fmt.Printf("Erreur au calcul de F(%d): %v\n", j, err)
				return
			}
			mu.Lock()
			fibList[j] = fib
			mu.Unlock()
		}(j)
	}

	// Attendre la fin des calculs
	wg.Wait()

	executionTime := time.Since(startTime)
	fmt.Printf("Génération des %d termes de Fibonacci terminée en %s.\n", n, executionTime)
}
