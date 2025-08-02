/*
================================================================================
Programme : Fibonacci Benchmark (Concurrent & Optimisé)
Date      : 2025-08-03
Version   : 1.2 (Optimisé Matrix Pool, Robustesse & Clarté)

--------------------------------------------------------------------------------
Description :
Ce programme calcule le N-ième nombre de la suite de Fibonacci (F(N)) en utilisant
plusieurs algorithmes distincts, exécutés de manière concurrente pour comparer
leur performance et vérifier l'exactitude de leurs résultats.

Il est fortement optimisé pour la gestion mémoire via l'utilisation intensive
de sync.Pool pour recycler les objets big.Int.

[Instructions d'Utilisation inchangées]
================================================================================
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/bits"
	"sort"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Constantes de Configuration
// ------------------------------------------------------------

const (
	// [CLARTÉ] Définition des constantes de configuration
	progressRefreshRate  = 100 * time.Millisecond // Taux de rafraîchissement de l'UI
	binetPrecisionMargin = 128                    // Marge de sécurité en bits pour Binet
)

// ------------------------------------------------------------
// Types et Structures
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

var verboseFlag bool

// ------------------------------------------------------------
// Constantes pour la Formule de Binet
// ------------------------------------------------------------

var (
	// phi (Nombre d'Or) et sqrt5. Initialisés dans init() pour haute précision.
	phi   *big.Float
	sqrt5 *big.Float

	// log2(Phi) ≈ 0.69424. Utilisé pour estimer la précision requise.
	log2Phi = 0.6942419136306173
)

// [ROBUSTESSE] Initialisation des constantes de haute précision à partir de chaînes.
func init() {
	var err error
	// Utilisation de Parse pour éviter la conversion intermédiaire en float64.
	phi, _, err = new(big.Float).Parse("1.61803398874989484820458683436563811772030917980576", 10)
	if err != nil {
		log.Fatalf("Erreur d'initialisation de Phi: %v", err)
	}
	sqrt5, _, err = new(big.Float).Parse("2.23606797749978969640917366873127623544061835961152", 10)
	if err != nil {
		log.Fatalf("Erreur d'initialisation de Sqrt5: %v", err)
	}
}

// ------------------------------------------------------------
// Gestion de l'Affichage de la Progression
// (Logique inchangée, utilise maintenant progressRefreshRate)
// ------------------------------------------------------------

type progressData struct {
	name string
	pct  float64
}

func progressPrinter(ctx context.Context, progress <-chan progressData, taskNames []string) {
	status := make(map[string]float64)
	ticker := time.NewTicker(progressRefreshRate)
	defer ticker.Stop()

	for _, k := range taskNames {
		status[k] = 0.0
	}

	if len(taskNames) == 0 {
		return
	}

	needsUpdate := true

	for {
		select {
		case <-ctx.Done():
			printStatus(status, taskNames)
			fmt.Println("\n(Annulé/Timeout)")
			return
		case p, ok := <-progress:
			if !ok {
				printStatus(status, taskNames)
				fmt.Println()
				return
			}
			if _, exists := status[p.name]; exists && status[p.name] != p.pct {
				status[p.name] = p.pct
				needsUpdate = true
			}
		case <-ticker.C:
			if needsUpdate {
				printStatus(status, taskNames)
				needsUpdate = false
			}
		}
	}
}

func printStatus(status map[string]float64, keys []string) {
	fmt.Print("\r")
	var parts []string
	// Assure un ordre consistant (les keys sont déjà triées dans main)
	for _, k := range keys {
		v, ok := status[k]
		if !ok {
			continue
		}
		// Augmentation de la largeur pour les noms d'algorithmes
		parts = append(parts, fmt.Sprintf("%-20s %6.2f%%", k, v))
	}
	output := strings.Join(parts, " | ")
	// Padding large pour effacer la ligne précédente.
	fmt.Printf("%-150s", output)
}

// ------------------------------------------------------------
// Gestion du Pool Mémoire (sync.Pool pour big.Int)
// ------------------------------------------------------------

func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// --- Structures Temporaires pour Algorithmes Spécifiques ---

// fastDoublingTemps gère les temporaires pour Fast Doubling.
// [CLARTÉ] Noms de variables clarifiés pour correspondre aux identités mathématiques.
type fastDoublingTemps struct {
	a_orig   *big.Int // F(k) sauvegardé
	f2k_term *big.Int // Terme pour F(2k): [2*F(k+1) – F(k)] (ex t1)
	fk_sq    *big.Int // F(k)^2 (ex t2)
	fkp1_sq  *big.Int // F(k+1)^2 (ex t3)
	new_a    *big.Int
	new_b    *big.Int
	t_sum    *big.Int
}

func (tmp *fastDoublingTemps) acquire(pool *sync.Pool) {
	tmp.a_orig = pool.Get().(*big.Int)
	tmp.f2k_term = pool.Get().(*big.Int)
	tmp.fk_sq = pool.Get().(*big.Int)
	tmp.fkp1_sq = pool.Get().(*big.Int)
	tmp.new_a = pool.Get().(*big.Int)
	tmp.new_b = pool.Get().(*big.Int)
	tmp.t_sum = pool.Get().(*big.Int)
}

func (tmp *fastDoublingTemps) release(pool *sync.Pool) {
	pool.Put(tmp.a_orig)
	pool.Put(tmp.f2k_term)
	pool.Put(tmp.fk_sq)
	pool.Put(tmp.fkp1_sq)
	pool.Put(tmp.new_a)
	pool.Put(tmp.new_b)
	pool.Put(tmp.t_sum)
}

// [OPTIMISATION] matrixTemps gère les temporaires pour la multiplication matricielle.
type matrixTemps struct {
	t1, t2 *big.Int
}

func (tmp *matrixTemps) acquire(pool *sync.Pool) {
	tmp.t1 = pool.Get().(*big.Int)
	tmp.t2 = pool.Get().(*big.Int)
}

func (tmp *matrixTemps) release(pool *sync.Pool) {
	pool.Put(tmp.t1)
	pool.Put(tmp.t2)
}

// ------------------------------------------------------------
// Fonctions Utilitaires
// ------------------------------------------------------------

// handleBaseCases gère les cas triviaux (n < 0, n=0, n=1).
func handleBaseCases(n int, progress chan<- float64) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("index négatif non supporté: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}
	return nil, nil
}

// ------------------------------------------------------------
// Algorithme 1: Fast Doubling (O(log N)) - OPTIMISÉ
// ------------------------------------------------------------

// fibFastDoubling utilise les identités de doublage.
// F(2k) = F(k) * [2*F(k+1) – F(k)]
// F(2k+1) = F(k)^2 + F(k+1)^2
func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Initialisation : a = F(0) = 0, b = F(1) = 1.
	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)

	totalBits := bits.Len(uint(n))
	temps := fastDoublingTemps{}

	// Acquisition des variables temporaires UNE SEULE FOIS.
	temps.acquire(pool)
	defer temps.release(pool)

	// Itération sur les bits de N.
	for i := totalBits - 1; i >= 0; i-- {
		// [AMÉLIORATION] Vérification du contexte à chaque itération (coût négligeable en O(log N)).
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		temps.a_orig.Set(a) // Sauvegarde F(k)

		// Calcul de F(2k) - Utilisation des noms clarifiés.
		temps.f2k_term.Lsh(b, 1)                         // 2*F(k+1)
		temps.f2k_term.Sub(temps.f2k_term, temps.a_orig) // [2*F(k+1) - F(k)]
		temps.new_a.Mul(temps.a_orig, temps.f2k_term)    // F(2k)

		// Calcul de F(2k+1)
		temps.fk_sq.Mul(temps.a_orig, temps.a_orig) // F(k)^2
		temps.fkp1_sq.Mul(b, b)                     // F(k+1)^2
		temps.new_b.Add(temps.fk_sq, temps.fkp1_sq) // F(2k+1)

		a.Set(temps.new_a)
		b.Set(temps.new_b)

		// Si le i-ème bit de n est 1, on avance d'un pas.
		if (uint(n)>>i)&1 == 1 {
			temps.t_sum.Add(a, b)
			a.Set(b)
			b.Set(temps.t_sum)
		}

		if progress != nil {
			progress <- (float64(totalBits-i) / float64(totalBits)) * 100.0
		}
	}

	// Retourne une copie de 'a'.
	return new(big.Int).Set(a), nil
}

// ------------------------------------------------------------
// Algorithme 2: Matrix Exponentiation (O(log N)) - OPTIMISÉ
// ------------------------------------------------------------

type mat2 struct {
	a, b, c, d *big.Int
}

// (Fonctions utilitaires mat2 inchangées : newMat2, release, setIdentity, setFibBase, set)
func newMat2(pool *sync.Pool) *mat2 {
	return &mat2{
		a: pool.Get().(*big.Int), b: pool.Get().(*big.Int),
		c: pool.Get().(*big.Int), d: pool.Get().(*big.Int),
	}
}

func (m *mat2) release(pool *sync.Pool) {
	pool.Put(m.a)
	pool.Put(m.b)
	pool.Put(m.c)
	pool.Put(m.d)
}

func (m *mat2) setIdentity() {
	m.a.SetInt64(1)
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

func (m *mat2) setFibBase() {
	m.a.SetInt64(1)
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// matMul calcule target = m1 * m2.
// [OPTIMISATION] Accepte matrixTemps pour éviter l'accès au pool dans la boucle.
func matMul(target, m1, m2 *mat2, temps *matrixTemps) {
	// Utilisation des variables temporaires fournies (t1, t2).
	t1 := temps.t1
	t2 := temps.t2

	// Calcul standard de multiplication matricielle 2x2.
	// target.a = (m1.a*m2.a) + (m1.b*m2.c)
	t1.Mul(m1.a, m2.a)
	t2.Mul(m1.b, m2.c)
	target.a.Add(t1, t2)
	// target.b = (m1.a*m2.b) + (m1.b*m2.d)
	t1.Mul(m1.a, m2.b)
	t2.Mul(m1.b, m2.d)
	target.b.Add(t1, t2)
	// target.c = (m1.c*m2.a) + (m1.d*m2.c)
	t1.Mul(m1.c, m2.a)
	t2.Mul(m1.d, m2.c)
	target.c.Add(t1, t2)
	// target.d = (m1.c*m2.b) + (m1.d*m2.d)
	t1.Mul(m1.c, m2.b)
	t2.Mul(m1.d, m2.d)
	target.d.Add(t1, t2)
}

// fibMatrix calcule F(N) en utilisant l'exponentiation matricielle binaire.
func fibMatrix(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// On calcule M^(n-1).
	exp := uint(n - 1)

	// [OPTIMISATION] Acquisition des temporaires de multiplication UNE SEULE FOIS.
	temps := matrixTemps{}
	temps.acquire(pool)
	defer temps.release(pool)

	res := newMat2(pool)
	defer res.release(pool)
	res.setIdentity()

	base := newMat2(pool)
	defer base.release(pool)
	base.setFibBase()

	// Matrice temporaire pour éviter l'aliasing.
	tempMat := newMat2(pool)
	defer tempMat.release(pool)

	totalSteps := bits.Len(exp)
	stepsDone := 0

	// Algorithme d'exponentiation binaire.
	for exp > 0 {
		// [AMÉLIORATION] Vérification réactive du contexte.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exp&1 == 1 {
			// Passage des temporaires pré-alloués.
			matMul(tempMat, res, base, &temps)
			res.set(tempMat)
		}

		if exp > 1 {
			matMul(tempMat, base, base, &temps)
			base.set(tempMat)
		}

		exp >>= 1
		stepsDone++

		if progress != nil && totalSteps > 0 {
			progress <- (float64(stepsDone) / float64(totalSteps)) * 100.0
		}
	}

	return new(big.Int).Set(res.a), nil
}

// ------------------------------------------------------------
// Algorithme 3: Formule de Binet (O(log N), basé sur Float)
// ------------------------------------------------------------

// fibBinet utilise la formule F(n) ≈ round(Phi^n / sqrt(5)).
func fibBinet(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Calcul de la précision nécessaire avec la marge de sécurité.
	prec := uint(float64(n)*log2Phi + binetPrecisionMargin)

	// Initialisation des floats avec la précision dynamique.
	phiPrec := new(big.Float).SetPrec(prec).Set(phi)
	sqrt5Prec := new(big.Float).SetPrec(prec).Set(sqrt5)

	// Calcul de Phi^n (exponentiation binaire sur big.Float).
	result := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phiPrec)
	exponent := uint(n)

	numBitsInN := bits.Len(uint(n))
	currentStep := 0

	for exponent > 0 {
		// [AMÉLIORATION] Vérification réactive du contexte.
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

		currentStep++
		if progress != nil && numBitsInN > 0 {
			progress <- (float64(currentStep) / float64(numBitsInN)) * 99.0
		}
	}

	// (Phi^n) / sqrt(5)
	result.Quo(result, sqrt5Prec)

	// Arrondir : floor(result + 0.5)
	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	result.Add(result, half)

	z := new(big.Int)
	result.Int(z)

	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

// ------------------------------------------------------------
// Algorithme 4: Itératif Optimisé (O(N))
// (Logique inchangée, déjà bien optimisée)
// ------------------------------------------------------------

func fibIterative(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)
	temp := pool.Get().(*big.Int)
	defer pool.Put(temp)

	// Ajustement dynamique de l'intervalle de rapport.
	reportInterval := n / 100 // Viser environ 100 mises à jour.
	if reportInterval < 1000 {
		reportInterval = 1000
	}
	// Protection contre la division par zéro si n < 100
	if reportInterval == 0 {
		reportInterval = 1
	}

	for i := 2; i <= n; i++ {
		if i%reportInterval == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				if progress != nil {
					progress <- (float64(i) / float64(n)) * 100.0
				}
			}
		}

		temp.Add(a, b)
		a.Set(b)
		b.Set(temp)
	}

	if progress != nil {
		progress <- 100.0
	}

	return new(big.Int).Set(b), nil
}

// ------------------------------------------------------------
// Logique d'Exécution Principale
// ------------------------------------------------------------

func main() {
	// 1. Analyse des Flags (Parsing)
	// Augmentation du défaut à 500k pour un benchmark plus exigeant.
	nFlag := flag.Int("n", 1000000, "Index N du terme de Fibonacci. Défaut 500,000.")
	timeoutFlag := flag.Duration("timeout", 2*time.Minute, "Temps d'exécution maximum global (ex: 1m, 30s).")
	runAlgosFlag := flag.String("runAlgos", "all", "Liste séparée par des virgules des algorithmes (ex: 'binet,fast').")
	flag.BoolVar(&verboseFlag, "v", false, "Sortie verbeuse : affiche le nombre de Fibonacci complet.")
	flag.Parse()

	n := *nFlag
	timeout := *timeoutFlag
	runAlgosStr := strings.ToLower(*runAlgosFlag)

	if n < 0 {
		log.Fatalf("L'index N doit être >= 0. Reçu : %d", n)
	}

	// 2. Définition et Sélection des Tâches
	definedTasks := []task{
		{"Fast-Doubling", fibFastDoubling},
		{"Matrix 2x2 (Opt)", fibMatrix}, // Nom mis à jour
		{"Binet (Float)", fibBinet},
		{"Iterative (O(N))", fibIterative},
	}

	selectedTasks, selectedAlgoNames := selectTasks(definedTasks, runAlgosStr)

	if len(selectedTasks) == 0 {
		log.Println("Aucun algorithme valide sélectionné. Sortie.")
		return
	}

	// [CLARTÉ] Tri des noms pour garantir un ordre d'affichage consistant dans progressPrinter
	sort.Strings(selectedAlgoNames)

	log.Printf("Calcul de F(%d) avec un timeout de %v...", n, timeout)
	log.Printf("Algorithmes : %s", strings.Join(selectedAlgoNames, ", "))

	// 3. Configuration du Contexte, Pool et Canaux
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	intPool := newIntPool()
	progressAggregatorCh := make(chan progressData, len(selectedTasks)*100)
	resultsCh := make(chan result, len(selectedTasks))

	// 4. Démarrage du Moniteur de Progression
	var printerWg sync.WaitGroup
	printerWg.Add(1)
	go func() {
		defer printerWg.Done()
		progressPrinter(ctx, progressAggregatorCh, selectedAlgoNames)
	}()

	// 5. Lancement des Calculs Concurrents (Fan-out)
	var calculationWg sync.WaitGroup
	var relayWg sync.WaitGroup

	log.Println("Lancement des calculs concurrents...")

	for _, t := range selectedTasks {
		calculationWg.Add(1)
		go runTask(ctx, t, n, intPool, &calculationWg, &relayWg, progressAggregatorCh, resultsCh)
	}

	// 6. Attente de la complétion (Synchronisation Fan-in)
	calculationWg.Wait()
	// Séquence d'arrêt critique : s'assurer que les relais ont vidé les canaux locaux.
	relayWg.Wait()
	close(progressAggregatorCh)
	printerWg.Wait()

	// 7. Traitement des Résultats
	processResults(resultsCh, selectedTasks, n)

	log.Println("Programme terminé.")
}

// selectTasks analyse l'entrée utilisateur.
func selectTasks(definedTasks []task, runAlgosStr string) ([]task, []string) {
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		// Nettoyage des suffixes pour correspondance facile
		s = strings.ReplaceAll(s, "(float)", "")
		s = strings.ReplaceAll(s, "(o(n))", "")
		s = strings.ReplaceAll(s, "2x2", "")
		s = strings.ReplaceAll(s, "(opt)", "")
		return s
	}

	availableAlgos := make(map[string]task)
	for _, t := range definedTasks {
		availableAlgos[normalize(t.name)] = t
	}

	var selectedTasks []task
	var selectedAlgoNames []string

	// Tri initial pour garantir l'ordre si "all" est sélectionné.
	sort.Slice(definedTasks, func(i, j int) bool {
		return definedTasks[i].name < definedTasks[j].name
	})

	if runAlgosStr == "all" || runAlgosStr == "" {
		for _, t := range definedTasks {
			selectedTasks = append(selectedTasks, t)
			selectedAlgoNames = append(selectedAlgoNames, t.name)
		}
		return selectedTasks, selectedAlgoNames
	}

	userRequestedAlgoNames := strings.Split(runAlgosStr, ",")
	addedAlgos := make(map[string]bool)

	for _, name := range userRequestedAlgoNames {
		normalizedName := normalize(strings.TrimSpace(name))
		found := false

		// Correspondance partielle.
		for key, task := range availableAlgos {
			if strings.Contains(key, normalizedName) && normalizedName != "" {
				if !addedAlgos[task.name] {
					selectedTasks = append(selectedTasks, task)
					selectedAlgoNames = append(selectedAlgoNames, task.name)
					addedAlgos[task.name] = true
					found = true
				}
				break
			}
		}

		if !found {
			log.Printf("Avertissement: Algorithme '%s' non reconnu ou non trouvé.", name)
		}
	}
	return selectedTasks, selectedAlgoNames
}

// runTask exécute une tâche et gère le relais de progression (Logique inchangée).
func runTask(ctx context.Context, currentTask task, n int, pool *sync.Pool, calculationWg *sync.WaitGroup, relayWg *sync.WaitGroup, progressAggregatorCh chan<- progressData, resultsCh chan<- result) {
	defer calculationWg.Done()

	// Canal local pour le découplage de la progression.
	localProgCh := make(chan float64, 10)

	// Goroutine de relais.
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
	v, err := currentTask.fn(ctx, localProgCh, n, pool)
	duration := time.Since(start)
	close(localProgCh) // Signale la fin au relais.

	resultsCh <- result{currentTask.name, v, duration, err}
}

// processResults collecte, trie, affiche et vérifie les résultats (Logique inchangée).
func processResults(resultsCh chan result, selectedTasks []task, n int) {
	results := make([]result, 0, len(selectedTasks))

	// Collecte des résultats (Fan-in).
	for i := 0; i < len(selectedTasks); i++ {
		r := <-resultsCh
		results = append(results, r)
		if r.err != nil {
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				log.Printf("⚠️ Tâche '%s' a dépassé le délai ou a été annulée après %v", r.name, r.duration.Round(time.Microsecond))
			} else {
				log.Printf("❌ Erreur pour la tâche '%s': %v (durée: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			}
		}
	}
	close(resultsCh)

	// Tri des résultats : réussites d'abord, puis par durée.
	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true
		}
		if results[i].err != nil && results[j].err == nil {
			return false
		}
		return results[i].duration < results[j].duration
	})

	// Affichage des Résultats Ordonnés
	fmt.Println("\n--------------------------- RÉSULTATS ORDONNÉS ---------------------------")
	for _, r := range results {
		status := "OK"
		valStr := "N/A"
		if r.err != nil {
			status = "Erreur"
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				status = "Timeout/Annulé"
			}
		} else if r.value != nil {
			valStr = summarizeBigInt(r.value)
		}
		// Formatage aligné (largeur 20)
		fmt.Printf("%-20s : %-15v [%-16s] Résultat: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	// Vérification
	fmt.Println("\n--------------------------- VÉRIFICATION ------------------------------")
	verifyAndPrintDetails(results, n)
}

// summarizeBigInt fournit une représentation courte d'un grand nombre.
func summarizeBigInt(v *big.Int) string {
	s := v.String()
	if len(s) > 20 {
		// Affiche un peu plus de chiffres pour le contexte.
		return s[:8] + "..." + s[len(s)-8:]
	}
	return s
}

// verifyAndPrintDetails compare les résultats.
func verifyAndPrintDetails(results []result, n int) {
	var firstSuccessfulResult *result
	allValidResultsIdentical := true
	successfulCount := 0

	for i := range results {
		if results[i].err == nil && results[i].value != nil {
			successfulCount++
			if firstSuccessfulResult == nil {
				firstSuccessfulResult = &results[i]
				fmt.Printf("Algorithme réussi le plus rapide : %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
				printFibResultDetails(firstSuccessfulResult.value, n)
			} else {
				if results[i].value.Cmp(firstSuccessfulResult.value) != 0 {
					allValidResultsIdentical = false
					log.Printf("⚠️ DIVERGENCE DÉTECTÉE ! Le résultat de '%s' diffère de '%s'.",
						results[i].name, firstSuccessfulResult.name)
					// [CLARTÉ] Ajout d'une note si Binet est impliqué (cause probable de divergence).
					if strings.Contains(results[i].name, "Binet") || strings.Contains(firstSuccessfulResult.name, "Binet") {
						log.Println("   (Note: La formule de Binet peut diverger pour de très grands N si la précision flottante est insuffisante).")
					}
				}
			}
		}
	}

	// Conclusion de la vérification.
	if successfulCount > 0 {
		if allValidResultsIdentical {
			fmt.Println("✅ Vérification réussie : Tous les algorithmes terminés ont donné des résultats identiques.")
		} else {
			fmt.Println("❌ Échec de la vérification : Les résultats des algorithmes divergent !")
		}
	} else {
		fmt.Println("❌ Aucun algorithme n'a réussi à terminer le calcul.")
	}
}

// printFibResultDetails affiche des informations détaillées (Logique inchangée).
func printFibResultDetails(value *big.Int, n int) {
	if value == nil {
		return
	}
	s := value.Text(10)
	digits := len(s)
	fmt.Printf("Nombre de chiffres dans F(%d) : %d\n", n, digits)

	if verboseFlag {
		fmt.Printf("Valeur = %s\n", s)
	} else if digits > 50 {
		floatVal := new(big.Float).SetInt(value)
		sci := floatVal.Text('e', 8)
		fmt.Printf("Valeur ≈ %s\n", sci)
		fmt.Printf("Valeur = %s...%s\n", s[:10], s[len(s)-10:])
	} else {
		fmt.Printf("Valeur = %s\n", s)
	}
}
