package fibonacci

import (
	"context"
	"log"
	"math/big"
	"math/bits"
	"sync"
)

const (
	binetPrecisionMargin = 128
	log2Phi              = 0.6942419136306173
)

var (
	// Constantes de haute précision initialisées de manière paresseuse et sécurisée (Thread-safe).
	phi       *big.Float
	sqrt5     *big.Float
	binetInit sync.Once
)

// initializeConstants initialise Phi et Sqrt5. Appelé via sync.Once.
func initializeConstants() {
	var err error
	// Représentations décimales de haute précision extraites du code original.
	phiStr := "1.61803398874989484820458683436563811772030917980576"
	phi, _, err = new(big.Float).Parse(phiStr, 10)
	if err != nil {
		// Panique si l'initialisation échoue (dépendance critique).
		log.Fatalf("Erreur critique d'initialisation de Phi: %v", err)
	}
	sqrt5Str := "2.23606797749978969640917366873127623544061835961152"
	sqrt5, _, err = new(big.Float).Parse(sqrt5Str, 10)
	if err != nil {
		log.Fatalf("Erreur critique d'initialisation de Sqrt5: %v", err)
	}
}

func init() {
	register("binet", "Binet (Float, O(log N))", &Binet{})
}

// Binet implémente la formule de Binet : F(N) = round( Phi^N / sqrt(5) ).
type Binet struct{}

// Calculate exécute la formule de Binet.
func (b *Binet) Calculate(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	// Initialisation sécurisée des constantes.
	binetInit.Do(initializeConstants)

	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Calcul de la précision requise en bits.
	prec := uint(float64(n)*log2Phi + binetPrecisionMargin)

	// Création de copies locales avec la précision adéquate.
	phiPrec := new(big.Float).SetPrec(prec).Set(phi)
	sqrt5Prec := new(big.Float).SetPrec(prec).Set(sqrt5)

	// Calcul de Phi^N via exponentiation binaire.
	result := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phiPrec)
	exponent := uint(n)

	numBitsInN := bits.Len(exponent)

	for currentStep := 0; exponent > 0; currentStep++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exponent&1 == 1 {
			result.Mul(result, base)
		}
		if exponent > 1 {
			base.Mul(base, base)
		}
		exponent >>= 1

		// Rapport de progression (jusqu'à 99%).
		if progress != nil && numBitsInN > 0 {
			pct := (float64(currentStep+1) / float64(numBitsInN)) * 99.0
			select {
			case progress <- pct:
			default:
			}
		}
	}

	// Division par sqrt(5).
	result.Quo(result, sqrt5Prec)

	// Arrondi à l'entier le plus proche (ajout de 0.5 et troncature).
	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	result.Add(result, half)

	z := new(big.Int)
	result.Int(z)

	if progress != nil {
		select {
		case progress <- 100.0:
		default:
		}
	}
	return z, nil
}
