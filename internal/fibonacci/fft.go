// @module(fibonacci)
// @author(Jules)
// @date(2023-10-27)
// @version(1.1)
//
// @description(Centralise la logique de multiplication via la Transformée de Fourier Rapide (FFT).)
package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// @function(mulFFT)
// @description(Multiplie deux grands entiers en utilisant la FFT.)
// @complexity(O(N log N), où N est le nombre de bits.)
// @rationale(Asymptotiquement plus rapide que la multiplication standard pour de très grands nombres.)
// @pedagogical(Le résultat est stocké dans `dest` pour s'intégrer à la stratégie de gestion mémoire par pooling.)
func mulFFT(dest, x, y *big.Int) {
	dest.Set(bigfft.Mul(x, y))
}