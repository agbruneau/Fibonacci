package fibonacci

import (
	"math/big"

	"github.com/agbru/fibcalc/internal/bigfft"
)

// mulFFT performs the multiplication of two *big.Int instances, x and y.
// It uses an algorithm based on the Fast Fourier Transform (FFT), and returns
// the result as a new *big.Int. This method is particularly efficient for
// multiplying very large numbers, typically offering a time complexity of
// O(N log N), where N is the number of bits in the operands. It serves as a
// high-performance alternative to the standard big.Int.Mul method for numbers
// exceeding a certain size threshold.
//
// Parameters:
//   - x: The first operand.
//   - y: The second operand.
//
// Returns:
//   - *big.Int: The product of x and y.
//   - error: An error if the calculation failed.
func mulFFT(x, y *big.Int) (*big.Int, error) {
	return bigfft.Mul(x, y)
}

// sqrFFT performs optimized squaring of a *big.Int using FFT.
// Squaring is more efficient than general multiplication because
// we only need to transform x once, saving approximately 33% of
// the FFT computation time for large numbers.
//
// Parameters:
//   - x: The operand to square.
//
// Returns:
//   - *big.Int: The result of x * x.
//   - error: An error if the calculation failed.
func sqrFFT(x *big.Int) (*big.Int, error) {
	return bigfft.Sqr(x)
}

// smartMultiply performs multiplication, choosing between Karatsuba (math/big)
// and FFT (internal/bigfft) based on the size of the operands.
// It also attempts to reuse the storage of `z` if `MulTo` is available/used.
//
// Parameters:
//   - z: The destination big.Int.
//   - x: The first operand.
//   - y: The second operand.
//   - threshold: The bit length threshold for switching to FFT.
//
// Returns:
//   - *big.Int: The result of x * y.
//   - error: An error if the calculation failed.
func smartMultiply(z, x, y *big.Int, threshold int) (*big.Int, error) {
	// Optimization: use MulTo if FFT is used to avoid allocation.
	// But first, check if we should use FFT.
	if threshold > 0 {
		bx := x.BitLen()
		by := y.BitLen()
		if bx > threshold && by > threshold {
			return bigfft.MulTo(z, x, y)
		}
	}
	// Handle nil z to be consistent with the MultiplicationStrategy contract
	// which allows z to be nil (see strategy.go documentation)
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, y), nil
}

// smartSquare performs optimized squaring, choosing between Karatsuba (math/big)
// and FFT (internal/bigfft) based on the size of the operand.
// Squaring is more efficient than general multiplication because we can
// exploit the symmetry of the computation (x * x).
//
// Parameters:
//   - z: The destination big.Int (may be reused for storage).
//   - x: The operand to square.
//   - threshold: The bit length threshold for switching to FFT.
//
// Returns:
//   - *big.Int: The result of x * x.
//   - error: An error if the calculation failed.
func smartSquare(z, x *big.Int, threshold int) (*big.Int, error) {
	// Use FFT-based squaring for large numbers
	if threshold > 0 && x.BitLen() > threshold {
		return bigfft.SqrTo(z, x)
	}
	// Handle nil z to be consistent with the MultiplicationStrategy contract
	// which allows z to be nil (see strategy.go documentation)
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, x), nil
}
