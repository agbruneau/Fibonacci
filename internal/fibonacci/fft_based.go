package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
)

// FFTBasedCalculator implémente `coreCalculator` en utilisant exclusivement la
// multiplication basée sur la FFT pour les opérations sur les grands nombres.
type FFTBasedCalculator struct{}

// Name retourne le nom de l'algorithme.
func (c *FFTBasedCalculator) Name() string {
	return "FFT-Based Doubling"
}

// CalculateCore calcule F(n) en utilisant l'algorithme "Fast Doubling" où
// toutes les multiplications de `big.Int` sont effectuées via `mulFFT`.
// Cette approche est optimale pour des nombres `n` extrêmement grands.
func (c *FFTBasedCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	s := acquireState()
	defer releaseState(s)

	numBits := bits.Len64(n)

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
		mulFFT(s.t3, s.f_k, s.t2)
		mulFFT(s.t1, s.f_k1, s.f_k1)
		mulFFT(s.t4, s.f_k, s.f_k)
		s.f_k.Set(s.t3)
		s.f_k1.Add(s.t1, s.t4)

		// Étape d'Addition
		if (n>>uint(i))&1 == 1 {
			s.t1.Set(s.f_k1)
			s.f_k1.Add(s.f_k1, s.f_k)
			s.f_k.Set(s.t1)
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