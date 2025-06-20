package main

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"sync"
)

// fibFunc is a type for functions calculating Fibonacci numbers.
// It takes a context for cancellation, a channel for progress, the index n,
// and a pool of big.Int objects for memory reuse.
type fibFunc func(ctx context.Context, progress chan<- progressData, n int, pool *sync.Pool) (*big.Int, error)

// ------------------------------------------------------------
// Fibonacci Calculation Algorithms
// ------------------------------------------------------------

// fibBinet calculates F(n) using Binet's formula.
//
// Concept:
// This is a direct mathematical formula using the golden ratio (φ).
// F(n) = (φ^n - (-φ)^-n) / √5
// For large n, this simplifies to F(n) ≈ round(φ^n / √5).
//
// Implementation:
// Uses high-precision floating-point numbers (`big.Float`).
// The main calculation is a binary exponentiation of φ to find φ^n efficiently.
//
// Strengths/Weaknesses:
// Conceptually simple, but vulnerable to precision errors inherent in
// floating-point calculations. Often less performant and accurate than
// integer-based methods for very large values of n.
//
// Note: This algorithm does not actively use the big.Int pool as it operates on big.Float.
func fibBinet(ctx context.Context, progress chan<- progressData, n int, _ *sync.Pool) (*big.Int, error) {
	taskName := "Binet" // Used for progress reporting
	if n < 0 {
		return nil, fmt.Errorf("negative index n is not supported: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- progressData{name: taskName, pct: 100.0}
		}
		return big.NewInt(int64(n)), nil
	}

	// Required precision increases with n.
	// bits for φ^n ≈ n * log2(φ)
	// Add a safety margin (+20) for precision.
	phiVal := (1 + math.Sqrt(5)) / 2
	prec := uint(float64(n)*math.Log2(phiVal) + 20) // Increased safety margin

	// Utility function to create big.Float with the correct precision
	newFloat := func() *big.Float { return new(big.Float).SetPrec(prec) }

	sqrt5 := newFloat().SetUint64(5)
	sqrt5.Sqrt(sqrt5)

	phi := newFloat().SetUint64(1)
	phi.Add(phi, sqrt5)
	phi.Quo(phi, newFloat().SetUint64(2))

	// Calculate φ^n by binary exponentiation to minimize multiplications.
	numBitsInN := bits.Len(uint(n))

	phiToN := newFloat().SetInt64(1) // Initialize phiToN = 1
	base := newFloat().Set(phi)      // base = phi

	exponent := uint(n)
	for i := 0; i < numBitsInN; i++ {
		// Cooperative context cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // context.Canceled or context.DeadlineExceeded
		default:
		}

		if (exponent>>i)&1 == 1 { // If the i-th bit of exponent is 1
			phiToN.Mul(phiToN, base)
		}
		base.Mul(base, base) // Square the base for the next iteration

		if progress != nil {
			progress <- progressData{name: taskName, pct: (float64(i+1) / float64(numBitsInN)) * 100.0}
		}
	}

	phiToN.Quo(phiToN, sqrt5) // (phi^n) / √5

	// Round to the nearest integer by adding 0.5 before truncating.
	// big.Float.Int() truncates towards zero.
	// To round to nearest, add 0.5 if positive, subtract 0.5 if negative, then get Int.
	// Since Fibonacci numbers are non-negative, adding 0.5 is sufficient.
	half := newFloat().SetFloat64(0.5)
	phiToN.Add(phiToN, half)

	resultInt := new(big.Int)
	phiToN.Int(resultInt) // Convert to big.Int (truncates)

	if progress != nil {
		progress <- progressData{name: taskName, pct: 100.0}
	}
	return resultInt, nil
}

