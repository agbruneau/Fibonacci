package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// MatrixExponentiation offers a classic and efficient approach to calculating Fibonacci
// numbers, with a time complexity of O(log n). This method is based on a
// fundamental property of the Fibonacci sequence, which can be expressed in
// matrix form:
//
//	[ F(n+1) F(n)   ] = [ 1 1 ]^n
//	[ F(n)   F(n-1) ]   [ 1 0 ]
//
// To compute F(n), the algorithm calculates the n-th power of the matrix Q = [[1, 1], [1, 0]]
// using a technique known as binary exponentiation (or exponentiation by squaring).
// This dramatically reduces the number of required matrix multiplications compared
// to a naive iterative approach.
//
// This implementation is further enhanced with several key optimizations:
//   - Zero-Allocation: A `sync.Pool` is used to recycle `matrixState` objects,
//     which hold the matrices and temporary variables. This practice minimizes
//     memory allocations and reduces pressure on the garbage collector.
//   - Parallel Processing: When dealing with matrices containing very large numbers
//     (as determined by a configurable threshold), the matrix multiplication
//     process is parallelized to leverage the power of multi-core processors.
//   - Symmetric Squaring: The algorithm uses a specialized function, `squareSymmetricMatrix`,
//     for squaring symmetric matrices. This optimization reduces the total number
//     of `big.Int` multiplications required, leading to a noticeable performance gain.
type MatrixExponentiation struct{}

// Name returns the descriptive name of the algorithm. This name is displayed in
// the application's user interface, providing a clear and concise identification
// of the calculation method, including its key performance characteristics.
func (c *MatrixExponentiation) Name() string {
	return "Matrix Exponentiation (O(log n), Parallel, Zero-Alloc)"
}

// CalculateCore computes F(n) using the matrix exponentiation method.
//
// This function implements the binary exponentiation algorithm to efficiently
// calculate the n-th power of the Fibonacci matrix. It also handles state
// management through pooling and reports progress to the caller.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - reporter: The function for reporting progress.
//   - n: The index of the Fibonacci number to calculate.
//   - threshold: The bit size threshold for parallelizing multiplications.
//   - fftThreshold: The bit size threshold for using FFT-based multiplication.
//
// Returns the calculated Fibonacci number and an error if one occurred.
func (c *MatrixExponentiation) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}

	state := acquireMatrixState()
	defer releaseMatrixState(state)

	mul := func(dest, x, y *big.Int) {
		if fftThreshold > 0 {
			// Utiliser FFT si le plus petit des deux opérandes dépasse le seuil
			minBitLen := x.BitLen()
			if b := y.BitLen(); b < minBitLen {
				minBitLen = b
			}
			if minBitLen > fftThreshold {
				mulFFT(dest, x, y)
				return
			}
		}
		dest.Mul(x, y)
	}

	exponent := n - 1
	numBits := bits.Len64(exponent)
	// Cache la vérification de parallélisme pour éviter les appels répétés
	numCPU := runtime.NumCPU()
	useParallel := numCPU > 1 && threshold > 0

	var invNumBits float64
	if numBits > 0 {
		invNumBits = 1.0 / float64(numBits)
	}

	for i := 0; i < numBits; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		reporter(float64(i) * invNumBits)

		if (exponent>>uint(i))&1 == 1 {
			// Décide du parallélisme selon la taille max des opérandes impliqués
			inParallel := useParallel && maxBitLenMatrix(state.p) > threshold
			multiplyMatrices(state.tempMatrix, state.res, state.p, state, inParallel, mul)
			state.res, state.tempMatrix = state.tempMatrix, state.res
		}

		if i < numBits-1 {
			inParallel := useParallel && maxBitLenMatrix(state.p) > threshold
			squareSymmetricMatrix(state.tempMatrix, state.p, state, inParallel, mul)
			state.p, state.tempMatrix = state.tempMatrix, state.p
		}
	}
	return new(big.Int).Set(state.res.a), nil
}

// multiplyMatrices décide dynamiquement entre la version classique (8 multiplications)
// et la version Strassen (7 multiplications + additions) en fonction d'un seuil sur
// la taille en bits des opérandes. Pour des petites tailles, la version classique
// évite le surcoût d'additions de Strassen.
// DefaultStrassenThresholdBits contrôle le basculement vers Strassen.
// Modifiable au démarrage via la configuration.
var DefaultStrassenThresholdBits = 256

func multiplyMatrices(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int)) {
    strassenThresholdBits := DefaultStrassenThresholdBits
    if maxBitLenTwoMatrices(m1, m2) <= strassenThresholdBits {
		multiplyMatricesClassic(dest, m1, m2, state, inParallel, mul)
		return
	}
	multiplyMatricesStrassen(dest, m1, m2, state, inParallel, mul)
}

