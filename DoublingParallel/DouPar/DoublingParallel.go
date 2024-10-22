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
		// Retourne une erreur si n est négatif
		return nil, errors.New("n doit être un entier positif")
	}
	if n == 0 {
		// Le premier terme de la séquence de Fibonacci est 0
		return big.NewInt(0), nil
	}
	if n == 1 {
		// Le deuxième terme de la séquence de Fibonacci est 1
		return big.NewInt(1), nil
	}

	// Vérifier si le résultat est déjà mémorisé pour éviter des recalculs inutiles
	if val, exists := memo.Load(n); exists {
		return val.(*big.Int), nil
	}

	// Initialisation des valeurs de Fibonacci de base
	a := big.NewInt(0) // F(0)
	b := big.NewInt(1) // F(1)
	c := new(big.Int)
	d := new(big.Int)

	// Algorithme de doublement pour calculer F(n)
	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		// c = a * (2*b - a)
		t1 := new(big.Int).Lsh(b, 1)  // 2*b
		t2 := new(big.Int).Sub(t1, a) // 2*b - a
		c.Mul(a, t2)                  // a * (2*b - a)

		// d = a^2 + b^2
		t3 := new(big.Int).Mul(a, a) // a^2
		t4 := new(big.Int).Mul(b, b) // b^2
		d.Add(t3, t4)                // a^2 + b^2

		// Mise à jour de a et b en fonction du bit actuel de n
		if ((n >> i) & 1) == 0 {
			a.Set(c)
			b.Set(d)
		} else {
			a.Set(d)
			b.Add(c, d)
		}
	}

	// Stocker le résultat dans la mémoïsation pour des utilisations futures
	memo.Store(n, a)

	return a, nil
}

// calcFibonacciSegment calcule une plage de termes de Fibonacci et envoie les résultats sur un canal
func calcFibonacciSegment(start, end int, results chan<- struct {
	index int
	fib   *big.Int
}, wg *sync.WaitGroup) {
	// Signale la fin de la goroutine à la fin de la fonction
	defer wg.Done()

	// Calculer chaque terme de la plage [start, end]
	for j := start; j <= end; j++ {
		fib, err := fibDoublingMemo(j)
		if err != nil {
			// En cas d'erreur, afficher un message d'erreur et continuer
			fmt.Printf("Erreur au calcul de F(%d): %v\n", j, err)
			continue
		}
		// Envoyer le résultat sur le canal
		results <- struct {
			index int
			fib   *big.Int
		}{index: j, fib: fib}
	}
}

func main() {
	// Définir les flags de la ligne de commande
	nPtr := flag.Int("n", 100000, "Nombre de termes de Fibonacci à générer")
	workersPtr := flag.Int("workers", 16, "Nombre de goroutines à utiliser")
	flag.Parse()

	n := *nPtr
	numWorkers := *workersPtr

	// Vérifier que n est un entier positif
	if n < 0 {
		fmt.Println("Erreur: n doit être un entier positif.")
		return
	}

	// Enregistrer le temps de départ pour mesurer le temps d'exécution
	startTime := time.Now()

	// Initialisation des structures de données
	results := make(chan struct {
		index int
		fib   *big.Int
	}, n)
	var wg sync.WaitGroup

	// Calcul des segments pour chaque goroutine
	segmentSize := n / numWorkers
	remaining := n % numWorkers

	// Démarrer les goroutines pour calculer les segments de Fibonacci
	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 {
			// Ajouter les éléments restants au dernier segment
			end += remaining
		}
		if end > n {
			end = n
		}
		// Ajouter une goroutine au groupe d'attente
		wg.Add(1)
		go calcFibonacciSegment(start, end, results, &wg)
	}

	// Fermeture du canal une fois tous les calculs terminés
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collecter les résultats et les stocker dans une liste
	fibList := make([]*big.Int, n+1)
	for res := range results {
		fibList[res.index] = res.fib
	}

	// Afficher uniquement le temps d'exécution dans le terminal
	executionTime := time.Since(startTime)
	fmt.Printf("Génération des %d termes de Fibonacci terminée en %s.\n", n, executionTime)
}
