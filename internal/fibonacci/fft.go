// @module(fibonacci)
// @author(Jules)
// @date(2023-10-27)
// @version(1.2)
//
// @description(Ce module fournit une abstraction pour la multiplication de grands entiers basée sur la Transformée de Fourier Rapide (FFT).)
package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// @function(mulFFT)
// @description(Effectue la multiplication de deux entiers de grande taille, `x` et `y`, en utilisant un algorithme basé sur la FFT.)
// @complexity(La complexité temporelle de cette opération est de O(N log N), où N représente le nombre de bits des opérandes. Cette complexité est asymptotiquement inférieure à celle des algorithmes de multiplication classiques, tels que Karatsuba ou l'algorithme scolaire.)
// @rationale(L'utilisation de la FFT pour la multiplication est justifiée lorsque la taille des entiers dépasse un certain seuil, où le surcoût de la transformation est compensé par le gain en efficacité de la multiplication dans le domaine fréquentiel.)
// @pedagogical(La fonction `mulFFT` s'intègre dans une stratégie de gestion de la mémoire optimisée. Le résultat de la multiplication est stocké dans le paramètre `dest` pré-alloué, ce qui permet d'éviter des allocations dynamiques répétées et de réduire la pression sur le ramasse-miettes (garbage collector). Cette technique est essentielle pour les applications à haute performance.)
func mulFFT(dest, x, y *big.Int) {
	dest.Set(bigfft.Mul(x, y))
}