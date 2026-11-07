// Seam 1 : la bibliothèque. Les attendus viennent de l'oracle (contracts/b000045.txt) ou de faits mathématiques.
package fib_test

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"fib"
)

var (
	cachedText  []string   // l'oracle, lu une fois (les tests ne sont pas parallèles)
	cachedTerms []*big.Int // les mêmes termes en entiers
)

var algos = []string{"iterative", "matrix", "fast"}

// oracleText rend F(0) … F(2000) tels qu'écrits dans contracts/b000045.txt. Les attendus ne sont jamais recalculés ici.
// Un oracle absent, mal formé ou sans exactement 2 001 termes fait échouer le test qui l'utilise.
func oracleText(t *testing.T) []string {
	t.Helper()
	if cachedText != nil {
		return cachedText
	}
	data, err := os.ReadFile(filepath.Join("..", "contracts", "b000045.txt"))
	if err != nil {
		t.Fatalf("oracle illisible : %v", err)
	}
	var texts []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == "" {
			continue
		}
		index, digits, found := strings.Cut(line, " ")
		if !found || index != strconv.Itoa(len(texts)) {
			t.Fatalf("oracle mal formé à la ligne %q", line)
		}
		texts = append(texts, digits)
	}
	if len(texts) != 2001 {
		t.Fatalf("oracle : %d termes, attendu 2001", len(texts))
	}
	cachedText = texts
	return texts
}

// oracle rend les termes de oracleText en entiers.
func oracle(t *testing.T) []*big.Int {
	t.Helper()
	if cachedTerms != nil {
		return cachedTerms
	}
	texts := oracleText(t)
	terms := make([]*big.Int, len(texts))
	for i, digits := range texts {
		term, ok := new(big.Int).SetString(digits, 10)
		if !ok {
			t.Fatalf("oracle mal formé : le terme %d n'est pas décimal : %q", i, digits)
		}
		terms[i] = term
	}
	cachedTerms = terms
	return terms
}

// mustFib rend Fib(n, algo), ou arrête le test.
func mustFib(t *testing.T, n int, algo string) *big.Int {
	t.Helper()
	got, err := fib.Fib(n, algo)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// mustFibList rend FibList(n), qui doit compter n + 1 termes, ou arrête le test.
func mustFibList(t *testing.T, n int) []string {
	t.Helper()
	list, err := fib.FibList(n)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != n+1 {
		t.Fatalf("FibList(%d) a %d termes, attendu %d", n, len(list), n+1)
	}
	return list
}

// check compare Fib(n, algo) au terme n de l'oracle.
func check(t *testing.T, n int, algo string) {
	t.Helper()
	want := oracle(t)[n]
	got := mustFib(t, n, algo)
	if got.Cmp(want) != 0 {
		t.Errorf("Fib(%d, %s) = %v, attendu %v", n, algo, got, want)
	}
}

// BIB-01 à BIB-04 : les premiers termes par l'itératif.
func TestBIB01a04_IterativePremiersTermes(t *testing.T) {
	for _, c := range []struct {
		id string
		n  int
	}{{"BIB-01", 0}, {"BIB-02", 1}, {"BIB-03", 2}, {"BIB-04", 10}} {
		t.Run(fmt.Sprintf("%s : Fib(%d, iterative) vaut le terme de l'oracle", c.id, c.n), func(t *testing.T) {
			check(t, c.n, "iterative")
		})
	}
}

// BIB-05 à BIB-07 : chaque algorithme égale l'oracle de 0 à 2000. Le balayage franchit int64 et uint64 (92 à 94).
func TestBIB05a07_BalayageContreOracle(t *testing.T) {
	oracle(t) // un oracle absent arrête le balayage ici, au lieu de produire 6 000 échecs identiques
	for _, c := range []struct{ id, algo string }{{"BIB-05", "iterative"}, {"BIB-06", "fast"}, {"BIB-07", "matrix"}} {
		for n := 0; n <= 2000; n++ {
			t.Run(fmt.Sprintf("%s : Fib(%d, %s) vaut le terme de l'oracle", c.id, n, c.algo), func(t *testing.T) {
				check(t, n, c.algo)
			})
		}
	}
}

// BIB-08 : les trois algorithmes rendent le même terme pour n = 100 000, qui compte 20 899 chiffres.
func TestBIB08_LesAlgorithmesConcordentA100000(t *testing.T) {
	reference := mustFib(t, 100000, "iterative")
	t.Run("BIB-08 : Fib(100000, iterative) compte 20899 chiffres", func(t *testing.T) {
		if got := len(reference.String()); got != 20899 {
			t.Errorf("Fib(100000, iterative) compte %d chiffres, attendu 20899", got)
		}
	})
	for _, algo := range []string{"matrix", "fast"} {
		t.Run(fmt.Sprintf("BIB-08 : Fib(100000, %s) égale Fib(100000, iterative)", algo), func(t *testing.T) {
			if got := mustFib(t, 100000, algo); got.Cmp(reference) != 0 {
				t.Errorf("Fib(100000, %s) diffère de Fib(100000, iterative)", algo)
			}
		})
	}
}

// BIB-09 : identité de Cassini, F(n-1)·F(n+1) - F(n)² = (-1)^n, pour n = 100 000.
func TestBIB09_CassiniAutourDe100000(t *testing.T) {
	for _, algo := range algos {
		t.Run(fmt.Sprintf("BIB-09 : %s respecte l'identité de Cassini à n = 100 000", algo), func(t *testing.T) {
			before, at, after := mustFib(t, 99999, algo), mustFib(t, 100000, algo), mustFib(t, 100001, algo)
			cassini := new(big.Int).Mul(before, after)
			cassini.Sub(cassini, new(big.Int).Mul(at, at))
			if cassini.Cmp(big.NewInt(1)) != 0 {
				t.Errorf("F(99999)·F(100001) - F(100000)² = %v, attendu 1", cassini)
			}
		})
	}
}

// BIB-10 à BIB-12 : FibList(n) égale, en chaînes décimales, les n + 1 premiers termes de l'oracle, dans l'ordre.
func TestBIB10a12_FibListEgaleLOracle(t *testing.T) {
	for _, c := range []struct {
		id string
		n  int
	}{{"BIB-10", 0}, {"BIB-11", 1}, {"BIB-12", 2000}} {
		t.Run(fmt.Sprintf("%s : FibList(%d) vaut les %d premiers termes de l'oracle", c.id, c.n, c.n+1), func(t *testing.T) {
			want, got := oracleText(t)[:c.n+1], mustFibList(t, c.n)
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("FibList(%d)[%d] = %s, attendu %s", c.n, i, got[i], want[i])
				}
			}
		})
	}
}