// multiplyMatricesStrassen: implémentation Strassen 2x2 (7 multiplications)
func multiplyMatricesStrassen(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int)) {
	// Let m1 = [[a, b], [c, d]] and m2 = [[e, f], [g, h]]
	// The temporary variables from the state object are used to store intermediate results.
	p1, p2, p3, p4, p5, p6, p7 := state.p1, state.p2, state.p3, state.p4, state.p5, state.p6, state.p7
	s1, s2, s3, s4, s5, s6, s7, s8, s9, s10 := state.s1, state.s2, state.s3, state.s4, state.s5, state.s6, state.s7, state.s8, state.s9, state.s10

	// Pre-calculate sums and differences
	s1.Sub(m2.b, m2.d)  // f - h
	s2.Add(m1.a, m1.b)  // a + b
	s3.Add(m1.c, m1.d)  // c + d
	s4.Sub(m2.c, m2.a)  // g - e
	s5.Add(m1.a, m1.d)  // a + d
	s6.Add(m2.a, m2.d)  // e + h
	s7.Sub(m1.b, m1.d)  // b - d
	s8.Add(m2.c, m2.d)  // g + h
	s9.Sub(m1.a, m1.c)  // a - c
	s10.Add(m2.a, m2.b) // e + f

	// Execute the 7 multiplications
	tasks := []func(){
		func() { mul(p1, m1.a, s1) }, // p1 = a * (f - h)
		func() { mul(p2, s2, m2.d) }, // p2 = (a + b) * h
		func() { mul(p3, s3, m2.a) }, // p3 = (c + d) * e
		func() { mul(p4, m1.d, s4) }, // p4 = d * (g - e)
		func() { mul(p5, s5, s6) },   // p5 = (a + d) * (e + h)
		func() { mul(p6, s7, s8) },   // p6 = (b - d) * (g + h)
		func() { mul(p7, s9, s10) },  // p7 = (a - c) * (e + f)
	}
	executeTasks(inParallel, tasks)

	// Calculate final matrix elements
	// Using temporary state variables to avoid modifying destination values prematurely.
	valA, valB, valC, valD := state.s1, state.s2, state.s3, state.s4
	valA.Add(p5, p4)
	valA.Sub(valA, p2)
	valA.Add(valA, p6)

	valB.Add(p1, p2)

	valC.Add(p3, p4)

	valD.Add(p5, p1)
	valD.Sub(valD, p3)
	valD.Sub(valD, p7)

	dest.a.Set(valA)
	dest.b.Set(valB)
	dest.c.Set(valC)
	dest.d.Set(valD)
}

// squareSymmetricMatrix computes the square of a symmetric matrix.
//
// This function is a performance optimization that reduces the number of integer
// multiplications required to square a matrix. For a symmetric matrix, where
// b equals c, some calculations become redundant. This method avoids those
// redundancies, resulting in a faster computation.
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

// multiplyMatricesClassic: multiplication 2x2 naïve (8 multiplications)
func multiplyMatricesClassic(dest, m1, m2 *matrix, state *matrixState, inParallel bool, mul func(dest, x, y *big.Int)) {
	// m1 = [[a,b],[c,d]], m2 = [[e,f],[g,h]]
	// Utilise les tampons de l'état pour éviter des allocations
	// a = a*e + b*g
	// b = a*f + b*h
	// c = c*e + d*g
	// d = c*f + d*h

	// Tampons
	ae, bg := state.p1, state.p2
	af, bh := state.p3, state.p4
	ce, dg := state.p5, state.p6
	cf, dh := state.s1, state.s2

	tasks := []func(){
		func() { mul(ae, m1.a, m2.a) },
		func() { mul(bg, m1.b, m2.c) },
		func() { mul(af, m1.a, m2.b) },
		func() { mul(bh, m1.b, m2.d) },
		func() { mul(ce, m1.c, m2.a) },
		func() { mul(dg, m1.d, m2.c) },
		func() { mul(cf, m1.c, m2.b) },
		func() { mul(dh, m1.d, m2.d) },
	}
	executeTasks(inParallel, tasks)

	dest.a.Add(ae, bg)
	dest.b.Add(af, bh)
	dest.c.Add(ce, dg)
	dest.d.Add(cf, dh)
}

// maxBitLenMatrix retourne la taille en bits maximale parmi les 4 éléments
// Optimisé: évite les appels répétés en utilisant des variables locales
func maxBitLenMatrix(m *matrix) int {
	max := m.a.BitLen()
	if b := m.b.BitLen(); b > max {
		max = b
	}
	if c := m.c.BitLen(); c > max {
		max = c
	}
	if d := m.d.BitLen(); d > max {
		max = d
	}
	return max
}

// maxBitLenMatrixCached retourne la taille maximale avec mise en cache optionnelle
// pour éviter les recalculs répétés sur la même matrice
func maxBitLenMatrixCached(m *matrix, cached *int) int {
	if cached != nil && *cached > 0 {
		return *cached
	}
	result := maxBitLenMatrix(m)
	if cached != nil {
		*cached = result
	}
	return result
}

// maxBitLenTwoMatrices retourne la taille en bits maximale parmi deux matrices
// Optimisé: calcul direct sans appels de fonction supplémentaires
func maxBitLenTwoMatrices(m1, m2 *matrix) int {
	max := m1.a.BitLen()
	if b := m1.b.BitLen(); b > max {
		max = b
	}
	if c := m1.c.BitLen(); c > max {
		max = c
	}
	if d := m1.d.BitLen(); d > max {
		max = d
	}
	if b := m2.a.BitLen(); b > max {
		max = b
	}
	if b := m2.b.BitLen(); b > max {
		max = b
	}
	if c := m2.c.BitLen(); c > max {
		max = c
	}
	if d := m2.d.BitLen(); d > max {
		max = d
	}
	return max
}

// executeTasks executes a set of tasks, in parallel if specified.
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
