// @module(fibonacci)
// @author(Jules)
// @date(2023-10-27)
// @version(1.1)
//
// @description(Implémentation du calcul de Fibonacci via l'exponentiation matricielle (O(log n)).)
// @pedagogical(Illustre l'exponentiation binaire, l'optimisation par symétrie, le parallélisme et la gestion mémoire zéro-allocation.)
package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// @struct(MatrixExponentiation)
// @description(Implémente `coreCalculator` avec l'exponentiation matricielle.)
// @theory(
//   La transformation de Fibonacci est linéaire et peut être représentée par la matrice Q = [[1, 1], [1, 0]].
//   Calculer F(n) équivaut à élever Q à la puissance n-1, soit Q^(n-1). F(n) est l'élément [0,0] de la matrice résultat.
//   L'exponentiation binaire (ou par la mise au carré) permet de calculer Q^k en O(log k) multiplications matricielles.
//   L'optimisation par symétrie (Q est symétrique, donc toutes ses puissances le sont) réduit le nombre de multiplications d'entiers de 8 à 4 pour chaque carré de matrice.
// )
type MatrixExponentiation struct{}

// @method(Name)
// @description(Retourne le nom de l'algorithme et ses optimisations.)
func (c *MatrixExponentiation) Name() string {
	return "Matrix Exponentiation (O(log n), Parallèle, Zéro-Alloc)"
}

// @method(CalculateCore)
// @description(Implémente la logique principale de l'algorithme.)
func (c *MatrixExponentiation) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}

	state := acquireMatrixState()
	defer releaseMatrixState(state)

	mul := func(dest, x, y *big.Int) {
		useFFT := fftThreshold > 0 && x.BitLen() > fftThreshold && y.BitLen() > fftThreshold
		if useFFT {
			mulFFT(dest, x, y)
		} else {
			dest.Mul(x, y)
		}
	}

	exponent := n - 1
	numBits := bits.Len64(exponent)
	useParallel := runtime.NumCPU() > 1 && threshold > 0

	var invNumBits float64
	if numBits > 0 {
		invNumBits = 1.0 / float64(numBits)
	}

	for i := 0; i < numBits; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		reporter(float64(i) * invNumBits)

		inParallel := useParallel && state.p.a.BitLen() > threshold
		if (exponent>>uint(i))&1 == 1 {
			multiplyMatrices(state.tempMatrix, state.res, state.p, state, inParallel, mul)
			state.res, state.tempMatrix = state.tempMatrix, state.res
		}

		if i < numBits-1 {
			squareSymmetricMatrix(state.tempMatrix, state.p, state, inParallel, mul)
			state.p, state.tempMatrix = state.tempMatrix, state.p
		}
	}
	return new(big.Int).Set(state.res.a), nil
}

// @function(multiplyMatrices)
// @description(Multiplie deux matrices 2x2, C = A * B.)
func multiplyMatrices(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int)) {
	tasks := []func(){
		func() { mul(state.t1, m1.a, m2.a) }, func() { mul(state.t2, m1.b, m2.c) },
		func() { mul(state.t3, m1.a, m2.b) }, func() { mul(state.t4, m1.b, m2.d) },
		func() { mul(state.t5, m1.c, m2.a) }, func() { mul(state.t6, m1.d, m2.c) },
		func() { mul(state.t7, m1.c, m2.b) }, func() { mul(state.t8, m1.d, m2.d) },
	}
	executeTasks(inParallel, tasks)

	dest.a.Add(state.t1, state.t2)
	dest.b.Add(state.t3, state.t4)
	dest.c.Add(state.t5, state.t6)
	dest.d.Add(state.t7, state.t8)
}

// @function(squareSymmetricMatrix)
// @description(Calcule le carré d'une matrice symétrique avec une optimisation réduisant le nombre de multiplications.)
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int)) {
	a2, b2, d2 := state.t1, state.t2, state.t3
	b_ad, ad := state.t4, state.t5
	ad.Add(mat.a, mat.d)

	tasks := []func(){
		func() { mul(a2, mat.a, mat.a) },
		func() { mul(b2, mat.b, mat.b) },
		func() { mul(d2, mat.d, mat.d) },
		func() { mul(b_ad, mat.b, ad) },
	}
	executeTasks(inParallel, tasks)

	dest.a.Add(a2, b2)
	dest.b.Set(b_ad)
	dest.c.Set(b_ad)
	dest.d.Add(b2, d2)
}

// @function(executeTasks)
// @description(Exécute un slice de fonctions, en parallèle si spécifié.)
func executeTasks(inParallel bool, tasks []func()) {
	if !inParallel || len(tasks) < 2 {
		for _, task := range tasks {
			task()
		}
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(tasks) - 1)
	for i := 0; i < len(tasks)-1; i++ {
		go func(i int) {
			defer wg.Done()
			tasks[i]()
		}(i)
	}
	tasks[len(tasks)-1]()
	wg.Wait()
}