// fibFastDoubling calculates F(n) using the "Fast Doubling" algorithm.
//
// Concept:
// A very efficient algorithm based on mathematical identities that allow
// transitioning from F(k) and F(k+1) to F(2k) and F(2k+1) in a few operations:
// F(2k)   = F(k) * [2*F(k+1) – F(k)]
// F(2k+1) = F(k)² + F(k+1)²
//
// Implementation:
// The algorithm iterates through the bits of index `n` from left to right (most
// significant to least significant). At each step, it applies the "doubling" formulas.
// If the current bit of `n` is 1, it takes an additional step to advance.
//
// Strengths/Weaknesses:
// Extremely fast and efficient (O(log n) complexity). It's one of the best
// algorithms for this problem. It heavily uses the `sync.Pool` to optimize
// `big.Int` allocations.
func fibFastDoubling(ctx context.Context, progress chan<- progressData, n int, pool *sync.Pool) (*big.Int, error) {
	taskName := "Fast Doubling" // Used for progress reporting
	if n < 0 {
		return nil, fmt.Errorf("negative index n is not supported: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- progressData{name: taskName, pct: 100.0}
		}
		return big.NewInt(int64(n)), nil
	}

	// Initialize F(k) and F(k+1)
	// a = F(k), b = F(k+1)
	a := pool.Get().(*big.Int).SetInt64(0)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(a) // Ensure 'a' is returned to the pool when done
	defer pool.Put(b) // Ensure 'b' is returned to the pool when done

	// Temporary variables for calculations, taken from the pool.
	t1 := pool.Get().(*big.Int)
	t2 := pool.Get().(*big.Int)
	defer pool.Put(t1)
	defer pool.Put(t2)

	totalBits := bits.Len(uint(n)) // Number of bits in n
	// Iterate from the most significant bit of n down to the least significant bit
	for i := totalBits - 1; i >= 0; i-- {
		// Cooperative context cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Doubling Step:
		// F(2k)   = F(k) * [2*F(k+1) – F(k)]
		// F(2k+1) = F(k)² + F(k+1)²
		//
		// Current a = F(k), b = F(k+1)
		// We calculate F(2k) and F(2k+1) and store them in a and b respectively.

		// t1 = 2*F(k+1) - F(k) = 2*b - a
		t1.Lsh(b, 1)  // t1 = 2*b
		t1.Sub(t1, a) // t1 = 2*b - a

		// t2 = F(k)^2 = a^2
		t2.Mul(a, a) // t2 = a*a

		// New a = F(2k) = F(k) * (2*F(k+1) - F(k)) = a * t1
		a.Mul(a, t1) // a = a * t1

		// t1 = F(k+1)^2 = b^2  (reusing t1)
		t1.Mul(b, b) // t1 = b*b

		// New b = F(2k+1) = F(k)^2 + F(k+1)^2 = t2 + t1
		b.Add(t2, t1) // b = t2 + t1 (which is F(k)^2 + F(k+1)^2)

		// If the i-th bit of n is 1, apply the "addition" step:
		// F(m+1) = F(m) + F(m-1)
		// Here, if current a=F(2k), b=F(2k+1), and bit is 1, we need F(2k+1), F(2k+2)
		// New a' = F(2k+1) = b
		// New b' = F(2k+2) = F(2k) + F(2k+1) = a + b (using OLD a and b from before this if block,
		// but since a and b are updated to F(2k) and F(2k+1) respectively in this iteration,
		// it means the new a' = F(2k+1) (which is current b),
		// and new b' = F(2k+2) = F(2k) + F(2k+1) (which is current a + current b).
		if (uint(n)>>i)&1 == 1 {
			// t1 = F(2k) + F(2k+1) (this is the new F(k+1), i.e., F(2k+2))
			t1.Add(a, b) // t1 = current_a (F(2k)) + current_b (F(2k+1))
			// a becomes F(2k+1)
			a.Set(b) // a = current_b (F(2k+1))
			// b becomes F(2k+2)
			b.Set(t1) // b = t1 (F(2k+2))
		}

		if progress != nil {
			progress <- progressData{name: taskName, pct: (float64(totalBits-i) / float64(totalBits)) * 100.0}
		}
	}

	if progress != nil {
		progress <- progressData{name: taskName, pct: 100.0}
	}
	// Return a new instance to avoid returning a pooled object that might be modified.
	return new(big.Int).Set(a), nil
}

// mat2 represents a 2x2 matrix of *big.Int.
// The elements are:
// | a  b |
// | c  d |
type mat2 struct{ a, b, c, d *big.Int }

// newMat2 creates a new mat2 whose components are taken from the pool.
func newMat2(pool *sync.Pool) *mat2 {
	return &mat2{
		a: pool.Get().(*big.Int), b: pool.Get().(*big.Int),
		c: pool.Get().(*big.Int), d: pool.Get().(*big.Int),
	}
}

// release puts the matrix components back into the pool.
func (m *mat2) release(pool *sync.Pool) {
	pool.Put(m.a)
	pool.Put(m.b)
	pool.Put(m.c)
	pool.Put(m.d)
}

