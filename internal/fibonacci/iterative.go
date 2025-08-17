package fibonacci

import (
	"context"
	"math/big"
	"sync"
)

// contextCheckInterval définit la fréquence de vérification du contexte pour les algorithmes O(N).
const contextCheckInterval = 4096

func init() {
	// Enregistrement automatique de l'algorithme.
	register("iterative", "Iterative (O(N))", &Iterative{})
}

// Iterative implémente l'algorithme itératif standard.
type Iterative struct{}

// Calculate exécute l'algorithme itératif.
func (i *Iterative) Calculate(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Acquisition des variables depuis le pool.
	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)
	temp := pool.Get().(*big.Int)
	defer pool.Put(temp)

	progressUpdateInterval := n/100 + 1

	for iter := 2; iter <= n; iter++ {
		// Vérification réactive du contexte.
		if iter%contextCheckInterval == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		temp.Add(a, b)
		a.Set(b)
		b.Set(temp)

		// Rapport de progression.
		if progress != nil && iter%progressUpdateInterval == 0 {
			pct := (float64(iter) / float64(n)) * 100.0
			select {
			case progress <- pct:
			default:
			}
		}
	}

	if progress != nil {
		select {
		case progress <- 100.0:
		default:
		}
	}
	// Retourne une nouvelle copie du résultat, car 'b' sera retourné au pool.
	return new(big.Int).Set(b), nil
}
