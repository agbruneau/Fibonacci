package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/bits"
	"sort"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Types et structures optimisées
// ------------------------------------------------------------
type fibFunc func(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)

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

// ------------------------------------------------------------
// Constantes précalculées pour Binet
// ------------------------------------------------------------
var (
	phi   = big.NewFloat(1.61803398874989484820458683436563811772030917980576286214)
	sqrt5 = big.NewFloat(2.23606797749978969640917366873127623544061835961152572427)
)

// ------------------------------------------------------------
// Progression optimisée avec pooling
// ------------------------------------------------------------
func progressPrinter(progress <-chan struct {
	name string
	pct  float64
}) {
	status := make(map[string]float64)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	lastPrintTime := time.Now()
	needsUpdate := true

	// Pré-tri des clés pour éviter le tri répétitif
	keys := []string{"Fast-doubling", "Binet", "Matrice 2x2"}

	for {
		select {
		case p, ok := <-progress:
			if !ok {
				printStatus(status, keys)
				fmt.Println()
				return
			}
			if status[p.name] != p.pct {
				status[p.name] = p.pct
				needsUpdate = true
			}
		case <-ticker.C:
			if needsUpdate || time.Since(lastPrintTime) > 500*time.Millisecond {
				printStatus(status, keys)
				needsUpdate = false
				lastPrintTime = time.Now()
			}
		}
	}
}

func printStatus(status map[string]float64, keys []string) {
	fmt.Print("\r")
	first := true
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

// ------------------------------------------------------------
// Pool de big.Int pour réutilisation
// ------------------------------------------------------------
func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// ------------------------------------------------------------
// Algorithmes optimisés avec réutilisation de mémoire
// ------------------------------------------------------------
func fibBinet(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	prec := uint(float64(n)*math.Log2(1.61803398875) + 10)
	phiPrec := new(big.Float).SetPrec(prec).Set(phi)
	sqrt5Prec := new(big.Float).SetPrec(prec).Set(sqrt5)

	totalSteps := bits.Len(uint(n))
	pow := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phiPrec)
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

	pow.Quo(pow, sqrt5Prec)
	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	pow.Add(pow, half)
	z := new(big.Int)
	pow.Int(z)

	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n < 2 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	a := big.NewInt(0)
	b := big.NewInt(1)
	totalBits := bits.Len(uint(n))

	for i := totalBits - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Utilisation du pool pour les opérations temporaires
		a2 := pool.Get().(*big.Int).Set(a)
		b2 := pool.Get().(*big.Int).Set(b)
		twoB := pool.Get().(*big.Int).Lsh(b, 1)
		twoBsubA := pool.Get().(*big.Int).Sub(twoB, a)
		c := pool.Get().(*big.Int).Mul(a, twoBsubA)
		d := pool.Get().(*big.Int).Add(a2.Mul(a2, a2), b2.Mul(b2, b2))

		a.Set(c)
		b.Set(d)

		// Libération des ressources du pool après utilisation
		pool.Put(a2)
		pool.Put(b2)
		pool.Put(twoB)
		pool.Put(twoBsubA)
		pool.Put(c)
		pool.Put(d)

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

// ------------------------------------------------------------
// Matrice optimisée avec réutilisation
// ------------------------------------------------------------
type mat2 struct {
	a, b, c, d *big.Int
}

func matMul(x, y *mat2, pool *sync.Pool) *mat2 {
	xa, xb, xc, xd := x.a, x.b, x.c, x.d
	ya, yb, yc, yd := y.a, y.b, y.c, y.d

	temp := make([]*big.Int, 4)
	for i := range temp {
		temp[i] = pool.Get().(*big.Int)
	}

	newA := new(big.Int).Add(temp[0].Mul(xa, ya), temp[1].Mul(xb, yc))
	newB := new(big.Int).Add(temp[0].Mul(xa, yb), temp[1].Mul(xb, yd))
	newC := new(big.Int).Add(temp[2].Mul(xc, ya), temp[3].Mul(xd, yc))
	newD := new(big.Int).Add(temp[0].Mul(xc, yb), temp[1].Mul(xd, yd))

	for i := range temp {
		pool.Put(temp[i])
	}

	return &mat2{a: newA, b: newB, c: newC, d: newD}
}

func fibMatrix(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
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
			res = matMul(res, base, pool)
		}
		if exp > 1 {
			base = matMul(base, base, pool)
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

// ------------------------------------------------------------
// Main avec améliorations
// ------------------------------------------------------------
func main() {
	// Go gère automatiquement le parallélisme depuis Go 1.5+
	// runtime.GOMAXPROCS(runtime.NumCPU()) // Désactivé car non nécessaire

	n := flag.Int("n", 20_000_000, "Indice n du terme de Fibonacci (≥0)")
	timeout := flag.Duration("timeout", 2*time.Minute, "Durée maximale d'exécution globale")
	flag.Parse()

	if *n < 0 {
		log.Fatalf("L'indice n doit être supérieur ou égal à 0. Reçu: %d", *n)
	}

	log.Printf("Calcul de F(%d) avec un timeout de %v...\n", *n, *timeout)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Pool de big.Int pour réduire les allocations mémoire
	intPool := newIntPool()

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
			v, err := currentTask.fn(ctx, localProg, *n, intPool)
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
	} else {
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

func printFibResultDetails(value *big.Int, n int, duration time.Duration) {
	digits := len(value.Text(10))
	fmt.Printf("F(%d) calculé en %v\n", n, duration.Round(time.Millisecond))
	fmt.Printf("Nombre de chiffres : %d\n", digits)

	if digits > 20 {
		sci := new(big.Float).SetInt(value).Text('e', 8)
		fmt.Printf("F(%d) ≈ %s (notation scientifique)\n", n, sci)
	} else {
		fmt.Printf("F(%d) = %s\n", n, value.Text(10))
	}
}
