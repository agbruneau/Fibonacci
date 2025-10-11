// EXPLICATION ACADÉMIQUE :
// Ce fichier centralise la logique de multiplication via la Transformée de Fourier Rapide (FFT).
// En isolant cette fonction, on la rend réutilisable par différents algorithmes (comme
// FastDoubling et MatrixExponentiation) sans dupliquer le code et en évitant les
// conflits de redéclaration dans le même package.
package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// mulFFT exécute la multiplication de deux grands entiers en utilisant la transformée de Fourier rapide (FFT).
// Cette méthode est asymptotiquement plus rapide que la multiplication standard pour des nombres
// de très grande taille. Elle alloue et retourne un nouveau `*big.Int` pour le résultat.
func mulFFT(x, y *big.Int) *big.Int {
	// bigfft.Mul est optimisé pour retourner un nouveau big.Int, ce qui correspond
	// à notre besoin de stocker le résultat dans des variables temporaires sans
	// modifier les opérandes.
	return bigfft.Mul(x, y)
}