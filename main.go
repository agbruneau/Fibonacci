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

// main.go
//
// Ce programme calcule le n-ième nombre de Fibonacci en utilisant trois algorithmes différents:
// 1. Formule de Binet (utilisant des big.Float pour la précision).
// 2. Algorithme de Doubling Rapide (Fast Doubling).
// 3. Algorithme d'exponentiation matricielle (Matrice 2x2).
//
// Il exécute ces algorithmes en concurrence, affiche leur progression,
// et compare leurs temps d'exécution ainsi que leurs résultats.
// Un pool de sync.Pool est utilisé pour réduire les allocations mémoire des objets big.Int.
//
// Utilisation:
//   go run main.go -n <indice> -timeout <durée>
// Exemple:
//   go run main.go -n 100000 -timeout 1m

// ------------------------------------------------------------
// Types et structures optimisées
// ------------------------------------------------------------

// fibFunc est un type pour les fonctions calculant Fibonacci.
// Elle prend un contexte pour l'annulation, un canal pour la progression,
// l'indice n, et un pool d'objets big.Int pour la réutilisation mémoire.
type fibFunc func(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)

// task représente une tâche de calcul de Fibonacci à exécuter.
type task struct {
	name string  // Nom de l'algorithme
	fn   fibFunc // Fonction de l'algorithme
}

// result stocke le résultat d'une tâche de calcul.
type result struct {
	name     string        // Nom de l'algorithme
	value    *big.Int      // Valeur de Fibonacci calculée
	duration time.Duration // Durée du calcul
	err      error         // Erreur éventuelle
}

// ------------------------------------------------------------
// Constantes précalculées pour Binet (utilisées comme base pour une précision dynamique)
// ------------------------------------------------------------
var (
	// phi est la valeur de base du nombre d'or.
	phi = big.NewFloat(1.61803398874989484820458683436563811772030917980576286214)
	// sqrt5 est la valeur de base de la racine carrée de 5.
	sqrt5 = big.NewFloat(2.23606797749978969640917366873127623544061835961152572427)
)

// ------------------------------------------------------------
// Gestion de l'affichage de la progression
// ------------------------------------------------------------

// progressData encapsule les informations de progression pour une tâche.
type progressData struct {
	name string  // Nom de la tâche
	pct  float64 // Pourcentage de progression
}

// progressPrinter gère l'affichage consolidé de la progression de toutes les tâches.
// Il rafraîchit l'affichage à intervalle régulier ou lors de nouvelles données de progression.
func progressPrinter(progress <-chan progressData) {
	status := make(map[string]float64)
	ticker := time.NewTicker(100 * time.Millisecond) // Rafraîchissement minimal pour l'interface
	defer ticker.Stop()
	lastPrintTime := time.Now()
	needsUpdate := true

	// Pré-tri des clés pour un affichage ordonné et constant.
	// Important: ces noms doivent correspondre aux task.name.
	keys := []string{"Fast-doubling", "Binet", "Matrice 2x2"}
	// Initialiser le statut pour toutes les clés pour un affichage complet dès le début
	for _, k := range keys {
		status[k] = 0.0
	}

	for {
		select {
		case p, ok := <-progress:
			if !ok { // Canal fermé, fin de la progression
				printStatus(status, keys)
				fmt.Println() // Nouvelle ligne finale
				return
			}
			if status[p.name] != p.pct {
				status[p.name] = p.pct
				needsUpdate = true
			}
		case <-ticker.C:
			// Rafraîchir si une mise à jour est nécessaire ou si un certain temps s'est écoulé,
			// pour montrer que le programme est toujours actif même si les pourcentages ne changent pas rapidement.
			if needsUpdate || time.Since(lastPrintTime) > 500*time.Millisecond {
				printStatus(status, keys)
				needsUpdate = false // Réinitialiser après l'impression
				lastPrintTime = time.Now()
			}
		}
	}
}

// printStatus affiche l'état actuel de la progression pour chaque tâche.
func printStatus(status map[string]float64, keys []string) {
	fmt.Print("\r") // Retour chariot pour effacer la ligne précédente
	first := true
	for _, k := range keys {
		v, ok := status[k]
		if !ok {
			continue
		} // Au cas où une tâche n'a pas encore envoyé de statut
		if !first {
			fmt.Print("   ")
		}
		fmt.Printf("%-14s %6.2f%%", k, v)
		first = false
	}
	fmt.Print("                 ") // Espaces pour effacer les restes de la ligne précédente
}

// ------------------------------------------------------------
// Pool de big.Int pour réutilisation mémoire
// ------------------------------------------------------------

