//
// MODULE ACADÉMIQUE : VALIDATION ET VÉRIFICATION FORMELLE EN GO
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier de test illustre un ensemble de méthodologies de validation logicielle
// appliquées à un module de calcul de haute performance. Il met en exergue les pratiques
// standard de l'industrie et introduit des techniques de vérification plus formelles.
//
// CONCEPTS FONDAMENTAUX DÉMONTRÉS :
//  1. TESTS PILOTÉS PAR LES DONNÉES (TABLE-DRIVEN TESTS) : La fonction `TestFibonacciCalculators`
//     utilise une structure de données (un tableau de structures) pour définir un corpus
//     de cas de test. Cette approche systématique améliore la lisibilité, la maintenabilité
//     et l'extensibilité de la suite de tests.
//  2. SOUS-TESTS ET PARALLÉLISATION (`t.Run()` et `t.Parallel()`) : Chaque combinaison
//     algorithme/cas de test est encapsulée dans un sous-test. Cette granularité permet :
//      - ISOLATION DES DÉFAILLANCES : L'échec d'un sous-test n'interrompt pas les autres.
//      - PRÉCISION DU DIAGNOSTIC : Le nommage hiérarchique des tests localise précisément la source de l'erreur.
//      - EXÉCUTION SÉLECTIVE : La commande `go test -run` permet de cibler des sous-ensembles de tests.
//      - EFFICACITÉ : `t.Parallel()` permet au planificateur de Go d'exécuter les tests non-dépendants en parallèle.
//  3. TESTS DE PERFORMANCE (BENCHMARKING) : Les fonctions `Benchmark*` s'intègrent au
//     framework de benchmark de Go pour mesurer la latence et les allocations mémoire.
//     Ces métriques sont essentielles pour quantifier l'efficacité des optimisations,
//     notamment la stratégie "zéro-allocation".
//  4. VÉRIFICATION DE PROPRIÉTÉS ARCHITECTURALES : Le test `TestLookupTableImmutability`
//     valide un contrat implicite fondamental : l'immuabilité de l'état partagé (la LUT),
//     prévenant ainsi les régressions subtiles et les effets de bord.
//  5. TESTS BASÉS SUR LES PROPRIÉTÉS (PROPERTY-BASED TESTING) : La fonction `TestFibonacciProperties`
//     utilise une approche plus formelle où, au lieu de tester des entrées-sorties spécifiques,
//     on vérifie qu'une propriété mathématique (l'Identité de Cassini) reste vraie pour un
//     large éventail d'entrées générées aléatoirement.
//
package fibonacci

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"golang.org/x/sync/errgroup"
)

// `knownFibResults` constitue un oracle de test, une source de vérité contenant
// des valeurs de référence pour la suite de Fibonacci. Ces valeurs, pré-calculées
// et validées, servent à vérifier l'exactitude des implémentations algorithmiques.
var knownFibResults = []struct {
	n      uint64
	result string
}{
	{0, "0"},
	{1, "1"},
	{2, "1"},
	{10, "55"},
	{20, "6765"},
	{50, "12586269025"},
	{92, "7540113804746346429"},
	{93, "12200160415121876738"}, // Dépasse la capacité de uint64
	{100, "354224848179261915075"},
	{1000, "43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875"},
}

// TestFibonacciCalculators est un test de table systématique qui confronte toutes
// les implémentations de l'interface `Calculator` à l'oracle de test `knownFibResults`.
func TestFibonacciCalculators(t *testing.T) {
	// NOTE ACADÉMIQUE : `context.Background()` sert de contexte racine non-annulable.
	// Pour des scénarios de test plus complexes, `context.WithTimeout` serait utilisé
	// pour prévenir les blocages indéfinis et garantir la terminaison des tests.
	ctx := context.Background()

	// Les implémentations à tester sont instanciées de la même manière que dans l'application
	// principale, garantissant ainsi que l'environnement de test est fidèle à l'environnement d'exécution.
	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
		"FFTBased":     NewCalculator(&FFTBasedCalculator{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			for _, testCase := range knownFibResults {
				t.Run(fmt.Sprintf("N=%d", testCase.n), func(t *testing.T) {
					t.Parallel() // Déclare ce sous-test comme parallélisable.

					expected := new(big.Int)
					expected.SetString(testCase.result, 10)

					result, err := calc.Calculate(ctx, nil, 0, testCase.n, DefaultParallelThreshold, 0)

					// --- ASSERTIONS DE VALIDITÉ ---
					if err != nil {
						t.Fatalf("Erreur inattendue retournée par le calcul : %v", err)
					}
					if result == nil {
						t.Fatal("Un résultat nul a été retourné sans erreur associée.")
					}
					if result.Cmp(expected) != 0 {
						t.Errorf("Divergence de résultat.\nAttendu: %s\nObtenu : %s", expected.String(), result.String())
					}
				})
			}
		})
	}
}

