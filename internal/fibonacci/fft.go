package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// mulFFT performs the multiplication of two *big.Int instances, x and y.
// It uses an algorithm based on the Fast Fourier Transform (FFT), and returns
// the result as a new *big.Int. This method is particularly efficient for
// multiplying very large numbers, typically offering a time complexity of
// O(N log N), where N is the number of bits in the operands. It serves as a
// high-performance alternative to the standard big.Int.Mul method for numbers
// exceeding a certain size threshold.
//
// The operands are x and y. It returns the product.
func mulFFT(x, y *big.Int) *big.Int {
	return bigfft.Mul(x, y)
}