// newIntPool crée un nouveau sync.Pool pour les objets *big.Int.
// Cela aide à réduire la pression sur le garbage collector en réutilisant des objets.
func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// ------------------------------------------------------------
// Algorithmes de calcul de Fibonacci
// ------------------------------------------------------------

// fibBinet calcule F(n) en utilisant la formule de Binet.
// F(n) = (phi^n - (-phi)^-n) / sqrt(5)
// Pour n grand, cela se simplifie à F(n) ≈ round(phi^n / sqrt(5)).
// Note: Cet algorithme utilise big.Float et n'utilise donc pas activement le pool de big.Int,
// car la majorité des allocations concerne les flottants à haute précision.
// Le pool est passé par cohérence avec la signature fibFunc.
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

	// La précision nécessaire augmente avec n.
	// bits pour phi^n ≈ n * log2(phi)
	// Ajout d'une marge de sécurité (+10 ou plus) pour la précision.
	phiVal, _ := phi.Float64() // Récupérer la valeur float64 de phi, ignorer l'indicateur de précision.
	prec := uint(float64(n)*math.Log2(phiVal) + 10)

	phiPrec := new(big.Float).SetPrec(prec).Set(phi)
	sqrt5Prec := new(big.Float).SetPrec(prec).Set(sqrt5)

	// Calcul de phi^n par exponentiation binaire (exponentiation by squaring)
	// pour minimiser le nombre de multiplications.
	powN := new(big.Float).SetPrec(prec)

	numBitsInN := bits.Len(uint(n))
	currentStep := 0

	phiToN := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phiPrec)

	exponent := uint(n)
	for exponent > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exponent&1 == 1 {
			phiToN.Mul(phiToN, base)
		}
		base.Mul(base, base)
		exponent >>= 1

		currentStep++
		if progress != nil && numBitsInN > 0 {
			progress <- (float64(currentStep) / float64(numBitsInN)) * 100.0
		}
	}
	powN = phiToN

	powN.Quo(powN, sqrt5Prec)

	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	powN.Add(powN, half)

	z := new(big.Int)
	powN.Int(z)

	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

// fibFastDoubling calcule F(n) en utilisant l'algorithme de "Fast Doubling".
// Formules:
// F(2k) = F(k) * [2*F(k+1) – F(k)]
// F(2k+1) = F(k)^2 + F(k+1)^2
// Cet algorithme est très efficace et utilise le pool de big.Int pour les calculs intermédiaires.
func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a) // Remettre 'a' dans le pool à la fin de la fonction
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b) // Remettre 'b' dans le pool à la fin de la fonction

	totalBits := bits.Len(uint(n))

	for i := totalBits - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		a_orig := pool.Get().(*big.Int).Set(a)
		// b_orig n'est pas explicitement nécessaire si on calcule d'abord a*a, puis b*b

		// F(2k) = a * (2*b - a)
		// t1 = 2*b - a
		t1 := pool.Get().(*big.Int)
		t1.Lsh(b, 1)       // t1 = 2*b
		t1.Sub(t1, a_orig) // t1 = 2*b - a  (utiliser a_orig car 'a' va être modifié)

		// new_a = a_orig * t1 = F(2k)
		new_a := pool.Get().(*big.Int)
		new_a.Mul(a_orig, t1)

		// F(2k+1) = a^2 + b^2
		// t2 = a_orig^2
		t2 := pool.Get().(*big.Int)
		t2.Mul(a_orig, a_orig)

		// t3 = b^2
		t3 := pool.Get().(*big.Int)
		t3.Mul(b, b)

		// new_b = t2 + t3 = F(2k+1)
		new_b := pool.Get().(*big.Int)
		new_b.Add(t2, t3)

		a.Set(new_a) // a devient F(2k)
		b.Set(new_b) // b devient F(2k+1)

		pool.Put(a_orig)
		pool.Put(t1)
		pool.Put(t2)
		pool.Put(t3)
		pool.Put(new_a)
		pool.Put(new_b)

		if (uint(n)>>i)&1 == 1 { // Si le i-ème bit de n est 1 (n est impair à cette étape)
			// On a F(2k) et F(2k+1). On veut F(2k+1) et F(2k+2).
			// new_a_step = F(2k+1) (qui est b actuel)
			// new_b_step = F(2k) + F(2k+1) (qui est a + b actuels)
			t_sum := pool.Get().(*big.Int).Add(a, b) // a est F(2k), b est F(2k+1)
			a.Set(b)                                 // a devient F(2k+1)
			b.Set(t_sum)                             // b devient F(2k+2)
			pool.Put(t_sum)
		}

		if progress != nil && totalBits > 0 {
			progress <- (float64(totalBits-i) / float64(totalBits)) * 100.0
		}
	}

	if progress != nil {
		progress <- 100.0
	}

	// 'a' contient F(n). Créer une copie pour le retour car 'a' appartient au pool.
	finalResult := new(big.Int).Set(a)
	return finalResult, nil
}

