package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// mulFFT performs the multiplication of two `*big.Int`, `x` and `y`, using
// an algorithm based on the Fast Fourier Transform (FFT). The result is stored
// in `dest`. This method is efficient for very large numbers, offering a
// complexity of O(N log N).
func mulFFT(dest, x, y *big.Int) {
	dest.Set(bigfft.Mul(x, y))
}