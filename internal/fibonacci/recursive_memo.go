package fibonacci

import (
	"context"
	"math/big"
	"sync"
)

func init() {
	// Enregistrement automatique de l'algorithme récursif avec mémoïsation.
	register("recursive_memo", "Recursive w/ Memo (O(N))", &RecursiveMemo{})
}

// RecursiveMemo implémente l'algorithme de Fibonacci en utilisant la récursion et la mémoïsation.
type RecursiveMemo struct{}

// Calculate exécute le calcul. Il met en place le cache et appelle la fonction helper récursive.
func (r *RecursiveMemo) Calculate(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Le cache de mémoïsation est local à chaque appel de Calculate.
	memo := make(map[int]*big.Int, n)

	result, err := r.fibRec(ctx, n, memo, pool, progress)
	if err != nil {
		return nil, err
	}

	// La valeur retournée par fibRec provient du cache.
	// Nous retournons une nouvelle copie pour garantir l'isolation avec l'appelant.
	return new(big.Int).Set(result), nil
}

// fibRec est la fonction récursive privée qui effectue le calcul.
func (r *RecursiveMemo) fibRec(ctx context.Context, n int, memo map[int]*big.Int, pool *sync.Pool, progress chan<- float64) (*big.Int, error) {
	// Vérification de l'annulation du contexte à chaque étape de la récursion.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Si la valeur est déjà dans le cache, on la retourne.
	if val, found := memo[n]; found {
		return val, nil
	}

	// Cas de base de la récursion.
	if n <= 1 {
		val := big.NewInt(int64(n))
		memo[n] = val
		return val, nil
	}

	// Appels récursifs pour F(n-1) et F(n-2).
	res1, err := r.fibRec(ctx, n-1, memo, pool, progress)
	if err != nil {
		return nil, err
	}
	res2, err := r.fibRec(ctx, n-2, memo, pool, progress)
	if err != nil {
		return nil, err
	}

	// Utilisation d'un objet du pool pour le calcul de la somme afin de réduire les allocations.
	sum := pool.Get().(*big.Int)
	sum.Add(res1, res2)

	// Le résultat stocké dans le cache doit être une nouvelle instance, pas celle du pool.
	result := new(big.Int).Set(sum)
	memo[n] = result

	// L'objet temporaire est retourné au pool.
	pool.Put(sum)

	// Rapport de progression basé sur la taille du cache.
	// On ne notifie pas à chaque fois pour éviter de surcharger le canal.
	if progress != nil && len(memo)%64 == 0 {
		pct := (float64(len(memo)) / float64(n)) * 100.0
		select {
		case progress <- pct:
		default:
		}
	}

	if n > 0 && len(memo) == n+1 && progress != nil {
		select {
		case progress <- 100.0:
		default:
		}
	}


	return result, nil
}