// mat2 représente une matrice 2x2 de *big.Int.
type mat2 struct {
	a, b, c, d *big.Int
}

// matMul effectue la multiplication de deux matrices m1 * m2.
func matMul(m1, m2 *mat2, pool *sync.Pool) *mat2 {
	t1 := pool.Get().(*big.Int)
	t2 := pool.Get().(*big.Int)
	defer pool.Put(t1)
	defer pool.Put(t2)

	// Calcul de chaque élément de la nouvelle matrice.
	// Les résultats new_a, new_b, etc., sont de nouvelles allocations.
	new_a := new(big.Int)
	t1.Mul(m1.a, m2.a) // m1.a * m2.a
	t2.Mul(m1.b, m2.c) // m1.b * m2.c
	new_a.Add(t1, t2)  // (m1.a*m2.a) + (m1.b*m2.c)

	new_b := new(big.Int)
	t1.Mul(m1.a, m2.b) // m1.a * m2.b
	t2.Mul(m1.b, m2.d) // m1.b * m2.d
	new_b.Add(t1, t2)  // (m1.a*m2.b) + (m1.b*m2.d)

	new_c := new(big.Int)
	t1.Mul(m1.c, m2.a) // m1.c * m2.a
	t2.Mul(m1.d, m2.c) // m1.d * m2.c
	new_c.Add(t1, t2)  // (m1.c*m2.a) + (m1.d*m2.c)

	new_d := new(big.Int)
	t1.Mul(m1.c, m2.b) // m1.c * m2.b
	t2.Mul(m1.d, m2.d) // m1.d * m2.d
	new_d.Add(t1, t2)  // (m1.c*m2.b) + (m1.d*m2.d)

	return &mat2{a: new_a, b: new_b, c: new_c, d: new_d}
}

// fibMatrix calcule F(n) par exponentiation de la matrice [[1,1],[1,0]].
func fibMatrix(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("indice négatif non supporté : %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	res := &mat2{
		a: new(big.Int).SetInt64(1), b: new(big.Int).SetInt64(0),
		c: new(big.Int).SetInt64(0), d: new(big.Int).SetInt64(1),
	}
	base := &mat2{
		a: new(big.Int).SetInt64(1), b: new(big.Int).SetInt64(1),
		c: new(big.Int).SetInt64(1), d: new(big.Int).SetInt64(0),
	}

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
			// Libérer les anciens Ints de res avant de réassigner si nécessaire
			// Cependant, matMul retourne une *nouvelle* mat2, donc les anciens Ints de res
			// seront récupérés par le GC si res n'est plus référencé ailleurs.
			res = matMul(res, base, pool)
		}
		if exp > 1 {
			base = matMul(base, base, pool)
		}

		stepsDone++
		if progress != nil && totalSteps > 0 {
			progress <- (float64(stepsDone) / float64(totalSteps)) * 100.0
		}
	}

	if progress != nil {
		progress <- 100.0
	}
	// Le résultat est copié pour ne pas retourner un *big.Int qui fait partie d'une matrice.
	return new(big.Int).Set(res.a), nil
}

