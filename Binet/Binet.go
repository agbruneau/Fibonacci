// main.go
// Version 3.0 — Calcul exact de F(n) par la formule de Binet
// Auteur : André‑Guy Bruneau (17 avril 2025)
// ---------------------------------------------------------------------
// Ce programme calcule le n-ième nombre de Fibonacci en précision
// arbitraire à l’aide de la formule fermée de Binet :
//
//	F(n) = round(φ^n / √5)
//
// où φ = (1 + √5) / 2.
// L’élévation à la puissance est réalisée par l’exponentiation rapide
// (square‑and‑multiply) en nombres flottants arbitraires (math/big.Float).
// Un contexte d’annulation (timeout) et un indicateur de progression en
// pourcentage sont intégrés pour le confort de l’utilisateur.
// ---------------------------------------------------------------------
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/bits"
	"time"
)

// Constantes irrationnelles stockées sous forme décimale longue afin de
// préserver la précision lors du chargement. Elles sont analysées à la volée
// selon la précision requise.
const (
	phiStr   = "1.61803398874989484820458683436563811772030917980576286214"
	sqrt5Str = "2.23606797749978969640917366873127623544061835961152572427"
)

// newFloat crée un *big.Float à partir d’une chaîne décimale en définissant la
// précision binaire (bits) donnée.
func newFloat(s string, prec uint) *big.Float {
	f, _, err := big.ParseFloat(s, 10, prec, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	return f
}

// powFloat élève x à la puissance n (n ≥ 0) par exponentiation rapide. La
// fonction respecte le contexte d’annulation et affiche la progression toutes
// les reportInterval.
func powFloat(ctx context.Context, x *big.Float, n int, prec uint, reportInterval time.Duration) (*big.Float, error) {
	result := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(x)
	exp := n

	totalBits := bits.Len(uint(exp))
	lastReport := time.Now()

	for exp > 0 {
		// Annulation éventuelle
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exp&1 == 1 {
			result.Mul(result, base)
		}
		base.Mul(base, base) // x = x²
		exp >>= 1

		// Affichage progression
		if time.Since(lastReport) >= reportInterval || exp == 0 {
			progress := float64(totalBits-bits.Len(uint(exp))) / float64(totalBits) * 100
			fmt.Printf("\rProgression : %.2f%% ", progress)
			lastReport = time.Now()
		}
	}
	fmt.Println()
	return result, nil
}

// fibBinet calcule F(n) exactement à l’aide de Binet.
func fibBinet(ctx context.Context, n int, reportInterval time.Duration) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être ≥ 0 (reçu %d)", n)
	}
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}

	// Précision : ⌈n·log2(φ) − log2(√5)⌉ + marge
	bitsNeeded := uint(float64(n)*math.Log2((1+math.Sqrt(5))/2) + 4)

	phi := newFloat(phiStr, bitsNeeded)
	sqrt5 := newFloat(sqrt5Str, bitsNeeded)

	pow, err := powFloat(ctx, phi, n, bitsNeeded, reportInterval)
	if err != nil {
		return nil, err
	}

	pow.Quo(pow, sqrt5)             // φ^n / √5
	pow.Add(pow, big.NewFloat(0.5)) // arrondi à l’entier le plus proche

	z := new(big.Int)
	pow.Int(z) // conversion en entier exact
	return z, nil
}

func main() {
	// Paramètres CLI
	n := flag.Int("n", 1_000_000, "Indice n du terme de Fibonacci (≥ 0)")
	timeout := flag.Duration("timeout", 2*time.Minute, "Durée maximale d’exécution")
	interval := flag.Duration("progress", time.Second, "Intervalle d’actualisation de la progression")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	log.Printf("Calcul de F(%d)…", *n)
	start := time.Now()

	value, err := fibBinet(ctx, *n, *interval)
	if err != nil {
		log.Fatalf("échec : %v", err)
	}

	duration := time.Since(start)
	fmt.Printf("F(%d) calculé en %v\n", *n, duration.Round(time.Millisecond))
	fmt.Printf("Nombre de chiffres : %d\n", len(value.Text(10)))

	// Affichage scientifique compact
	sci := new(big.Float).SetInt(value).Text('e', 8)
	fmt.Printf("F(%d) ≈ %s (notation scientifique)\n", *n, sci)
}
