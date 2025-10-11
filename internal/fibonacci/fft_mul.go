// MODULE ACADÉMIQUE : DISPATCH DE MULTIPLICATION HYBRIDE
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier illustre une stratégie d'optimisation avancée : la sélection
// dynamique d'algorithmes. Au lieu d'utiliser une seule méthode, ce module
// choisit le meilleur algorithme de multiplication (standard, Karatsuba ou FFT)
// en fonction de la taille des nombres à multiplier. Il démontre :
//  1. L'ARCHITECTURE HYBRIDE À PLUSIEURS NIVEAUX : Une extension du concept
//     observé dans `karatsuba.go`. Le système combine trois algorithmes avec des
//     compromis performance/overhead différents pour couvrir efficacement toutes
//     les échelles de grandeur.
//  2. LA CONFIGURABILITÉ POUR L'OPTIMISATION : Les seuils de basculement entre
//     les algorithmes ne sont pas des constantes figées. Ils sont exposés comme
//     des variables globales pour permettre un réglage fin (tuning) par des
//     benchmarks sur la machine cible, via des flags au lancement de l'application.
//  3. L'ABSTRACTION DE LA COMPLEXITÉ : Les modules de calcul (comme `fastdoubling`)
//     n'ont pas besoin de connaître la complexité sous-jacente. Ils appellent une
//     seule fonction (`Mul`), et le dispatcher se charge de la décision, rendant
//     le code client plus propre et maintenable.
//
package fibonacci

import (
	"math/big"

	"github.com/remyoudompheng/bigfft"
)

// KaratsubaThresholdBits est le seuil (en bits) pour passer de la multiplication
// standard à l'algorithme de Karatsuba.
// Initialisé à une valeur par défaut raisonnable, peut être surchargé au démarrage.
var KaratsubaThresholdBits = 2048 // 32 mots * 64 bits/mot

// FFTThresholdBits est le seuil (en bits) pour passer de Karatsuba à la
// multiplication par FFT (Schönhage-Strassen).
// Initialisé à une valeur par défaut raisonnable, peut être surchargé au démarrage.
var FFTThresholdBits = 200000 // Valeur inspirée du README de la bibliothèque bigfft

// Mul est la fonction de dispatch principale pour la multiplication.
// Elle choisit l'algorithme le plus approprié en fonction de la taille
// des opérandes x et y.
func Mul(z, x, y *big.Int) *big.Int {
	xBits := x.BitLen()
	yBits := y.BitLen()

	// 1. Pour les très grands nombres, utiliser la FFT.
	if xBits > FFTThresholdBits || yBits > FFTThresholdBits {
		return bigfft.Mul(x, y)
	}

	// 2. Pour les nombres de taille moyenne, utiliser Karatsuba.
	if xBits > KaratsubaThresholdBits || yBits > KaratsubaThresholdBits {
		// Note: `MulKaratsuba` a sa propre logique de seuil interne, mais elle
		// est basée sur une constante. En appelant `MulKaratsuba` seulement
		// au-dessus de notre seuil configurable, nous la contrôlons de l'extérieur.
		// Pour une intégration parfaite, nous devrions modifier `karatsuba.go`
		// pour qu'il utilise aussi cette variable, mais cela sort du cadre de
		// cet exercice d'intégration.
		return MulKaratsuba(z, x, y)
	}

	// 3. Pour les petits nombres, utiliser la multiplication standard de Go.
	return z.Mul(x, y)
}