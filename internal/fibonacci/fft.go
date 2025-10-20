package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// mulFFT effectue la multiplication de deux `*big.Int`, `x` et `y`, en
// utilisant un algorithme basé sur la Transformée de Fourier Rapide (FFT).
// Le résultat est stocké dans `dest`. Cette méthode est efficace pour les
// très grands nombres, offrant une complexité en O(N log N).
func mulFFT(dest, x, y *big.Int) {
	dest.Set(bigfft.Mul(x, y))
}