// set updates the target matrix values with those of another matrix.
func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// mul performs the multiplication of two matrices m1 * m2 and stores the result in the receiver matrix (m).
// m = m1 * m2
// For safety, the receiver 'm' should not be an alias of m1 or m2 if their original values are needed later,
// as 'm' is modified in place. Using a temporary matrix for results is safer if m could be m1 or m2.
// This implementation allocates temporary big.Ints for products and sums to correctly update m.
func (m *mat2) mul(m1, m2 *mat2, pool *sync.Pool) {
	// Temporary variables for intermediate products, acquired from the pool.
	p1 := pool.Get().(*big.Int) // m1.a * m2.a
	p2 := pool.Get().(*big.Int) // m1.b * m2.c
	p3 := pool.Get().(*big.Int) // m1.a * m2.b
	p4 := pool.Get().(*big.Int) // m1.b * m2.d
	p5 := pool.Get().(*big.Int) // m1.c * m2.a
	p6 := pool.Get().(*big.Int) // m1.d * m2.c
	p7 := pool.Get().(*big.Int) // m1.c * m2.b
	p8 := pool.Get().(*big.Int) // m1.d * m2.d

	// Values for the new matrix, also from pool to avoid modifying m before all parts are computed
	valA := pool.Get().(*big.Int)
	valB := pool.Get().(*big.Int)
	valC := pool.Get().(*big.Int)
	valD := pool.Get().(*big.Int)

	defer pool.Put(p1)
	defer pool.Put(p2)
	defer pool.Put(p3)
	defer pool.Put(p4)
	defer pool.Put(p5)
	defer pool.Put(p6)
	defer pool.Put(p7)
	defer pool.Put(p8)
	defer pool.Put(valA)
	defer pool.Put(valB)
	defer pool.Put(valC)
	defer pool.Put(valD)

	// Calculate m.a = (m1.a * m2.a) + (m1.b * m2.c)
	p1.Mul(m1.a, m2.a)
	p2.Mul(m1.b, m2.c)
	valA.Add(p1, p2)

	// Calculate m.b = (m1.a * m2.b) + (m1.b * m2.d)
	p3.Mul(m1.a, m2.b)
	p4.Mul(m1.b, m2.d)
	valB.Add(p3, p4)

	// Calculate m.c = (m1.c * m2.a) + (m1.d * m2.c)
	p5.Mul(m1.c, m2.a)
	p6.Mul(m1.d, m2.c)
	valC.Add(p5, p6)

	// Calculate m.d = (m1.c * m2.b) + (m1.d * m2.d)
	p7.Mul(m1.c, m2.b)
	p8.Mul(m1.d, m2.d)
	valD.Add(p7, p8)

	// Now set the receiver matrix 'm' values
	m.a.Set(valA)
	m.b.Set(valB)
	m.c.Set(valC)
	m.d.Set(valD)
}

// fibMatrix calculates F(n) by exponentiation of the matrix Q = [[1,1],[1,0]].
//
// Concept:
// Based on the property that:
//
//	Q^k  =  | F(k+1)  F(k)   |
//	       | F(k)    F(k-1) |
//
// We need to calculate Q^(n-1). F(n) will be the top-left element (res.a)
// of the resulting matrix Q^(n-1).
// Example: For n=2, Q^(2-1) = Q^1 = [[1,1],[1,0]]. F(2)=1, which is res.a.
// For n=3, Q^(3-1) = Q^2 = [[1,1],[1,0]] * [[1,1],[1,0]] = [[2,1],[1,1]]. F(3)=2, which is res.a.
//
// Implementation:
// The code calculates this matrix power using exponentiation by squaring
// (also known as binary exponentiation), a technique that reduces the
// number of matrix multiplications from O(n) to O(log n).
// The 2x2 matrix multiplication is implemented in the `mat2.mul` method.
//
// Strengths/Weaknesses:
// Very elegant and also very performant (logarithmic complexity).
// Can be slightly slower in practice than Fast Doubling due to the overhead
// of managing the 4 matrix elements and potentially more arithmetic operations
// per effective "doubling" step compared to Fast Doubling's direct formulas.
func fibMatrix(ctx context.Context, progress chan<- progressData, n int, pool *sync.Pool) (*big.Int, error) {
	taskName := "Matrix 2x2" // Used for progress reporting
	if n < 0 {
		return nil, fmt.Errorf("negative index n is not supported: %d", n)
	}
	// Base cases
	if n == 0 { // F(0) = 0
		if progress != nil {
			progress <- progressData{name: taskName, pct: 100.0}
		}
		return big.NewInt(0), nil
	}
	if n == 1 { // F(1) = 1
		if progress != nil {
			progress <- progressData{name: taskName, pct: 100.0}
		}
		return big.NewInt(1), nil
	}

	// Result matrix, initialized to identity matrix [[1,0],[0,1]]
	// This 'res' matrix will accumulate the powers of 'base'.
	res := newMat2(pool)
	defer res.release(pool)
	res.a.SetInt64(1) // res = I
	res.b.SetInt64(0)
	res.c.SetInt64(0)
	res.d.SetInt64(1)

	// Base matrix Q = [[1,1],[1,0]]
	base := newMat2(pool)
	defer base.release(pool)
	base.a.SetInt64(1)
	base.b.SetInt64(1)
	base.c.SetInt64(1)
	base.d.SetInt64(0)

	// Temporary matrix for multiplication results to avoid aliasing issues
	// when doing res = res * base or base = base * base.
	tempProduct := newMat2(pool)
	defer tempProduct.release(pool)

	// We need to calculate Q^(n-1)
	exp := uint(n - 1)
	totalSteps := bits.Len(exp) // Max number of iterations for progress reporting

	for i := 0; exp > 0; i++ {
		// Cooperative context cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exp&1 == 1 { // If current bit of exponent is 1
			// res = res * base
			tempProduct.mul(res, base, pool) // Store res * base in tempProduct
			res.set(tempProduct)             // Update res = tempProduct
		}
		exp >>= 1    // Halve the exponent (equivalent to exp = exp / 2)
		if exp > 0 { // Only square base if there are more steps
			// base = base * base (square the base for the next iteration)
			tempProduct.mul(base, base, pool) // Store base * base in tempProduct
			base.set(tempProduct)             // Update base = tempProduct
		}

		if progress != nil && totalSteps > 0 { // Avoid division by zero if totalSteps is 0 (e.g. n=1, exp=0)
			currentProgress := (float64(i+1) / float64(totalSteps)) * 100.0
			if currentProgress > 100.0 { // Cap progress at 100%
				currentProgress = 100.0
			}
			progress <- progressData{name: taskName, pct: currentProgress}
		}
	}

	if progress != nil {
		progress <- progressData{name: taskName, pct: 100.0} // Final progress update
	}

	// The result F(n) is in res.a (top-left element of Q^(n-1))
	return new(big.Int).Set(res.a), nil
}