// TestLookupTableImmutability vérifie une propriété de sécurité fondamentale :
// la table de consultation (LUT) doit garantir l'immuabilité de ses données en
// retournant des copies, et non des références à son état interne.
func TestLookupTableImmutability(t *testing.T) {
	val1 := lookupSmall(10)
	expected := big.NewInt(55)
	if val1.Cmp(expected) != 0 {
		t.Fatalf("La valeur F(10) initiale est incorrecte. Attendu 55, obtenu %s", val1.String())
	}

	// Tentative de mutation de la valeur obtenue.
	// Si `lookupSmall` a incorrectement retourné une référence, cette opération
	// corrompra l'état global partagé de la LUT.
	val1.Add(val1, big.NewInt(1))

	// Re-lecture de la même valeur.
	val2 := lookupSmall(10)

	// La valeur re-lue doit impérativement être identique à la valeur originale.
	if val2.Cmp(expected) != 0 {
		t.Fatalf("Violation du principe d'immuabilité : la LUT a été modifiée par un appelant. F(10) devrait être 55, mais est maintenant %s", val2.String())
	}
}

// TestNilCoreCalculatorPanic vérifie que la fabrique `NewCalculator` échoue de manière prévisible
// (via une panique) lorsqu'elle est invoquée avec un `coreCalculator` nul, respectant ainsi son contrat.
func TestNilCoreCalculatorPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCalculator aurait dû paniquer avec un noyau nil, mais ne l'a pas fait.")
		}
	}()
	_ = NewCalculator(nil)
}

// TestProgressReporter valide que les calculateurs notifient leur progression de manière monotone
// et se terminent avec une notification finale de 1.0.
func TestProgressReporter(t *testing.T) {
	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			progressChan := make(chan ProgressUpdate, 200)
			var lastProgress float64
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				for update := range progressChan {
					if update.Value < lastProgress {
						t.Errorf("Régression de la progression détectée (non-monotone). Précédent: %f, Actuel: %f", lastProgress, update.Value)
					}
					lastProgress = update.Value
				}
			}()

			_, err := calc.Calculate(context.Background(), progressChan, 0, 10000, DefaultParallelThreshold, 0)
			close(progressChan)
			wg.Wait()

			if err != nil {
				t.Fatalf("Le calcul a échoué : %v", err)
			}

			if lastProgress != 1.0 {
				t.Errorf("La progression finale attendue est 1.0, mais la dernière valeur reçue est %f", lastProgress)
			}
		})
	}
}

// TestContextCancellation vérifie la réactivité des algorithmes à une annulation de contexte,
// une caractéristique essentielle pour les systèmes robustes et réactifs.
func TestContextCancellation(t *testing.T) {
	const n = 100_000_000 // Un grand nombre pour assurer un temps de calcul non-trivial.

	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			_, err := calc.Calculate(ctx, nil, 0, n, DefaultParallelThreshold, 0)

			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("L'erreur retournée ne correspond pas à l'annulation du contexte. Attendu: %v, Obtenu: %v", context.DeadlineExceeded, err)
			}
		})
	}
}

// --- TEST BASÉ SUR LES PROPRIÉTÉS (PROPERTY-BASED TESTING) ---
//
// CONTEXTE THÉORIQUE :
// Les tests unitaires traditionnels valident le comportement du système pour un
// sous-ensemble discret et fini de l'espace des entrées. Cette approche, bien
// qu'indispensable, ne peut garantir l'absence de défauts pour des entrées non anticipées.
// Le test basé sur les propriétés (PBT) propose un paradigme complémentaire. Il consiste
// à énoncer des propriétés universelles (invariants, lois) qui doivent être respectées
// pour toutes les entrées valides. Un moteur de test génère ensuite un grand nombre
// de cas de test aléatoires pour tenter de réfuter (falsifier) ces propriétés.
//
// PROPRIÉTÉ SÉLECTIONNÉE : L'IDENTITÉ DE CASSINI
// Pour la suite de Fibonacci, l'identité de Cassini stipule que pour tout n > 0 :
//   F(n-1) * F(n+1) - F(n)² = (-1)ⁿ
// Cette relation constitue un invariant robuste, idéal pour la validation par PBT.

func TestFibonacciProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	// NOTE: Pour une reproductibilité parfaite des tests, on fixerait le 'seed' :
	// parameters.Rng.Seed(1234)

	// Générateur d'entrées : entiers dans l'intervalle [1, 2000].
	// Cet intervalle est choisi pour équilibrer la couverture de test et le temps d'exécution.
	// n=0 est exclu car l'identité de Cassini requiert F(n-1).
	uint64Gen := gen.UInt64Range(1, 2000)

	properties := gopter.NewProperties(parameters)
	properties.Property("Identité de Cassini pour les nombres de Fibonacci", prop.ForAll(
		func(n uint64) bool {
			calc := NewCalculator(&OptimizedFastDoubling{})
			ctx := context.Background()

			// Calcul parallèle des trois termes de Fibonacci requis.
			var f_n_minus_1, f_n, f_n_plus_1 *big.Int
			var g errgroup.Group
			g.Go(func() error { var err error; f_n_minus_1, err = calc.Calculate(ctx, nil, 0, n-1, DefaultParallelThreshold, 0); return err })
			g.Go(func() error { var err error; f_n, err = calc.Calculate(ctx, nil, 0, n, DefaultParallelThreshold, 0); return err })
			g.Go(func() error { var err error; f_n_plus_1, err = calc.Calculate(ctx, nil, 0, n+1, DefaultParallelThreshold, 0); return err })

			if err := g.Wait(); err != nil {
				t.Logf("Échec du calcul de Fibonacci pour n=%d : %v", n, err)
				return false // La propriété ne peut être vérifiée.
			}

			// Évaluation du membre de gauche : F(n-1) * F(n+1) - F(n)²
			term1 := new(big.Int).Mul(f_n_minus_1, f_n_plus_1)
			term2 := new(big.Int).Mul(f_n, f_n)
			leftSide := new(big.Int).Sub(term1, term2)

			// Évaluation du membre de droite : (-1)ⁿ
			rightSide := big.NewInt(1)
			if n%2 != 0 {
				rightSide.Neg(rightSide)
			}

			// La propriété est vérifiée si les deux membres sont égaux.
			return leftSide.Cmp(rightSide) == 0
		},
		uint64Gen,
	))

	properties.TestingRun(t)
}

// --- SUITE DE TESTS DE PERFORMANCE (BENCHMARKS) ---

// runBenchmark est une fonction d'aide qui standardise l'exécution des benchmarks
// pour différents algorithmes et tailles d'entrée.
func runBenchmark(b *testing.B, calc Calculator, n uint64) {
	ctx := context.Background()
	// `b.ReportAllocs()` demande au framework de mesurer et de rapporter le nombre
	// d'allocations mémoire par opération, en plus du temps d'exécution.
	b.ReportAllocs()
	b.ResetTimer() // Réinitialise le chronomètre et les statistiques d'allocation.

	// La boucle est exécutée `b.N` fois. Le framework ajuste `b.N` de manière
	// itérative jusqu'à obtenir une mesure statistiquement stable.
	for i := 0; i < b.N; i++ {
		// L'appel à la fonction dont la performance est mesurée.
		// Les éventuels résultats sont assignés à une variable "black hole" pour
		// s'assurer que le compilateur n'élimine pas l'appel (optimisation "dead code").
		_, _ = calc.Calculate(ctx, nil, 0, n, DefaultParallelThreshold, 0)
	}
}

func BenchmarkFastDoubling1M(b *testing.B) {
	runBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 1_000_000)
}

func BenchmarkMatrixExp1M(b *testing.B) {
	runBenchmark(b, NewCalculator(&MatrixExponentiation{}), 1_000_000)
}

func BenchmarkFastDoubling10M(b *testing.B) {
	runBenchmark(b, NewCalculator(&OptimizedFastDoubling{}), 10_000_000)
}

func BenchmarkMatrixExp10M(b *testing.B) {
	runBenchmark(b, NewCalculator(&MatrixExponentiation{}), 10_000_000)
}

func BenchmarkFFTBased10M(b *testing.B) {
	runBenchmark(b, NewCalculator(&FFTBasedCalculator{}), 10_000_000)
}