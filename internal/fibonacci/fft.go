package fibonacci

import (
	"math/big"

	"example.com/fibcalc/internal/bigfft"
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
func mulFFT(x, y *big.Int) *big.Int {
	return bigfft.Mul(x, y)
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
func smartMultiply(z, x, y *big.Int, threshold int) *big.Int {
	// Optimization: use MulTo if FFT is used to avoid allocation.
	// But first, check if we should use FFT.
	if threshold > 0 {
		bx := x.BitLen()
		by := y.BitLen()
		if bx > threshold && by > threshold {
			return bigfft.MulTo(z, x, y)
		}
	}
	return z.Mul(x, y)
}