// ------------------------------------------------------------
// Fonction principale (Main)
// ------------------------------------------------------------
func main() {
	nFlag := flag.Int("n", 30000000, "Indice n du terme de Fibonacci (entier non-négatif)")
	timeoutFlag := flag.Duration("timeout", 1*time.Minute, "Durée maximale d'exécution globale")
	flag.Parse()

	n := *nFlag
	timeout := *timeoutFlag

	if n < 0 {
		log.Fatalf("L'indice n doit être supérieur ou égal à 0. Reçu: %d", n)
	}

	log.Printf("Calcul de F(%d) avec un timeout de %v...\n", n, timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	intPool := newIntPool()

	tasks := []task{
		{"Fast-doubling", fibFastDoubling},
		{"Binet", fibBinet},
		{"Matrice 2x2", fibMatrix},
	}

	progressAggregatorCh := make(chan progressData, len(tasks)*2)
	go progressPrinter(progressAggregatorCh)

	var wg sync.WaitGroup
	var relayWg sync.WaitGroup
	resultsCh := make(chan result, len(tasks))

	log.Println("Lancement des calculs concurrents...")

	for _, t := range tasks {
		wg.Add(1)
		go func(currentTask task) {
			defer wg.Done()

			localProgCh := make(chan float64, 10)

			relayWg.Add(1)
			go func() {
				defer relayWg.Done()
				for p := range localProgCh {
					select {
					case progressAggregatorCh <- progressData{currentTask.name, p}:
					case <-ctx.Done():
						return
					}
				}
			}()

			start := time.Now()
			v, err := currentTask.fn(ctx, localProgCh, n, intPool)
			duration := time.Since(start)
			close(localProgCh)

			resultsCh <- result{currentTask.name, v, duration, err}
		}(t)
	}

	wg.Wait()
	log.Println("Goroutines de calcul terminées.")

	relayWg.Wait()
	log.Println("Goroutines relais de progression terminées.")

	close(progressAggregatorCh)

	results := make([]result, 0, len(tasks))
	timeoutOccurredOverall := false // Pour suivre si un timeout global a affecté l'ensemble
	if ctx.Err() == context.DeadlineExceeded {
		timeoutOccurredOverall = true
	}

	for i := 0; i < len(tasks); i++ {
		r := <-resultsCh
		if r.err != nil {
			if r.err == context.DeadlineExceeded {
				log.Printf("⚠️ Tâche '%s' interrompue par timeout (ou contexte annulé) après %v", r.name, r.duration.Round(time.Microsecond))
				// timeoutOccurredTask = true // Inutile si on vérifie timeoutOccurredOverall
			} else {
				log.Printf("❌ Erreur pour la tâche '%s' : %v (durée: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			}
		}
		results = append(results, r)
	}
	close(resultsCh)

	if timeoutOccurredOverall {
		log.Println("Comparaison des résultats et détails finaux peuvent être affectés car le timeout global a été atteint.")
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true
		}
		if results[i].err != nil && results[j].err == nil {
			return false
		}
		// Si les deux ont des erreurs ou les deux n'en ont pas, trier par durée
		return results[i].duration < results[j].duration
	})

	fmt.Println("\n--------------------------- RÉSULTATS ORDONNÉS ---------------------------")
	for _, r := range results {
		status := "OK"
		valStr := "N/A"
		if r.err != nil {
			status = fmt.Sprintf("Erreur: %v", r.err)
			if r.err == context.DeadlineExceeded {
				status = "Timeout/Annulé"
			}
		} else if r.value != nil {
			if len(r.value.String()) > 15 {
				valStr = r.value.String()[:5] + "..." + r.value.String()[len(r.value.String())-5:]
			} else {
				valStr = r.value.String()
			}
		}
		fmt.Printf("%-15s : %-12v [%-14s] Résultat: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	var firstSuccessfulResult *result
	allValidResultsIdentical := true
	foundSuccessful := false

	for i := range results {
		if results[i].err == nil && results[i].value != nil {
			if !foundSuccessful { // Premier résultat réussi trouvé
				firstSuccessfulResult = &results[i]
				foundSuccessful = true
				fmt.Printf("\nAlgorithme le plus rapide (ayant réussi) : %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
				printFibResultDetails(firstSuccessfulResult.value, n, firstSuccessfulResult.duration)
			} else {
				// Comparer avec le premier résultat valide trouvé.
				if results[i].value.Cmp(firstSuccessfulResult.value) != 0 {
					allValidResultsIdentical = false
					log.Printf("⚠️ DISCORDANCE ! Résultat de '%s' (%s...) différent de '%s' (%s...)",
						results[i].name, results[i].value.String()[:min(10, len(results[i].value.String()))],
						firstSuccessfulResult.name, firstSuccessfulResult.value.String()[:min(10, len(firstSuccessfulResult.value.String()))])
				}
			}
		}
	}

	if foundSuccessful {
		if allValidResultsIdentical {
			fmt.Println("✅ Tous les résultats valides produits sont identiques.")
		} else {
			fmt.Println("❌ Les résultats des algorithmes valides divergent !")
		}
	} else {
		fmt.Println("\nAucun algorithme n'a terminé avec succès pour produire un résultat.")
	}

	log.Println("Programme terminé.")
}

// printFibResultDetails affiche des informations détaillées sur le nombre de Fibonacci calculé.
func printFibResultDetails(value *big.Int, n int, duration time.Duration) {
	if value == nil {
		return
	}
	digits := len(value.Text(10))
	fmt.Printf("F(%d) calculé en %v\n", n, duration.Round(time.Millisecond))
	fmt.Printf("Nombre de chiffres : %d\n", digits)

	if digits > 20 {
		floatVal := new(big.Float).SetPrec(uint(digits + 10)).SetInt(value)
		sci := floatVal.Text('e', 8)
		fmt.Printf("Valeur de F(%d) ≈ %s (notation scientifique)\n", n, sci)
	} else {
		fmt.Printf("Valeur de F(%d) = %s\n", n, value.Text(10))
	}
}

// min renvoie le minimum de deux entiers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
