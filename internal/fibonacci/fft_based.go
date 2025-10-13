// @module(fibonacci)
// @author(Jules)
// @date(2023-10-27)
// @version(1.2)
//
// @description(Ce module fournit une implémentation de l'algorithme de Fibonacci qui utilise la multiplication basée sur la FFT pour les très grands nombres.)

package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
)

// @struct(FFTBasedCalculator)
// @description(Implémente `coreCalculator` en utilisant la multiplication FFT pour toutes les opérations sur les grands nombres.)
type FFTBasedCalculator struct{}

// @method(Name)
// @description(Retourne le nom de l'algorithme.)
func (c *FFTBasedCalculator) Name() string {
	return "FFT-Based Doubling"
}

// @method(CalculateCore)
// @description(Calcule F(n) en utilisant l'algorithme "Fast Doubling" avec des multiplications FFT.)
// @pedagogical(Cette implémentation est une spécialisation de l'algorithme "Fast Doubling". Elle est conçue pour les scénarios où la taille des nombres est systématiquement très grande, justifiant l'utilisation inconditionnelle de la multiplication basée sur la FFT, qui a une meilleure complexité asymptotique (O(N log N)) que les méthodes standards.)
func (c *FFTBasedCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	// Acquisition d'un état pré-alloué depuis un pool pour éviter les allocations dynamiques.
	s := acquireState()
	defer releaseState(s) // Libération de l'état dans le pool après usage.

	numBits := bits.Len64(n)

	// Le calcul de la progression est pondéré pour refléter la complexité.
	var totalWork, workDone, workOfStep, four big.Int
	four.SetInt64(4)
	if numBits > 0 {
		totalWork.Exp(&four, big.NewInt(int64(numBits)), nil).Sub(&totalWork, big.NewInt(1)).Div(&totalWork, big.NewInt(3))
	}
	lastReportedProgress := -1.0
	const reportThreshold = 0.01 // Seuil de changement pour notifier la progression.

	// Itération sur les bits de n, du plus significatif au moins significatif.
	for i := numBits - 1; i >= 0; i-- {
		// Vérification de l'annulation du contexte pour un arrêt précoce.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Étape de Doublage (Doubling): Calcule F(2k) et F(2k+1) à partir de F(k) et F(k+1).
		// F(2k) = F(k) * [2*F(k+1) - F(k)]
		// F(2k+1) = F(k+1)^2 + F(k)^2
		s.t2.Lsh(s.f_k1, 1).Sub(s.t2, s.f_k)

		// Utilisation systématique de la multiplication FFT.
		mulFFT(s.t3, s.f_k, s.t2)      // F(2k)
		mulFFT(s.t1, s.f_k1, s.f_k1) // F(k+1)^2
		mulFFT(s.t4, s.f_k, s.f_k)     // F(k)^2

		s.f_k.Set(s.t3)
		s.f_k1.Add(s.t1, s.t4)

		// Étape d'Addition (Addition-Step): Si le bit courant de n est à 1, met à jour les valeurs pour calculer F(2k+1).
		if (n>>uint(i))&1 == 1 {
			s.t1.Set(s.f_k1)
			s.f_k1.Add(s.f_k1, s.f_k)
			s.f_k.Set(s.t1)
		}

		// Mise à jour de la progression.
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