// main.go
// Version 8.8 — Calcul concurrent de Fibonacci (Go 1.22) - Suppression affichage nombre de chiffres
// Auteur : André-Guy Bruneau — 17 avril 2025 (modifié)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/bits"
	"runtime"
	"sort"
	"sync"
	"time"
)

//------------------------------------------------------------
// Types et structures
//------------------------------------------------------------

type fibFunc func(ctx context.Context, progress chan<- float64, n int) (*big.Int, error)

type task struct {
	name string
	fn   fibFunc
}

type result struct {
	name     string
	value    *big.Int
	duration time.Duration
	err      error
}

//------------------------------------------------------------
// Constantes pour la formule de Binet
//------------------------------------------------------------

const (
	phiStr   = "1.61803398874989484820458683436563811772030917980576286214"
	sqrt5Str = "2.23606797749978969640917366873127623544061835961152572427"
)

//------------------------------------------------------------
// Affichage de la progression
//------------------------------------------------------------

func progressPrinter(progress <-chan struct {
	name string
	pct  float64
}) {
	status := make(map[string]float64)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	lastPrintTime := time.Now()
	needsUpdate := true

	for {
		select {
		case p, ok := <-progress:
			if !ok {
				printStatus(status)
				fmt.Println()
				return
			}
			if status[p.name] != p.pct {
				status[p.name] = p.pct
				needsUpdate = true
			}
		case <-ticker.C:
			if needsUpdate || time.Since(lastPrintTime) > 500*time.Millisecond {
				printStatus(status)
				needsUpdate = false
				lastPrintTime = time.Now()
			}
		}
	}
}

func printStatus(status map[string]float64) {
	fmt.Print("\r")
	first := true
	keys := make([]string, 0, len(status))
	for k := range status {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := status[k]
		if !first {
			fmt.Print("   ")
		}
		fmt.Printf("%-14s %6.2f%%", k, v)
		first = false
	}
	fmt.Print("                 ")
}

//------------------------------------------------------------
// Outils communs
//------------------------------------------------------------

func newFloat(s string, prec uint) *big.Float {
	f, _, err := big.ParseFloat(s, 10, prec, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	return f
}

//------------------------------------------------------------
// Algorithmes de calcul de Fibonacci
//------------------------------------------------------------

// --- 1. Formule de Binet ---
func fibBinet(ctx context.Context, progress chan<- float64, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n == 0 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(0), nil
	}
	if n == 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(1), nil
	}

	prec := uint(float64(n)*math.Log2(1.61803398875) + 10)
	phi := newFloat(phiStr, prec)
	sqrt5 := newFloat(sqrt5Str, prec)

	totalSteps := bits.Len(uint(n))
	pow := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phi)
	stepsDone := 0

	for exp := uint(n); exp > 0; exp >>= 1 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if exp&1 == 1 {
			pow.Mul(pow, base)
		}
		if exp > 1 {
			base.Mul(base, base)
		}
		stepsDone++
		if progress != nil && totalSteps > 0 {
			progress <- float64(stepsDone) / float64(totalSteps) * 100.0
		}
	}

	pow.Quo(pow, sqrt5)
	half := big.NewFloat(0.5).SetPrec(prec)
	pow.Add(pow, half)

	z := new(big.Int)
	pow.Int(z)

	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

// --- 2. Fast-doubling ---
func fibFastDoubling(ctx context.Context, progress chan<- float64, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n < 2 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	a, b := big.NewInt(0), big.NewInt(1)
	totalBits := bits.Len(uint(n))

	for i := totalBits - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		a2 := new(big.Int).Mul(a, a)
		b2 := new(big.Int).Mul(b, b)
		twoB := new(big.Int).Lsh(b, 1)
		twoBsubA := new(big.Int).Sub(twoB, a)
		c := new(big.Int).Mul(a, twoBsubA)
		d := new(big.Int).Add(a2, b2)
		a.Set(c)
		b.Set(d)
		if (uint(n)>>i)&1 == 1 {
			a.Set(d)
			b.Add(c, d)
		}
		if progress != nil && totalBits > 0 {
			progress <- float64(totalBits-i) / float64(totalBits) * 100.0
		}
	}
	if progress != nil {
		progress <- 100.0
	}
	return a, nil
}

