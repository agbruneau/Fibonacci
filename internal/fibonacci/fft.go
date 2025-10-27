package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// mulFFT performs the multiplication of two `*big.Int` instances, `x` and `y`,
// using an algorithm based on the Fast Fourier Transform (FFT). The result is
// stored in `dest`. This method is particularly efficient for multiplying very
// large numbers, typically offering a time complexity of O(N log N), where N is
// the number of bits in the operands. It serves as a high-performance alternative
// to the standard `big.Int.Mul` method for numbers exceeding a certain size
// threshold.
func mulFFT(dest, x, y *big.Int) {
	dest.Set(bigfft.Mul(x, y))
}