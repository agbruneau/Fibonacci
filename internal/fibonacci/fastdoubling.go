package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// OptimizedFastDoubling implémente l'interface `coreCalculator` en utilisant
// l'algorithme "Fast Doubling".
//
// L'algorithme "Fast Doubling" repose sur les identités suivantes :
// F(2k) = F(k) * [2*F(k+1) - F(k)]
// F(2k+1) = F(k)² + F(k+1)²
// Il itère sur les bits de `n` du plus significatif au moins significatif,
// effectuant une étape de "doublage" et, si le bit est à 1, une étape
// "d'addition", atteignant une complexité en O(log n).
type OptimizedFastDoubling struct{}

// Name retourne le nom de l'algorithme.
func (fd *OptimizedFastDoubling) Name() string {
	return "Fast Doubling (O(log n), Parallèle, Zéro-Alloc)"
}

// CalculateCore exécute le calcul de F(n) en utilisant l'algorithme
// "Fast Doubling". Il optimise la gestion de la mémoire en utilisant un pool
// d'objets pour les états de calcul et parallélise les multiplications de
// grands nombres.
func (fd *OptimizedFastDoubling) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	mul := func(dest, x, y *big.Int) {
		if fftThreshold > 0 && x.BitLen() > fftThreshold && y.BitLen() > fftThreshold {
			mulFFT(dest, x, y)
		} else {
			dest.Mul(x, y)
		}
	}

	s := acquireState()
	defer releaseState(s)

	numBits := bits.Len64(n)
	useParallel := runtime.GOMAXPROCS(0) > 1 && threshold > 0

	var totalWork, workDone, workOfStep, four big.Int
	four.SetInt64(4)
	if numBits > 0 {
		totalWork.Exp(&four, big.NewInt(int64(numBits)), nil).Sub(&totalWork, big.NewInt(1)).Div(&totalWork, big.NewInt(3))
	}
	lastReportedProgress := -1.0
	const reportThreshold = 0.01

	for i := numBits - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Étape de Doublage
		s.t2.Lsh(s.f_k1, 1).Sub(s.t2, s.f_k)

		if useParallel && s.f_k1.BitLen() > threshold {
			parallelMultiply3Optimized(s, mul)
		} else {
			mul(s.t3, s.f_k, s.t2)
			mul(s.t1, s.f_k1, s.f_k1)
			mul(s.t4, s.f_k, s.f_k)
		}

		s.f_k.Add(s.t1, s.t4)
		s.f_k, s.f_k1, s.t3 = s.t3, s.f_k, s.f_k1

		// Étape d'Addition
		if (n>>uint(i))&1 == 1 {
			s.t1.Add(s.f_k, s.f_k1)
			s.f_k, s.f_k1, s.t1 = s.f_k1, s.t1, s.f_k
		}

		if totalWork.Sign() > 0 {
			j := int64(numBits - 1 - i)
			workOfStep.Exp(&four, big.NewInt(j), nil)
			workDone.Add(&workDone, &workOfStep)
			workDoneFloat, _ := new(big.Float).SetInt(&workDone).Float64()
			totalWorkFloat, _ := new(big.Float).SetInt(&totalWork).Float64()
			currentProgress := workDoneFloat / totalWorkFloat
			if currentProgress-lastReportedProgress >= reportThreshold || i == 0 {
				reporter(currentProgress)
				lastReportedProgress = currentProgress
			}
		}
	}
	return new(big.Int).Set(s.f_k), nil
}

// parallelMultiply3Optimized exécute les trois multiplications de l'étape de
// doublage en parallèle pour optimiser les performances sur les machines
// multi-cœurs.
func parallelMultiply3Optimized(s *calculationState, mul func(dest, x, y *big.Int)) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		mul(s.t3, s.f_k, s.t2)
	}()
	go func() {
		defer wg.Done()
		mul(s.t1, s.f_k1, s.f_k1)
	}()
	mul(s.t4, s.f_k, s.f_k)
	wg.Wait()
}