// --- 3. Matrice 2x2 ---
type mat2 struct{ a, b, c, d *big.Int }

func matMul(x, y *mat2) *mat2 {
	xa, xb, xc, xd := x.a, x.b, x.c, x.d
	ya, yb, yc, yd := y.a, y.b, y.c, y.d
	newA := new(big.Int).Add(new(big.Int).Mul(xa, ya), new(big.Int).Mul(xb, yc))
	newB := new(big.Int).Add(new(big.Int).Mul(xa, yb), new(big.Int).Mul(xb, yd))
	newC := new(big.Int).Add(new(big.Int).Mul(xc, ya), new(big.Int).Mul(xd, yc))
	newD := new(big.Int).Add(new(big.Int).Mul(xc, yb), new(big.Int).Mul(xd, yd))
	return &mat2{a: newA, b: newB, c: newC, d: newD}
}
func fibMatrix(ctx context.Context, progress chan<- float64, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n < 2 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}
	res := &mat2{a: big.NewInt(1), b: big.NewInt(0), c: big.NewInt(0), d: big.NewInt(1)}
	base := &mat2{a: big.NewInt(1), b: big.NewInt(1), c: big.NewInt(1), d: big.NewInt(0)}
	exp := uint(n - 1)
	totalSteps := bits.Len(exp)
	stepsDone := 0
	for ; exp > 0; exp >>= 1 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if exp&1 == 1 {
			res = matMul(res, base)
		}
		if exp > 1 {
			base = matMul(base, base)
		}
		stepsDone++
		if progress != nil && totalSteps > 0 {
			progress <- float64(stepsDone) / float64(totalSteps) * 100.0
		}
	}
	if progress != nil {
		progress <- 100.0
	}
	return new(big.Int).Set(res.a), nil
}

