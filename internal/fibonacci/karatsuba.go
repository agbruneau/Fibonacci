// MODULE ACADÉMIQUE : MULTIPLICATION DE GRANDS NOMBRES - ALGORITHME DE KARATSUBA
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier fournit une implémentation de l'algorithme de Karatsuba, une méthode
// de multiplication de grands nombres plus efficace que l'algorithme "classique"
// pour les nombres dépassant un certain seuil. Il illustre des concepts clés :
//  1. ALGORITHMIQUE AVANCÉE : Application d'un algorithme "diviser pour régner" qui
//     réduit la complexité de la multiplication de O(n^2) à environ O(n^1.585).
//  2. OPTIMISATION HYBRIDE : Démonstration de l'importance d'une approche hybride.
//     Karatsuba a un surcoût (overhead) qui le rend moins performant pour les petits
//     nombres. Le code bascule donc vers l'algorithme standard de `math/big`
//     en dessous d'un seuil critique (`karatsubaThreshold`), combinant le meilleur
//     des deux mondes.
//  3. GESTION MANUELLE DE LA MÉMOIRE (POOLING) : Pour atteindre des performances
//     maximales, l'implémentation utilise un pool d'objets (`sync.Pool`) pour
//     recycler les `big.Int` temporaires, évitant des milliers d'allocations
//     mémoire dans les appels récursifs et réduisant la pression sur le GC.
//
package fibonacci

import (
	"math/big"
)

// karatsubaThreshold est le nombre de "mots" (words) dans un `big.Int` en dessous
// duquel la multiplication standard de Go est plus rapide que Karatsuba.
// Un `big.Int` est un slice de `big.Word`. La taille d'un `big.Word` dépend de
// l'architecture (généralement 64 bits). Un seuil de 32 mots correspond à
// 32 * 64 = 2048 bits. C'est une valeur de départ raisonnable qui peut être
// affinée par des benchmarks.
const karatsubaThreshold = 32 // en `big.Word`

// karatsubaState contient tous les `big.Int` temporaires nécessaires pour un
// appel de la fonction de Karatsuba, afin d'éviter les allocations.
type karatsubaState struct {
	x0, x1, y0, y1 *big.Int
	z0, z1, z2     *big.Int
	t1, t2         *big.Int
}

// Reset réinitialise l'état pour la prochaine utilisation.
func (s *karatsubaState) Reset() {
	// Il n'est pas nécessaire de remettre les valeurs à zéro car elles sont
	// toujours utilisées comme destination (.Set, .Mul, .Add, .Sub) avant
	// d'être lues.
}

// karatsubaPool est un pool d'objets `karatsubaState` pour la réutilisation.
var karatsubaPool = NewPool(func() *karatsubaState {
	s := &karatsubaState{}
	s.x0 = new(big.Int)
	s.x1 = new(big.Int)
	s.y0 = new(big.Int)
	s.y1 = new(big.Int)
	s.z0 = new(big.Int)
	s.z1 = new(big.Int)
	s.z2 = new(big.Int)
	s.t1 = new(big.Int)
	s.t2 = new(big.Int)
	return s
})

func acquireKaratsubaState() *karatsubaState { return acquireFromPool(karatsubaPool) }
func releaseKaratsubaState(s *karatsubaState) { releaseToPool(karatsubaPool, s) }


// MulKaratsuba multiplie deux `big.Int`, x et y, et stocke le résultat dans z.
// Il utilise l'algorithme de Karatsuba si les nombres sont assez grands.
func MulKaratsuba(z, x, y *big.Int) *big.Int {
	// Pour les petits nombres, l'algorithme standard est plus rapide.
	// On utilise la taille en "mots" (`.Bits()`) comme mesure.
	if x.BitLen() < karatsubaThreshold*64 || y.BitLen() < karatsubaThreshold*64 {
		return z.Mul(x, y)
	}

	// Acquérir un état temporaire depuis le pool.
	s := acquireKaratsubaState()
	defer releaseKaratsubaState(s) // Garantit la libération.

	// Lancer la multiplication récursive.
	karatsubaRecursive(z, x, y, s)
	return z
}

// karatsubaRecursive est le cœur de l'algorithme.
func karatsubaRecursive(z, x, y *big.Int, s *karatsubaState) {
	// Cas de base de la récursion : si les nombres sont petits, on utilise la multiplication standard.
	if x.BitLen() < karatsubaThreshold*64 || y.BitLen() < karatsubaThreshold*64 {
		z.Mul(x, y)
		return
	}

	// 1. Déterminer le point de division (m).
	// On choisit m comme la moitié de la longueur du plus grand des deux nombres.
	m := (max(x.BitLen(), y.BitLen()) + 1) / 2

	// 2. Diviser x et y en deux parties (haute et basse) au point m.
	// x = x1 * 2^m + x0
	// y = y1 * 2^m + y0
	s.x1.Rsh(x, uint(m)) // x1 = x >> m
	s.x0.Sub(x, s.t1.Lsh(s.x1, uint(m))) // x0 = x - (x1 << m)
	s.y1.Rsh(y, uint(m)) // y1 = y >> m
	s.y0.Sub(y, s.t1.Lsh(s.y1, uint(m))) // y0 = y - (y1 << m)

	// 3. Calculs récursifs des sous-problèmes.
	// Chaque appel récursif a besoin de son propre état pour ses temporaires.
	// On acquiert un nouvel état pour chaque appel et on le libère juste après.

	// z2 = x1 * y1
	s_rec_z2 := acquireKaratsubaState()
	karatsubaRecursive(s.z2, s.x1, s.y1, s_rec_z2)
	releaseKaratsubaState(s_rec_z2)

	// z0 = x0 * y0
	s_rec_z0 := acquireKaratsubaState()
	karatsubaRecursive(s.z0, s.x0, s.y0, s_rec_z0)
	releaseKaratsubaState(s_rec_z0)

	// z1 = (x0 + x1) * (y0 + y1)
	s.t1.Add(s.x0, s.x1)
	s.t2.Add(s.y0, s.y1)
	s_rec_z1 := acquireKaratsubaState()
	karatsubaRecursive(s.z1, s.t1, s.t2, s_rec_z1)
	releaseKaratsubaState(s_rec_z1)

	// 4. Combiner les résultats.
	// z1 = z1 - z2 - z0
	s.z1.Sub(s.z1, s.z2)
	s.z1.Sub(s.z1, s.z0)

	// z = z2 * 2^(2m) + z1 * 2^m + z0
	s.t1.Lsh(s.z2, uint(2*m)) // t1 = z2 << (2*m)
	s.t2.Lsh(s.z1, uint(m))   // t2 = z1 << m
	z.Add(s.t1, s.t2)
	z.Add(z, s.z0)
}

// max est une fonction utilitaire pour trouver le maximum de deux entiers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}