package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Types et structures optimisées
// ------------------------------------------------------------
type fibFunc func(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)

// ------------------------------------------------------------
// Pool de big.Int pour réutilisation (réduction des allocations)
// ------------------------------------------------------------
func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// ------------------------------------------------------------
// Fast-doubling optimisé avec réutilisation de mémoire
// ------------------------------------------------------------
func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n < 2 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	a := big.NewInt(0)
	b := big.NewInt(1)
	totalBits := bits.Len(uint(n))

	for i := totalBits - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Réutilisation des big.Int via le pool [[2]]
		a2 := pool.Get().(*big.Int).Set(a)
		b2 := pool.Get().(*big.Int).Set(b)
		twoB := pool.Get().(*big.Int).Lsh(b, 1)
		twoBsubA := pool.Get().(*big.Int).Sub(twoB, a)
		c := pool.Get().(*big.Int).Mul(a, twoBsubA)
		d := pool.Get().(*big.Int).Add(a2.Mul(a2, a2), b2.Mul(b2, b2))

		a.Set(c)
		b.Set(d)

		// Libération des ressources du pool après utilisation [[4]]
		pool.Put(a2)
		pool.Put(b2)
		pool.Put(twoB)
		pool.Put(twoBsubA)
		pool.Put(c)
		pool.Put(d)

		if (uint(n)>>i)&1 == 1 {
			a.Set(d)
			b.Add(c, d)
		}

		if progress != nil && totalBits > 0 {
			progress <- float64(totalBits-i) / float64(totalBits) * 100.0
		}
	}

	if progress != nil {
		progress <- 100.0
	}
	return a, nil
}

// ------------------------------------------------------------
// Affichage de la progression simplifié
// ------------------------------------------------------------
func progressPrinter(progress <-chan float64) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case pct, ok := <-progress:
			if !ok {
				fmt.Printf("\rProgression : 100.00%%\n")
				return
			}
			fmt.Printf("\rProgression : %.2f%%", pct)
		case <-ticker.C:
		}
	}
}

// ------------------------------------------------------------
// Main optimisé pour Fast-doubling uniquement
// ------------------------------------------------------------
func main() {
	// Go gère automatiquement le parallélisme depuis Go 1.5+
	runtime.GOMAXPROCS(runtime.NumCPU()) // Conservé pour compatibilité explicite

	n := flag.Int("n", 100_000_000, "Indice n du terme de Fibonacci (≥0)")
	timeout := flag.Duration("timeout", 2*time.Minute, "Durée maximale d'exécution globale")
	flag.Parse()

	if *n < 0 {
		log.Fatalf("L'indice n doit être supérieur ou égal à 0. Reçu: %d", *n)
	}

	log.Printf("Calcul de F(%d) avec un timeout de %v...\n", *n, *timeout)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Pool de big.Int pour réduire les allocations mémoire [[2]]
	intPool := newIntPool()

	progressCh := make(chan float64, 8)
	go progressPrinter(progressCh)

	start := time.Now()
	result, err := fibFastDoubling(ctx, progressCh, *n, intPool)
	duration := time.Since(start)
	close(progressCh)

	if err != nil {
		if err == context.DeadlineExceeded {
			log.Fatalf("❌ Calcul interrompu par timeout (%v) après %v", *timeout, duration)
		}
		log.Fatalf("❌ Erreur lors du calcul : %v", err)
	}

	log.Println("Calcul terminé avec succès.")
	printFibResultDetails(result, *n, duration)
}

// ------------------------------------------------------------
// Fonction d'affichage des détails du résultat
// ------------------------------------------------------------
func printFibResultDetails(value *big.Int, n int, duration time.Duration) {
	digits := len(value.Text(10))
	fmt.Printf("F(%d) calculé en %v\n", n, duration.Round(time.Millisecond))
	fmt.Printf("Nombre de chiffres : %d\n", digits)

	if digits > 20 {
		sci := new(big.Float).SetInt(value).Text('e', 8)
		fmt.Printf("F(%d) ≈ %s (notation scientifique)\n", n, sci)
	} else {
		fmt.Printf("F(%d) = %s\n", n, value.Text(10))
	}
}
