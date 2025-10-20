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

// knownFibResults est un oracle de test contenant des valeurs de référence
// pour la suite de Fibonacci, utilisées pour valider l'exactitude des calculs.
var knownFibResults = []struct {
	n      uint64
	result string
}{
	{0, "0"}, {1, "1"}, {2, "1"}, {10, "55"}, {20, "6765"},
	{50, "12586269025"}, {92, "7540113804746346429"},
	{93, "12200160415121876738"}, {100, "354224848179261915075"},
	{1000, "43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875"},
}

// TestFibonacciCalculators valide systématiquement toutes les implémentations
// de `Calculator` par rapport à l'oracle de test `knownFibResults`.
func TestFibonacciCalculators(t *testing.T) {
	ctx := context.Background()
	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
		"FFTBased":     NewCalculator(&FFTBasedCalculator{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			for _, testCase := range knownFibResults {
				t.Run(fmt.Sprintf("N=%d", testCase.n), func(t *testing.T) {
					t.Parallel()
					expected := new(big.Int)
					expected.SetString(testCase.result, 10)
					result, err := calc.Calculate(ctx, nil, 0, testCase.n, DefaultParallelThreshold, 0)

					if err != nil {
						t.Fatalf("Erreur inattendue : %v", err)
					}
					if result == nil {
						t.Fatal("Résultat nul retourné sans erreur.")
					}
					if result.Cmp(expected) != 0 {
						t.Errorf("Résultat incorrect.\nAttendu: %s\nObtenu : %s", expected.String(), result.String())
					}
				})
			}
		})
	}
}

// TestLookupTableImmutability vérifie que la table de consultation (LUT)
// garantit l'immuabilité de ses données en retournant des copies.
func TestLookupTableImmutability(t *testing.T) {
	val1 := lookupSmall(10)
	expected := big.NewInt(55)
	if val1.Cmp(expected) != 0 {
		t.Fatalf("Valeur F(10) incorrecte. Attendu 55, obtenu %s", val1.String())
	}
	val1.Add(val1, big.NewInt(1))
	val2 := lookupSmall(10)
	if val2.Cmp(expected) != 0 {
		t.Fatalf("Violation d'immuabilité : LUT modifiée. F(10) attendu 55, obtenu %s", val2.String())
	}
}

// TestNilCoreCalculatorPanic vérifie que `NewCalculator` panique si appelé
// avec un `coreCalculator` nul.
func TestNilCoreCalculatorPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCalculator aurait dû paniquer avec un noyau nil.")
		}
	}()
	_ = NewCalculator(nil)
}

// TestProgressReporter valide la notification monotone de la progression.
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
						t.Errorf("Progression non-monotone. Précédent: %f, Actuel: %f", lastProgress, update.Value)
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
				t.Errorf("Progression finale attendue 1.0, obtenue %f", lastProgress)
			}
		})
	}
}

// TestContextCancellation vérifie la réactivité des algorithmes à une
// annulation de contexte.
func TestContextCancellation(t *testing.T) {
	calculators := map[string]Calculator{
		"FastDoubling": NewCalculator(&OptimizedFastDoubling{}),
		"MatrixExp":    NewCalculator(&MatrixExponentiation{}),
	}

	for name, calc := range calculators {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			_, err := calc.Calculate(ctx, nil, 0, 100_000_000, DefaultParallelThreshold, 0)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Erreur attendue : %v, Obtenue : %v", context.DeadlineExceeded, err)
			}
		})
	}
}

// TestFibonacciProperties utilise des tests basés sur les propriétés pour
// valider l'identité de Cassini pour un large éventail d'entrées.
func TestFibonacciProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	uint64Gen := gen.UInt64Range(1, 2000)
	properties := gopter.NewProperties(parameters)

	properties.Property("Identité de Cassini", prop.ForAll(
		func(n uint64) bool {
			calc := NewCalculator(&OptimizedFastDoubling{})
			ctx := context.Background()

			var f_n_minus_1, f_n, f_n_plus_1 *big.Int
			var g errgroup.Group
			g.Go(func() error { var err error; f_n_minus_1, err = calc.Calculate(ctx, nil, 0, n-1, DefaultParallelThreshold, 0); return err })
			g.Go(func() error { var err error; f_n, err = calc.Calculate(ctx, nil, 0, n, DefaultParallelThreshold, 0); return err })
			g.Go(func() error { var err error; f_n_plus_1, err = calc.Calculate(ctx, nil, 0, n+1, DefaultParallelThreshold, 0); return err })

			if err := g.Wait(); err != nil {
				t.Logf("Échec du calcul pour n=%d : %v", n, err)
				return false
			}

			term1 := new(big.Int).Mul(f_n_minus_1, f_n_plus_1)
			term2 := new(big.Int).Mul(f_n, f_n)
			leftSide := new(big.Int).Sub(term1, term2)

			rightSide := big.NewInt(1)
			if n%2 != 0 {
				rightSide.Neg(rightSide)
			}

			return leftSide.Cmp(rightSide) == 0
		},
		uint64Gen,
	))

	properties.TestingRun(t)
}

func runBenchmark(b *testing.B, calc Calculator, n uint64) {
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

// ExampleCalculator_Calculate illustre l'utilisation de base d'un Calculator
// pour calculer un nombre de Fibonacci.
func ExampleCalculator_Calculate() {
	// Crée un nouveau calculateur avec l'algorithme Fast Doubling.
	calculator := NewCalculator(&OptimizedFastDoubling{})

	// Calcule le 20ème nombre de Fibonacci.
	result, err := calculator.Calculate(context.Background(), nil, 0, 20, DefaultParallelThreshold, 0)
	if err != nil {
		fmt.Printf("Erreur de calcul : %v\n", err)
		return
	}

	fmt.Println(result)
	// Output: 6765
}