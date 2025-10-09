// MODULE ACADÉMIQUE : ALGORITHME "FAST DOUBLING" AVEC MULTIPLICATION DE KARATSUBA
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier est une variante de `fastdoubling.go`. Son but est de démontrer
// comment une optimisation modulaire (ici, un algorithme de multiplication
// plus rapide) peut être "injectée" dans un algorithme existant pour en
// améliorer les performances. Il illustre :
//  1. L'AMÉLIORATION INCREMENTALE : Au lieu de réécrire l'algorithme, on remplace
//     une de ses composantes critiques (la multiplication) par une version plus
//     performante.
//  2. L'INTÉGRATION DE MODULES : Montre comment le module `karatsuba.go` est
//     utilisé par ce module pour accomplir sa tâche.
//  3. L'ANALYSE DE PERFORMANCE COMPARATIVE : La coexistence de cette version avec
//     `OptimizedFastDoubling` permet de réaliser des benchmarks précis pour
//     mesurer le gain de performance apporté par Karatsuba dans ce contexte
//     spécifique.
//
package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// FastDoublingKaratsuba est une implémentation de `coreCalculator` qui utilise
// la multiplication de Karatsuba.
type FastDoublingKaratsuba struct{}

// Name retourne le nom de l'algorithme pour l'affichage.
func (fd *FastDoublingKaratsuba) Name() string {
	return "Fast Doubling (Karatsuba)"
}

// CalculateCore implémente la logique principale de l'algorithme.
// Le code est très similaire à `OptimizedFastDoubling`, la seule différence
// étant l'appel à `MulKaratsuba` au lieu de `big.Mul`.
func (fd *FastDoublingKaratsuba) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}

	s := acquireState()
	defer releaseState(s)

	s.f_k.SetInt64(0)  // F(0)
	s.f_k1.SetInt64(1) // F(1)

	numBits := bits.Len64(n)
	useParallel := runtime.GOMAXPROCS(0) > 1

	for i := numBits - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Étape de Doubling
		// t2 = 2*f_k1 - f_k
		s.t2.Lsh(s.f_k1, 1)
		s.t2.Sub(s.t2, s.f_k)

		// Utilisation de la multiplication de Karatsuba pour les opérations coûteuses.
		if useParallel && s.f_k1.BitLen() > threshold {
			parallelMultiply3Karatsuba(s)
		} else {
			MulKaratsuba(s.t3, s.f_k, s.t2)    // t3 = f_k * t2
			MulKaratsuba(s.t1, s.f_k1, s.f_k1) // t1 = f_k1²
			MulKaratsuba(s.t4, s.f_k, s.f_k)   // t4 = f_k²
		}

		// f_k = t3
		// f_k1 = t1 + t4
		s.f_k.Set(s.t3)
		s.f_k1.Add(s.t1, s.t4)

		// Étape d'Addition
		if (n>>uint(i))&1 == 1 {
			s.t1.Set(s.f_k1)
			s.f_k1.Add(s.f_k1, s.f_k)
			s.f_k.Set(s.t1)
		}

		// Rapport de progression simplifié pour cet exemple.
		// Une implémentation complète utiliserait le même modèle pondéré que OptimizedFastDoubling.
		if reporter != nil {
			progress := float64(numBits-i) / float64(numBits)
			reporter(progress)
		}
	}

	return new(big.Int).Set(s.f_k), nil
}

// parallelMultiply3Karatsuba exécute les trois multiplications en parallèle
// en utilisant MulKaratsuba.
func parallelMultiply3Karatsuba(s *calculationState) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Tâche A: t3 = f_k * t2
	go func() {
		defer wg.Done()
		MulKaratsuba(s.t3, s.f_k, s.t2)
	}()

	// Tâche B: t1 = f_k1 * f_k1
	go func() {
		defer wg.Done()
		MulKaratsuba(s.t1, s.f_k1, s.f_k1)
	}()

	// Tâche C: t4 = f_k * f_k (dans la goroutine principale)
	MulKaratsuba(s.t4, s.f_k, s.f_k)

	wg.Wait()
}