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

func smartMultiply(z, x, y *big.Int, fftThreshold int, karatsubaThreshold int) (*big.Int, error) {
	bx := x.BitLen()
	by := y.BitLen()

	// Tier 1: FFT Multiplication
	if fftThreshold > 0 && bx > fftThreshold && by > fftThreshold {
		return bigfft.MulTo(z, x, y)
	}

	// Tier 2: Optimized Karatsuba Multiplication
	if karatsubaThreshold > 0 && bx > karatsubaThreshold && by > karatsubaThreshold {
		if z == nil {
			z = new(big.Int)
		}
		return bigfft.KaratsubaMultiplyTo(z, x, y), nil
	}

	// Tier 3: standard math/big Multiplication
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, y), nil
}

// smartSquare performs optimized squaring, choosing between standard Mul,
// optimized Karatsuba (internal/bigfft), and FFT (internal/bigfft) based on the size.
func smartSquare(z, x *big.Int, fftThreshold int, karatsubaThreshold int) (*big.Int, error) {
	bx := x.BitLen()

	// Tier 1: FFT Squaring
	if fftThreshold > 0 && bx > fftThreshold {
		return bigfft.SqrTo(z, x)
	}

	// Tier 2: Optimized Karatsuba Squaring
	if karatsubaThreshold > 0 && bx > karatsubaThreshold {
		if z == nil {
			z = new(big.Int)
		}
		return bigfft.KaratsubaSqrTo(z, x), nil
	}

	// Tier 3: standard math/big Squaring
	if z == nil {
		z = new(big.Int)
	}
	return z.Mul(x, x), nil
}