//------------------------------------------------------------
// main
//------------------------------------------------------------

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	n := flag.Int("n", 10_000_000, "Indice n du terme de Fibonacci (≥0)")
	timeout := flag.Duration("timeout", 2*time.Minute, "Durée maximale d'exécution globale")
	flag.Parse()

	if *n < 0 {
		log.Fatalf("L'indice n doit être supérieur ou égal à 0. Reçu: %d", *n)
	}

	log.Printf("Calcul de F(%d) avec un timeout de %v...\n", *n, *timeout)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	tasks := []task{
		{"Fast-doubling", fibFastDoubling},
		{"Binet", fibBinet},
		{"Matrice 2x2", fibMatrix},
	}

	progressCh := make(chan struct {
		name string
		pct  float64
	}, len(tasks)*2)
	go progressPrinter(progressCh)

	var wg sync.WaitGroup
	var relayWg sync.WaitGroup
	resCh := make(chan result, len(tasks))

	log.Println("Lancement des calculs concurrents...")
	for _, t := range tasks {
		wg.Add(1)
		go func(currentTask task) {
			defer wg.Done()
			localProg := make(chan float64, 8)
			relayWg.Add(1)
			go func() {
				defer relayWg.Done()
				for p := range localProg {
					select {
					case progressCh <- struct {
						name string
						pct  float64
					}{currentTask.name, p}:
					default:
					}
				}
			}()
			start := time.Now()
			v, err := currentTask.fn(ctx, localProg, *n)
			duration := time.Since(start)
			close(localProg)
			resCh <- result{currentTask.name, v, duration, err}
		}(t)
	}

	wg.Wait()
	log.Println("Goroutines de calcul terminées.")
	relayWg.Wait()
	log.Println("Goroutines relais de progression terminées.")
	close(progressCh)

	results := make([]result, 0, len(tasks))
	timeoutOccurred := false
	for i := 0; i < len(tasks); i++ {
		r := <-resCh
		if r.err != nil {
			if r.err == context.DeadlineExceeded {
				log.Printf("⚠️ Tâche '%s' interrompue par timeout (%v) après %v", r.name, *timeout, r.duration)
				timeoutOccurred = true
			} else {
				log.Printf("❌ Erreur pour la tâche '%s' : %v (durée: %v)", r.name, r.err, r.duration)
			}
		}
		results = append(results, r)
	}
	close(resCh)

	if timeoutOccurred {
		log.Println("Comparaison des résultats ignorée en raison d'un timeout.")
		fmt.Println("\n--------------------------- TEMPS D'EXÉCUTION (Timeout) ---------------------------")
		sort.Slice(results, func(i, j int) bool { return results[i].name < results[j].name })
		for _, r := range results {
			status := "Terminé"
			if r.err == context.DeadlineExceeded {
				status = "Timeout"
			} else if r.err != nil {
				status = fmt.Sprintf("Erreur (%v)", r.err)
			}
			fmt.Printf("%-15s : %-10v (%s)\n", r.name, r.duration.Round(time.Microsecond), status)
		}
		return
	}

	//--------------------------------------------------------
	// Affichage et Vérification (si aucun timeout)
	//--------------------------------------------------------
	sort.Slice(results, func(i, j int) bool { return results[i].duration < results[j].duration })

	equal := true
	firstValidResult := -1
	for i := range results {
		if results[i].err == nil && results[i].value != nil {
			firstValidResult = i
			break
		}
	}

	if firstValidResult == -1 {
		log.Println("Aucun résultat valide n'a été produit par les algorithmes.")
		equal = false
	} else {
		baseValue := results[firstValidResult].value
		for i := firstValidResult + 1; i < len(results); i++ {
			if results[i].err == nil && results[i].value != nil {
				if results[i].value.Cmp(baseValue) != 0 {
					equal = false
					// log.Printf("Différence détectée...") // Déjà supprimé
					break
				}
			} else if results[i].err != nil {
				log.Printf("Note : La tâche '%s' a échoué, la comparaison d'égalité peut être incomplète.", results[i].name)
			}
		}
	}

	fmt.Println("\n--------------------------- RÉSULTATS ---------------------------")
	for _, r := range results {
		status := "OK"
		if r.err != nil {
			status = fmt.Sprintf("Erreur: %v", r.err)
		}
		fmt.Printf("%-15s : %-12v [%s]\n", r.name, r.duration.Round(time.Microsecond), status)
	}

	if firstValidResult != -1 && results[0].err == nil {
		fmt.Printf("Algorithme le plus rapide : %s\n", results[0].name)
		if equal {
			fmt.Println("✅ Tous les résultats valides sont identiques.")
		}
		printFibResultDetails(results[0].value, *n, results[0].duration)
	} else if firstValidResult == -1 {
		fmt.Println("Aucun algorithme n'a terminé avec succès.")
	} else { // Le plus rapide a échoué, chercher le suivant
		foundWinner := false
		for _, r := range results {
			if r.err == nil && r.value != nil {
				fmt.Printf("Algorithme le plus rapide ayant réussi : %s (%v)\n", r.name, r.duration.Round(time.Microsecond))
				if equal {
					fmt.Println("✅ Tous les résultats valides sont identiques.")
				}
				printFibResultDetails(r.value, *n, r.duration)
				foundWinner = true
				break
			}
		}
		if !foundWinner {
			fmt.Println("Aucun algorithme n'a réussi.")
		}
	}
}

// Fonction utilitaire pour afficher les détails du résultat F(n)
func printFibResultDetails(value *big.Int, n int, duration time.Duration) {
	// Calculer le nombre de chiffres (nécessaire pour la condition suivante)
	digits := len(value.Text(10))
	// Afficher le temps de calcul
	fmt.Printf("F(%d) calculé en %v\n", n, duration.Round(time.Millisecond))
	// fmt.Printf("Nombre de chiffres : %d\n", digits) // <-- LIGNE SUPPRIMÉE

	// Afficher en notation scientifique ou complète selon la taille
	if digits > 20 {
		sci := new(big.Float).SetInt(value).Text('e', 8)
		fmt.Printf("F(%d) ≈ %s (notation scientifique)\n", n, sci)
	} else {
		fmt.Printf("F(%d) = %s\n", n, value.Text(10))
	}
}