// progressData encapsulates progress information for a task.
// fibIterative calculates F(n) using a simple iterative approach.
//
// Concept:
// This method directly applies the Fibonacci definition F(n) = F(n-1) + F(n-2).
// It starts with F(0)=0 and F(1)=1 and iteratively calculates each subsequent
// Fibonacci number up to F(n).
//
// Implementation:
// Uses a loop and two variables to keep track of the previous two Fibonacci numbers.
// `big.Int` objects are used for calculations, and the `sync.Pool` is leveraged
// to reduce allocations for these objects.
//
// Strengths/Weaknesses:
//   - Simple to understand and implement.
//   - Very memory efficient, especially with the sync.Pool.
//   - Slower than logarithmic algorithms (Fast Doubling, Matrix) for very large n,
//     as its complexity is O(n) in terms of additions. However, each addition is on
//     large numbers, so the bit complexity is higher.
//   - Can be faster for small n where the overhead of more complex algorithms is greater.
//   - Progress reporting is straightforward (percentage of iterations completed).
func fibIterative(ctx context.Context, progress chan<- progressData, n int, pool *sync.Pool) (*big.Int, error) {
	taskName := "Iterative" // Used for progress reporting
	if n < 0 {
		return nil, fmt.Errorf("negative index n is not supported for iterative method: %d", n)
	}
	if n == 0 {
		if progress != nil {
			progress <- progressData{name: taskName, pct: 100.0}
		}
		return big.NewInt(0), nil
	}
	if n == 1 {
		if progress != nil {
			progress <- progressData{name: taskName, pct: 100.0}
		}
		return big.NewInt(1), nil
	}

	a := pool.Get().(*big.Int).SetInt64(0) // F(i-2)
	b := pool.Get().(*big.Int).SetInt64(1) // F(i-1)
	currentFib := pool.Get().(*big.Int)    // To store F(i)

	defer pool.Put(a)
	defer pool.Put(b)
	defer pool.Put(currentFib)

	// Loop from 2 to n
	for i := 2; i <= n; i++ {
		// Cooperative context cancellation check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// currentFib = a + b
		currentFib.Add(a, b)
		// a = b
		a.Set(b)
		// b = currentFib
		b.Set(currentFib)

		if progress != nil {
			// Progress is based on the number of iterations out of n.
			// For n=0 or n=1, progress is 100% immediately.
			// For n > 1, progress goes from (2/n)*100 to (n/n)*100.
			// To make it smoother from 0 to 100, we can consider (i-1) out of (n-1) steps.
			if n > 1 { // Avoid division by zero if n=0 or n=1 (though handled by base cases)
				progPct := (float64(i-1) / float64(n-1)) * 100.0
				// Ensure progress doesn't exceed 100 due to float inaccuracies for the last step
				if i == n {
					progPct = 100.0
				}
				progress <- progressData{name: taskName, pct: progPct}
			}
		}
	}

	if progress != nil && n > 1 { // Ensure final 100% for n > 1 if not already sent
		progress <- progressData{name: taskName, pct: 100.0}
	}

	// The result is in 'b' (which holds F(n) after the loop finishes)
	// Return a new big.Int with the result, not the one from the pool.
	return new(big.Int).Set(b), nil
}
