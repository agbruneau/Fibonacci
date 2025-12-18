package fibonacci

import (
	"context"
	"fmt"
	"math/big"
)

// MaxLUTIndex defines the size of the expanded lookup table.
// F(1024) has approximately 214 decimal digits, which is small enough
// to store 1025 pointers to big.Int (~100-200KB including big.Int overhead).
const MaxLUTIndex = 1024

// LUTCalculator provides instant responses for Fibonacci numbers up to MaxLUTIndex.
type LUTCalculator struct{}

// Name returns the descriptive name of the LUT algorithm.
func (c *LUTCalculator) Name() string {
	return fmt.Sprintf("Lookup Table (O(1), up to n=%d)", MaxLUTIndex)
}

// CalculateCore returns F(n) from the pre-computed table.
func (c *LUTCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
	if n > MaxLUTIndex {
		return nil, fmt.Errorf("n=%d exceeds LUT maximum index of %d", n, MaxLUTIndex)
	}

	if reporter != nil {
		reporter(1.0)
	}

	return lookupLarge(n), nil
}

var extendedLUT [MaxLUTIndex + 1]*big.Int

func init() {
	extendedLUT[0] = big.NewInt(0)
	if MaxLUTIndex > 0 {
		extendedLUT[1] = big.NewInt(1)
		for i := uint64(2); i <= MaxLUTIndex; i++ {
			extendedLUT[i] = new(big.Int).Add(extendedLUT[i-1], extendedLUT[i-2])
		}
	}
}

// lookupLarge returns a copy of the n-th Fibonacci number from the extended LUT.
func lookupLarge(n uint64) *big.Int {
	if n > MaxLUTIndex {
		return nil
	}
	return new(big.Int).Set(extendedLUT[n])
}
