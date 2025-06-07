// main_test.go

package main

import (
	"context"
	"math/big"
	"sync"
	"testing"
)

// TestFibonacciAlgorithms vérifie la correction de chaque algorithme de Fibonacci
// en utilisant une approche "table-driven".
func TestFibonacciAlgorithms(t *testing.T) {
	// Cas de test avec des valeurs de Fibonacci bien connues.
	testCases := []struct {
		name    string
		n       int
		want    *big.Int
		wantErr bool // Si une erreur est attendue (par exemple, pour n < 0)
	}{
		{"n=0", 0, big.NewInt(0), false},
		{"n=1", 1, big.NewInt(1), false},
		{"n=2", 2, big.NewInt(1), false},
		{"n=7", 7, big.NewInt(13), false},
		{"n=10", 10, big.NewInt(55), false},
		{"n=20", 20, big.NewInt(6765), false},
		{"n négatif", -1, nil, true},
	}

	// Map des algorithmes à tester.
	algos := map[string]fibFunc{
		"Doublage Rapide": fibFastDoubling,
		"Matrice 2x2":     fibMatrix,
		"Binet":           fibBinet,
	}

	pool := newIntPool()
	ctx := context.Background()

	// Itère sur chaque algorithme.
	for algoName, algoFunc := range algos {
		// Itère sur chaque cas de test.
		for _, tc := range testCases {
			// t.Run permet de créer des sous-tests, ce qui facilite le débogage.
			// Le nom du test sera, par exemple, "Doublage Rapide/n=10".
			t.Run(algoName+"/"+tc.name, func(t *testing.T) {
				// Exécute la fonction de l'algorithme.
				// Le canal de progression n'est pas nécessaire pour le test de correction.
				got, err := algoFunc(ctx, nil, tc.n, pool)

				// Vérifie si une erreur était attendue.
				if tc.wantErr {
					if err == nil {
						t.Errorf("attendait une erreur pour n=%d, mais n'en a pas eu", tc.n)
					}
					return // Le test est terminé si une erreur était attendue et a eu lieu.
				}

				// Vérifie si une erreur inattendue est survenue.
				if err != nil {
					t.Fatalf("erreur inattendue: %v", err)
				}

				// Compare le résultat obtenu avec le résultat attendu.
				if got.Cmp(tc.want) != 0 {
					t.Errorf("pour F(%d), attendait %s, mais a obtenu %s", tc.n, tc.want.String(), got.String())
				}
			})
		}
	}
}

// TestFibonacciConsistencyForLargeN vérifie que les algorithmes exacts
// produisent le même résultat pour un n plus grand.
func TestFibonacciConsistencyForLargeN(t *testing.T) {
	n := 1000 // Un n assez grand pour être significatif, mais pas trop long à calculer.

	pool := newIntPool()
	ctx := context.Background()

	// Calcul avec le Doublage Rapide (considéré comme référence).
	resFastDoubling, err := fibFastDoubling(ctx, nil, n, pool)
	if err != nil {
		t.Fatalf("Le Doublage Rapide a échoué pour n=%d: %v", n, err)
	}

	// Calcul avec la Matrice 2x2.
	resMatrix, err := fibMatrix(ctx, nil, n, pool)
	if err != nil {
		t.Fatalf("La Matrice 2x2 a échoué pour n=%d: %v", n, err)
	}

	// Compare les deux résultats.
	if resFastDoubling.Cmp(resMatrix) != 0 {
		t.Errorf("Discordance pour F(%d) entre Doublage Rapide et Matrice 2x2", n)
		t.Logf("Doublage Rapide: %s...", resFastDoubling.String()[:20])
		t.Logf("Matrice 2x2:     %s...", resMatrix.String()[:20])
	}

	// Note: On ne compare pas avec Binet pour les grands n car sa précision basée
	// sur les flottants peut entraîner de légères erreurs d'arrondi.
}

// ------------------------------------------------------------
// Benchmarks
// ------------------------------------------------------------

// Un n commun pour tous les benchmarks pour une comparaison équitable.
const benchmarkN = 100000

// BenchmarkFibFastDoubling mesure les performances de l'algorithme de Doublage Rapide.
func BenchmarkFibFastDoubling(b *testing.B) {
	pool := newIntPool()
	ctx := context.Background()
	b.ReportAllocs() // Affiche le nombre d'allocations mémoire.
	b.ResetTimer()   // Réinitialise le timer pour ne pas inclure le temps de setup.

	for i := 0; i < b.N; i++ {
		// Le résultat n'est pas vérifié ici, on se concentre sur la performance.
		_, _ = fibFastDoubling(ctx, nil, benchmarkN, pool)
	}
}

// BenchmarkFibMatrix mesure les performances de l'algorithme d'exponentiation de matrice.
func BenchmarkFibMatrix(b *testing.B) {
	pool := newIntPool()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = fibMatrix(ctx, nil, benchmarkN, pool)
	}
}

// BenchmarkFibBinet mesure les performances de l'algorithme de Binet.
func BenchmarkFibBinet(b *testing.B) {
	// Le pool n'est pas utilisé par Binet, mais on le passe pour la consistance de l'API.
	var pool *sync.Pool
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = fibBinet(ctx, nil, benchmarkN, pool)
	}
}
