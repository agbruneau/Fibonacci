//go:build gmp

package fibonacci

import (
	"context"
	"math/big"

	"github.com/ncw/gmp"
)

func init() {
	RegisterCalculator("gmp", func() coreCalculator { return &GMPCalculator{} })
}

// GMPCalculator implements the Fibonacci calculation using the GMP library.
// It requires the 'gmp' build tag and the libgmp library installed on the system.
// This implementation uses the Fast Doubling algorithm but leverages GMP's
// highly optimized C assembly routines for arithmetic operations.
type GMPCalculator struct{}

// Name returns the name of the algorithm.
func (c *GMPCalculator) Name() string {
	return "GMP (Fast Doubling)"
}

// CalculateCore executes the calculation using GMP's optimized arithmetic.
func (c *GMPCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}

	// Use Fast Doubling algorithm:
	// F(2k) = F(k) * (2*F(k+1) - F(k))
	// F(2k+1) = F(k+1)^2 + F(k)^2

	a := gmp.NewInt(0) // F(0)
	b := gmp.NewInt(1) // F(1)

	// Pre-allocate temporaries to avoid allocation in loop
	t1 := gmp.NewInt(0)
	t2 := gmp.NewInt(0)

	// Determine the highest bit
	// n is uint64.
	var numBits int
	for i := 63; i >= 0; i-- {
		if (n>>i)&1 == 1 {
			numBits = i + 1
			break
		}
	}

	// Progress reporting setup
	var lastReported float64
	totalWork := CalcTotalWork(numBits)
	currentWork := 0.0
	powers := PrecomputePowers4(numBits)

	// Iterate from MSB-1 down to 0
	// We skip the MSB itself because it corresponds to the initial state (0,1) -> (1,1) if we considered it,
	// but the standard loop structure usually starts with k=0 (F(0), F(1)) and processes all bits of n.
	// Actually, let's trace:
	// Init: k=0, a=F(0)=0, b=F(1)=1.
	// If we process MSB (which is always 1):
	//   Doubling: F(0), F(1) -> F(0), F(1) (0, 1).
	//   Bit 1: (0, 1) -> (1, 1) -> k=1. Correct.
	// So we should iterate from numBits-1 down to 0.

	for i := numBits - 1; i >= 0; i-- {
		// Check cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calculate F(2k) and F(2k+1) from F(k) and F(k+1)
		// Let a = F(k), b = F(k+1)

		// t1 = 2b
		t1.MulUint32(b, 2)
		// t1 = 2b - a
		t1.Sub(t1, a)
		// t1 = a * (2b - a)  -> New a (F(2k))
		t1.Mul(a, t1)

		// t2 = a^2
		t2.Mul(a, a)
		// a = b^2 (reuse variable a temporarily, we don't need old a anymore as it's used in t1 and t2 calculation starts now)
		// Wait, t1 calculation used 'a'. Now we need 'a' for t2. t2 uses old 'a'.
		// Correct order:
		// t1 = a * (2b - a)  (depends on old a, old b)
		// t2 = a^2 + b^2     (depends on old a, old b)

		// My previous logic:
		// t2.Mul(a, a)  <-- Uses old a. Good.
		// a.Mul(b, b)   <-- Overwrites a with b^2. OLD a is lost here? No, 'a' variable is overwritten, but t1 holds new a? No.
		// t1 holds new F(2k).
		// We need to store b^2 somewhere.
		// Reuse 'a' variable? 'a' holds F(k). We need it for t2.Mul(a,a).

		// Refined logic:
		// 1. t1 = 2*b - a
		// 2. t1 = a * t1      (t1 is now F(2k))
		// 3. t2 = a^2
		// 4. b = b^2          (b is now b^2)
		// 5. t2 = t2 + b      (t2 is now F(2k+1))
		// 6. a = t1           (a is now F(2k))
		// 7. b = t2           (b is now F(2k+1))

		// Wait step 4 overwrites b. We need b for step 1?
		// Step 1 is done before step 4.
		// Step 1 uses old b. Step 4 overwrites it.
		// Step 1: t1 = 2*old_b - old_a.
		// Step 2: t1 = old_a * t1. (t1 has F(2k)).
		// Step 3: t2 = old_a^2.
		// Step 4: b = old_b^2.
		// Step 5: t2 = t2 + b. (t2 has F(2k+1)).
		// Step 6: a = t1.
		// Step 7: b = t2.

		// This looks correct. 'a' is modified in step 6, but used in step 1,2,3. So step 6 must be after 3.
		// 'b' is modified in step 4, but used in step 1. So step 4 must be after 1.

		// Let's re-verify code:
		// t1.MulUint32(b, 2)  // t1 = 2b
		// t1.Sub(t1, a)       // t1 = 2b - a
		// t1.Mul(a, t1)       // t1 = a*(2b-a) = F(2k). Correct.

		// t2.Mul(a, a)        // t2 = a^2. Correct.
		// a.Mul(b, b)         // a = b^2. Reuse 'a' to store b^2?
		//                     // Wait, 'a' is needed for next loop iteration as F(2k).
		//                     // But we have t1 holding F(2k). So we can overwrite 'a' here temporarily?
		//                     // Yes, provided we update a=t1 later.
		// t2.Add(t2, a)       // t2 = a^2 + b^2 = F(2k+1). Correct.

		// a.Set(t1)           // a = F(2k). Correct.
		// b.Set(t2)           // b = F(2k+1). Correct.

		// Code in previous thought:
		/*
		t1.MulUint32(b, 2)
		t1.Sub(t1, a)
		t1.Mul(a, t1)

		t2.Mul(a, a)
		a.Mul(b, b) // a becomes b^2
		t2.Add(t2, a)

		a.Set(t1)
		b.Set(t2)
		*/
		// This looks correct.

		// Implementation:
		// t1 = 2b
		t1.MulUint32(b, 2)
		// t1 = 2b - a
		t1.Sub(t1, a)
		// t1 = a * (2b - a)  -> New a
		t1.Mul(a, t1)

		// t2 = a^2
		t2.Mul(a, a)
		// Store b^2 in a temporarily (since we have new a in t1)
		a.Mul(b, b)
		// t2 = a^2 + b^2 -> New b
		t2.Add(t2, a)

		// Update a and b
		a.Set(t1)
		b.Set(t2)

		// If current bit is 1, advance to k = 2k + 1
		// (a, b) = (b, a+b)
		if (n>>i)&1 == 1 {
			t1.Add(a, b)
			a.Set(b)
			b.Set(t1)
		}

		// Report progress
		currentWork = ReportStepProgress(reporter, &lastReported, totalWork, currentWork, i, numBits, powers)
	}

	// Result is F(n) which is in 'a'.
	// Convert gmp.Int to big.Int using Bytes()
	// gmp.Int.Bytes() returns big-endian absolute value.
	// Since F(n) >= 0, this is sufficient.
	bytes := a.Bytes()
	res := new(big.Int).SetBytes(bytes)

	return res, nil
}