// PERF-GO-1, hors catalogue : garde de l'optimisation de FibList (base 10^18) au-delà de l'oracle, contre Fib(k, fast).
func TestPERFGO1_FibListAuDelaDeLOracle(t *testing.T) {
	t.Run("PERF-GO-1 : FibList(10000) égale Fib(k, fast) en décimal pour chaque k", func(t *testing.T) {
		for k, got := range mustFibList(t, 10000) {
			if want := mustFib(t, k, "fast").String(); got != want {
				t.Fatalf("FibList(10000)[%d] = %s, attendu %s", k, got, want)
			}
		}
	})
}

// PERF-GO-2, hors catalogue : garde de l'optimisation de FibList (réserve de limbes d'avance), pour n de 0 à 300.
func TestPERFGO2_FibListReserveDeLimbes(t *testing.T) {
	texts := oracleText(t) // un oracle absent arrête le test ici, au lieu de produire 301 échecs identiques
	for n := 0; n <= 300; n++ {
		t.Run(fmt.Sprintf("PERF-GO-2 : le dernier terme de FibList(%d) vaut celui de l'oracle", n), func(t *testing.T) {
			if got := mustFibList(t, n)[n]; got != texts[n] {
				t.Errorf("FibList(%d)[%d] = %s, attendu %s", n, n, got, texts[n])
			}
		})
	}
}

// Bornes d'allocations de PERF-GO-3 et PERF-GO-4 : un chronomètre serait instable, ce compte est exact.
// Observé : 4 pour iterative, 6 pour FibList, sans marge; le mutant d'une réserve trop courte en lit 10.
// Sur 3 passes, FibList a lu 8 une fois sur 30 : une allocation parasite, divisée par 3. Sur 20, elle est absorbée.
const (
	maxAllocsIterative = 4
	maxAllocsFibList   = 6
	allocsRuns         = 20
)

// PERF-GO-3, hors catalogue : garde de l'optimisation d'iterative (addition sur place) : pas d'allocation par pas.
func TestPERFGO3_IterativeNAllouePasParPas(t *testing.T) {
	t.Run(fmt.Sprintf("PERF-GO-3 : Fib(20000, iterative) alloue au plus %d fois", maxAllocsIterative), func(t *testing.T) {
		if got := testing.AllocsPerRun(allocsRuns, func() { mustFib(t, 20000, "iterative") }); got > maxAllocsIterative {
			t.Errorf("Fib(20000, iterative) alloue %v fois, attendu %d au plus", got, maxAllocsIterative)
		}
	})
}

// PERF-GO-4, hors catalogue : garde de l'optimisation de FibList (addition sur place) : pas d'allocation par pas.
func TestPERFGO4_FibListNAllouePasParPas(t *testing.T) {
	t.Run(fmt.Sprintf("PERF-GO-4 : FibList(5000) alloue au plus %d fois", maxAllocsFibList), func(t *testing.T) {
		if got := testing.AllocsPerRun(allocsRuns, func() { mustFibList(t, 5000) }); got > maxAllocsFibList {
			t.Errorf("FibList(5000) alloue %v fois, attendu %d au plus", got, maxAllocsFibList)
		}
	})
}

// BIB-13 : Fib refuse tout nom qui n'est pas exactement iterative, matrix ou fast.
// `all` est la comparaison de la CLI, pas un algorithme.
func TestBIB13_AlgorithmeInconnuEstRefuse(t *testing.T) {
	for _, algo := range []string{"all", "bogus", "", "FAST"} {
		t.Run(fmt.Sprintf("BIB-13 : Fib(5, %q) est refusé", algo), func(t *testing.T) {
			if got, err := fib.Fib(5, algo); err == nil {
				t.Errorf("Fib(5, %q) = %v, attendu une erreur", algo, got)
			}
		})
	}
}

// BIB-14 : un indice négatif est refusé (l'indice est un int signé, donc 1.5 et NaN ne peuvent pas être passés).
func TestBIB14_IndiceNegatifEstRefuse(t *testing.T) {
	for _, algo := range algos {
		t.Run(fmt.Sprintf("BIB-14 : Fib(-1, %s) est refusé", algo), func(t *testing.T) {
			if got, err := fib.Fib(-1, algo); err == nil {
				t.Errorf("Fib(-1, %s) = %v, attendu une erreur", algo, got)
			}
		})
	}
	t.Run("BIB-14 : FibList(-1) est refusé", func(t *testing.T) {
		if got, err := fib.FibList(-1); err == nil {
			t.Errorf("FibList(-1) = %v, attendu une erreur", got)
		}
	})
}
