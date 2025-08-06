/*
================================================================================
Programme : Fibonacci Benchmark (Concurrent & Refactorisé)
Date      : 2025-08-06
Version   : 2.0 (Refactorisation de la Configuration & Amélioration de la Réactivité)

--------------------------------------------------------------------------------
Description :
Ce programme calcule le N-ième nombre de la suite de Fibonacci (F(N)) en utilisant
plusieurs algorithmes distincts, exécutés de manière concurrente pour comparer
leur performance et vérifier l'exactitude de leurs résultats.

Cette version introduit une configuration structurée, une sélection d'algorithmes
plus robuste et une réactivité améliorée à l'annulation du contexte.
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
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================
// Section 1: Configuration et Types Principaux
// ============================================================

const (
	progressRefreshRate  = 100 * time.Millisecond
	binetPrecisionMargin = 128
	// NOUVEAU: Constante pour la vérification réactive du contexte.
	contextCheckInterval = 1024
)

// NOUVEAU: Structure de configuration centralisée.
type Config struct {
	N           int
	Timeout     time.Duration
	Verbose     bool
	AlgoKeys    []string
	RunAllAlgos bool
}

// fibFunc représente la signature d'une fonction de calcul de Fibonacci.
type fibFunc func(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)

// task associe un nom à une fonction de calcul.
type task struct {
	name string
	fn   fibFunc
}

// result encapsule le résultat d'une tâche.
type result struct {
	name     string
	value    *big.Int
	duration time.Duration
	err      error
}

// MODIFIÉ: Map globale des algorithmes disponibles pour une recherche O(1).
var availableTasks = map[string]task{
	"fast-doubling": {"Fast-Doubling", fibFastDoubling},
	"matrix":        {"Matrix 2x2 (Opt)", fibMatrix},
	"binet":         {"Binet (Float)", fibBinet},
	"iterative":     {"Iterative (O(N))", fibIterative},
}

// ============================================================
// Section 2: Initialisation des Constantes de Haute Précision
// ============================================================

var (
	phi     *big.Float
	sqrt5   *big.Float
	log2Phi = 0.6942419136306173
)

func init() {
	var err error
	phi, _, err = new(big.Float).Parse("1.61803398874989484820458683436563811772030917980576", 10)
	if err != nil {
		log.Fatalf("Erreur d'initialisation de Phi: %v", err)
	}
	sqrt5, _, err = new(big.Float).Parse("2.23606797749978969640917366873127623544061835961152", 10)
	if err != nil {
		log.Fatalf("Erreur d'initialisation de Sqrt5: %v", err)
	}
}

// ============================================================
// Section 3: Gestion de l'Affichage et des Pools Mémoire
// ============================================================

// --- Logique d'affichage de la progression (inchangée) ---
type progressData struct {
	name string
	pct  float64
}

func progressPrinter(ctx context.Context, progress <-chan progressData, taskNames []string) {
	status := make(map[string]float64, len(taskNames))
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
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%-20s %6.2f%%", k, status[k]))
	}
	output := strings.Join(parts, " | ")
	fmt.Printf("%-150s", output)
}

// --- Gestion des pools mémoire ---

func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} { return new(big.Int) },
	}
}

// MODIFIÉ: Renommage pour plus de clarté sémantique.
type fastDoublingWorkspace struct {
	a_orig, f2k_term, fk_sq, fkp1_sq, new_a, new_b, t_sum *big.Int
}

func (ws *fastDoublingWorkspace) acquire(pool *sync.Pool) {
	ws.a_orig = pool.Get().(*big.Int)
	ws.f2k_term = pool.Get().(*big.Int)
	ws.fk_sq = pool.Get().(*big.Int)
	ws.fkp1_sq = pool.Get().(*big.Int)
	ws.new_a = pool.Get().(*big.Int)
	ws.new_b = pool.Get().(*big.Int)
	ws.t_sum = pool.Get().(*big.Int)
}

func (ws *fastDoublingWorkspace) release(pool *sync.Pool) {
	pool.Put(ws.a_orig)
	pool.Put(ws.f2k_term)
	pool.Put(ws.fk_sq)
	pool.Put(ws.fkp1_sq)
	pool.Put(ws.new_a)
	pool.Put(ws.new_b)
	pool.Put(ws.t_sum)
}

// MODIFIÉ: Renommage pour plus de clarté sémantique.
type matrixWorkspace struct {
	t1, t2 *big.Int
}

func (ws *matrixWorkspace) acquire(pool *sync.Pool) {
	ws.t1 = pool.Get().(*big.Int)
	ws.t2 = pool.Get().(*big.Int)
}

func (ws *matrixWorkspace) release(pool *sync.Pool) {
	pool.Put(ws.t1)
	pool.Put(ws.t2)
}

// ============================================================
// Section 4: Implémentation des Algorithmes
// ============================================================

// --- Fonction utilitaire pour cas de base (inchangée) ---
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

// --- Algorithme 1: Fast Doubling (O(log N)) ---
func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)

	totalBits := bits.Len(uint(n))
	ws := fastDoublingWorkspace{}
	ws.acquire(pool)
	defer ws.release(pool)

	for i := totalBits - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ws.a_orig.Set(a)
		ws.f2k_term.Lsh(b, 1).Sub(ws.f2k_term, ws.a_orig)
		ws.new_a.Mul(ws.a_orig, ws.f2k_term)
		ws.fk_sq.Mul(ws.a_orig, ws.a_orig)
		ws.fkp1_sq.Mul(b, b)
		ws.new_b.Add(ws.fk_sq, ws.fkp1_sq)
		a.Set(ws.new_a)
		b.Set(ws.new_b)

		if (uint(n)>>i)&1 == 1 {
			ws.t_sum.Add(a, b)
			a.Set(b)
			b.Set(ws.t_sum)
		}
		if progress != nil {
			progress <- (float64(totalBits-i) / float64(totalBits)) * 100.0
		}
	}
	return new(big.Int).Set(a), nil
}

// --- Algorithme 2: Matrix Exponentiation (O(log N)) ---
type mat2 struct{ a, b, c, d *big.Int }

func newMat2(pool *sync.Pool) *mat2 {
	return &mat2{a: pool.Get().(*big.Int), b: pool.Get().(*big.Int), c: pool.Get().(*big.Int), d: pool.Get().(*big.Int)}
}
func (m *mat2) release(pool *sync.Pool) { pool.Put(m.a); pool.Put(m.b); pool.Put(m.c); pool.Put(m.d) }
func (m *mat2) setIdentity()            { m.a.SetInt64(1); m.b.SetInt64(0); m.c.SetInt64(0); m.d.SetInt64(1) }
func (m *mat2) setFibBase()             { m.a.SetInt64(1); m.b.SetInt64(1); m.c.SetInt64(1); m.d.SetInt64(0) }
func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}
func matMul(target, m1, m2 *mat2, ws *matrixWorkspace) {
	ws.t1.Mul(m1.a, m2.a)
	ws.t2.Mul(m1.b, m2.c)
	target.a.Add(ws.t1, ws.t2)
	ws.t1.Mul(m1.a, m2.b)
	ws.t2.Mul(m1.b, m2.d)
	target.b.Add(ws.t1, ws.t2)
	ws.t1.Mul(m1.c, m2.a)
	ws.t2.Mul(m1.d, m2.c)
	target.c.Add(ws.t1, ws.t2)
	ws.t1.Mul(m1.c, m2.b)
	ws.t2.Mul(m1.d, m2.d)
	target.d.Add(ws.t1, ws.t2)
}

func fibMatrix(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	exp := uint(n - 1)
	ws := matrixWorkspace{}
	ws.acquire(pool)
	defer ws.release(pool)
	res := newMat2(pool)
	defer res.release(pool)
	res.setIdentity()
	base := newMat2(pool)
	defer base.release(pool)
	base.setFibBase()
	tempMat := newMat2(pool)
	defer tempMat.release(pool)

	totalSteps := bits.Len(exp)
	for stepsDone := 0; exp > 0; stepsDone++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if exp&1 == 1 {
			matMul(tempMat, res, base, &ws)
			res.set(tempMat)
		}
		if exp > 1 {
			matMul(tempMat, base, base, &ws)
			base.set(tempMat)
		}
		exp >>= 1
		if progress != nil && totalSteps > 0 {
			progress <- (float64(stepsDone) / float64(totalSteps)) * 100.0
		}
	}
	return new(big.Int).Set(res.a), nil
}

// --- Algorithme 3: Formule de Binet (O(log N)) ---
func fibBinet(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	prec := uint(float64(n)*log2Phi + binetPrecisionMargin)
	phiPrec := new(big.Float).SetPrec(prec).Set(phi)
	sqrt5Prec := new(big.Float).SetPrec(prec).Set(sqrt5)

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
		if progress != nil && numBitsInN > 0 {
			progress <- (float64(currentStep) / float64(numBitsInN)) * 99.0
		}
	}

	result.Quo(result, sqrt5Prec)
	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	result.Add(result, half)
	z, _ := result.Int(new(big.Int))
	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

// --- Algorithme 4: Itératif Optimisé (O(N)) ---
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

	for i := 2; i <= n; i++ {
		// MODIFIÉ: Vérification réactive du contexte pour garantir l'annulation rapide.
		if i%contextCheckInterval == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		temp.Add(a, b)
		a.Set(b)
		b.Set(temp)

		// Rapport de progression moins fréquent, pour ne pas polluer le check de contexte.
		if progress != nil && i%(n/100+1) == 0 {
			progress <- (float64(i) / float64(n)) * 100.0
		}
	}
	if progress != nil {
		progress <- 100.0
	}
	return new(big.Int).Set(b), nil
}

// ============================================================
// Section 5: Logique d'Exécution Principale
// ============================================================

// NOUVEAU: Fonction dédiée à l'analyse des flags.
func parseAndValidateFlags() (*Config, error) {
	cfg := &Config{}

	flag.IntVar(&cfg.N, "n", 2500000, "Index N du terme de Fibonacci.")
	flag.DurationVar(&cfg.Timeout, "timeout", 2*time.Minute, "Temps d'exécution maximum global (ex: '1m', '30s').")
	flag.BoolVar(&cfg.Verbose, "v", false, "Sortie verbeuse : affiche le nombre de Fibonacci complet.")

	// Utilisation d'un flag personnalisé pour gérer "all" ou une liste.
	runAlgosStr := flag.String("runAlgos", "all", "Liste des algorithmes à exécuter (clés: fast-doubling, matrix, binet, iterative), séparés par des virgules, ou 'all'.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage de %s:\n", os.Args[0])
		flag.PrintDefaults()
		// Afficher les clés d'algorithmes disponibles.
		var keys []string
		for k := range availableTasks {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(os.Stderr, "\nAlgorithmes disponibles via -runAlgos: %s\n", strings.Join(keys, ", "))
	}

	flag.Parse()

	if cfg.N < 0 {
		return nil, fmt.Errorf("l'index N doit être >= 0. Reçu : %d", cfg.N)
	}

	if strings.ToLower(strings.TrimSpace(*runAlgosStr)) == "all" {
		cfg.RunAllAlgos = true
	} else {
		cfg.AlgoKeys = strings.Split(*runAlgosStr, ",")
		for i, key := range cfg.AlgoKeys {
			cfg.AlgoKeys[i] = strings.TrimSpace(strings.ToLower(key))
		}
	}

	return cfg, nil
}

// NOUVEAU: Logique de sélection des tâches simplifiée et robuste.
func selectTasks(cfg *Config) ([]task, []string) {
	var selectedTasks []task
	var selectedNames []string

	if cfg.RunAllAlgos {
		// Garantir un ordre stable pour l'affichage
		var keys []string
		for k := range availableTasks {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			t := availableTasks[key]
			selectedTasks = append(selectedTasks, t)
			selectedNames = append(selectedNames, t.name)
		}
		return selectedTasks, selectedNames
	}

	// Utilisation d'un map pour éviter les doublons.
	addedTasks := make(map[string]bool)
	for _, key := range cfg.AlgoKeys {
		if t, ok := availableTasks[key]; ok {
			if !addedTasks[t.name] {
				selectedTasks = append(selectedTasks, t)
				selectedNames = append(selectedNames, t.name)
				addedTasks[t.name] = true
			}
		} else if key != "" {
			log.Printf("Avertissement: Clé d'algorithme '%s' non reconnue.", key)
		}
	}

	// Trier les noms pour un affichage cohérent.
	sort.Strings(selectedNames)
	return selectedTasks, selectedNames
}

func main() {
	// 1. Configuration
	cfg, err := parseAndValidateFlags()
	if err != nil {
		log.Fatalf("Erreur de configuration: %v", err)
	}

	// 2. Sélection des Tâches
	selectedTasks, selectedAlgoNames := selectTasks(cfg)
	if len(selectedTasks) == 0 {
		log.Println("Aucun algorithme valide sélectionné. Sortie.")
		return
	}

	log.Printf("Calcul de F(%d) avec un timeout de %v...", cfg.N, cfg.Timeout)
	log.Printf("Algorithmes : %s", strings.Join(selectedAlgoNames, ", "))

	// 3. Initialisation de la Concurrence
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	intPool := newIntPool()
	progressAggregatorCh := make(chan progressData, len(selectedTasks)*100)
	resultsCh := make(chan result, len(selectedTasks))

	// 4. Lancement du Moniteur de Progression
	var printerWg sync.WaitGroup
	printerWg.Add(1)
	go func() {
		defer printerWg.Done()
		progressPrinter(ctx, progressAggregatorCh, selectedAlgoNames)
	}()

	// 5. Lancement des Calculs (Fan-out)
	var calculationWg sync.WaitGroup
	log.Println("Lancement des calculs concurrents...")
	for _, t := range selectedTasks {
		calculationWg.Add(1)
		go runTask(ctx, t, cfg, intPool, &calculationWg, progressAggregatorCh, resultsCh)
	}

	// 6. Attente et Synchronisation
	// Goroutine pour fermer les canaux après la fin des calculs.
	go func() {
		calculationWg.Wait()
		close(progressAggregatorCh)
		close(resultsCh)
	}()

	// 7. Traitement des Résultats
	results := collectAndSortResults(resultsCh, len(selectedTasks))
	printerWg.Wait() // Attendre que l'imprimante finisse après la fermeture de son canal.

	processResults(results, cfg)

	log.Println("Programme terminé.")
}

// MODIFIÉ: runTask gère maintenant une attente de relais de progression simplifiée.
func runTask(ctx context.Context, currentTask task, cfg *Config, pool *sync.Pool, wg *sync.WaitGroup, progressAggregatorCh chan<- progressData, resultsCh chan<- result) {
	defer wg.Done()

	localProgCh := make(chan float64, 10)
	var relayWg sync.WaitGroup
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
	v, err := currentTask.fn(ctx, localProgCh, cfg.N, pool)
	duration := time.Since(start)

	close(localProgCh)
	relayWg.Wait() // S'assurer que tous les messages de progression sont relayés avant d'envoyer le résultat.

	resultsCh <- result{currentTask.name, v, duration, err}
}

// NOUVEAU: Fonction dédiée à la collecte et au tri des résultats.
func collectAndSortResults(resultsCh <-chan result, numTasks int) []result {
	results := make([]result, 0, numTasks)
	for r := range resultsCh {
		results = append(results, r)
		if r.err != nil {
			logMsg := fmt.Sprintf("❌ Erreur pour la tâche '%s': %v (durée: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				logMsg = fmt.Sprintf("⚠️ Tâche '%s' a dépassé le délai ou a été annulée après %v", r.name, r.duration.Round(time.Microsecond))
			}
			log.Println(logMsg)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true
		}
		if results[i].err != nil && results[j].err == nil {
			return false
		}
		return results[i].duration < results[j].duration
	})
	return results
}

// MODIFIÉ: Logique de traitement des résultats utilisant la config.
func processResults(results []result, cfg *Config) {
	fmt.Println("\n--------------------------- RÉSULTATS ORDONNÉS ---------------------------")
	for _, r := range results {
		status, valStr := "OK", "N/A"
		if r.err != nil {
			status = "Erreur"
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				status = "Timeout/Annulé"
			}
		} else if r.value != nil {
			valStr = summarizeBigInt(r.value)
		}
		fmt.Printf("%-20s : %-15v [%-16s] Résultat: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	fmt.Println("\n--------------------------- VÉRIFICATION ------------------------------")
	verifyAndPrintDetails(results, cfg)
}

// --- Fonctions d'affichage et de vérification (logique interne peu modifiée) ---
func summarizeBigInt(v *big.Int) string {
	s := v.String()
	if len(s) > 20 {
		return s[:8] + "..." + s[len(s)-8:]
	}
	return s
}

func verifyAndPrintDetails(results []result, cfg *Config) {
	var firstSuccessfulResult *result
	allValidResultsIdentical := true
	successfulCount := 0

	for i := range results {
		if results[i].err == nil && results[i].value != nil {
			successfulCount++
			if firstSuccessfulResult == nil {
				firstSuccessfulResult = &results[i]
				fmt.Printf("Algorithme réussi le plus rapide : %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
				printFibResultDetails(firstSuccessfulResult.value, cfg)
			} else if results[i].value.Cmp(firstSuccessfulResult.value) != 0 {
				allValidResultsIdentical = false
				log.Printf("⚠️ DIVERGENCE DÉTECTÉE ! Le résultat de '%s' diffère de '%s'.", results[i].name, firstSuccessfulResult.name)
				if strings.Contains(results[i].name, "Binet") || strings.Contains(firstSuccessfulResult.name, "Binet") {
					log.Println("  (Note: La formule de Binet peut diverger pour de très grands N si la précision flottante est insuffisante).")
				}
			}
		}
	}

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

func printFibResultDetails(value *big.Int, cfg *Config) {
	if value == nil {
		return
	}
	s := value.Text(10)
	digits := len(s)
	fmt.Printf("Nombre de chiffres dans F(%d) : %d\n", cfg.N, digits)

	if cfg.Verbose {
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
