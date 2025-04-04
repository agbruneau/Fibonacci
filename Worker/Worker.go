// =============================================================================
// Programme : Calcul ultra-optimisé et parallèle de listes de Fibonacci(n) en Go
// Auteur    : André-Guy Bruneau // Adapté par l'IA Gemini 2.5 PRo Experimental 03-2025
// Date      : 2025-04-03 // Date de la modification
// Version   : 1.4 // Introduction du calcul parallèle pour une liste de `n`.
//
// Description :
// Version 1.4 : Introduction du calcul parallèle pour une liste de `n`.
// - Utilisation d'un modèle Worker Pool pour distribuer les calculs.
// Version 1.3 : Intégration des "Minor Potential Considerations" de la V1.2.
// - Remplacement de la comparaison directe d'erreurs de contexte par errors.Is dans la goroutine principale pour plus de robustesse.
// - Ajout de commentaires sur la portabilité de l'affichage de progression ('\r') et la nature de la métrique de progression (bits.Len).
// Version 1.2 : Intégration des "Minor Potential Considerations" de la V1.1.
// - Remplacement du cache map[int]*big.Int par un cache LRU (github.com/hashicorp/golang-lru/v2)
//   pour limiter l'utilisation mémoire du cache.
// - Utilisation de string (strconv.Itoa(n)) comme clé de cache pour supprimer la limite théorique de int.
// - Ajout du paramètre Config.CacheSize.
// - Suppression du sync.RWMutex car la bibliothèque LRU gère sa propre synchronisation.
// Version 1.1 : Intégration des suggestions de raffinement de la V1.0.
// - Optimisation de multiplyMatrices pour utiliser 2 *big.Int temporaires.
// - Propagation du contexte (context.Context) dans Calculate et fastDoubling.
// Version 1.0 : Ajout du suivi de progression.
// ... (Historique précédent omis pour la brièveté) ...
// =============================================================================

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big" // Importé mais bits.Len n'est plus utilisé dans fastDoubling V1.6
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// --- Constantes ---
const (
	// ProgressReportInterval : Fréquence théorique de mise à jour si l'affichage était actif.
	ProgressReportInterval = 1 * time.Second
)

// --- Structures ---

// Config structure pour les paramètres de configuration.
type Config struct {
	NsToCalculate   []int
	Timeout         time.Duration
	Precision       int
	Workers         int
	EnableCache     bool
	CacheSize       int
	EnableProfiling bool
}

// DefaultConfig retourne la configuration par défaut.
func DefaultConfig() Config {
	exampleNs := []int{
		1000000, 500000, 1000001, 2000000, 500000,
		3000000, 1500000, 100, 0, 1, 2, 4000000, 700000, 800000, 950000, 1100000,
		2500000, 3500000, 10000000, 20000000,
	}
	numWorkers := runtime.NumCPU()
	return Config{
		NsToCalculate:   exampleNs,
		Timeout:         5 * time.Minute,
		Precision:       10,
		Workers:         numWorkers,
		EnableCache:     true,
		CacheSize:       2048,
		EnableProfiling: false,
	}
}

// Metrics structure pour collecter les métriques agrégées de performance.
type Metrics struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalTasks        int
	CompletedTasks    atomic.Int64
	SuccessfulTasks   atomic.Int64
	FailedTasks       atomic.Int64
	MatrixOpsCount    atomic.Int64
	CacheHits         atomic.Int64
	TempAllocsAvoided atomic.Int64
}

// NewMetrics initialise une nouvelle structure Metrics.
func NewMetrics(totalTasks int) *Metrics {
	return &Metrics{
		StartTime:  time.Now(),
		TotalTasks: totalTasks,
	}
}

// AddMatrixOps incrémente le compteur d'opérations matricielles de manière atomique.
func (m *Metrics) AddMatrixOps(n int64) {
	m.MatrixOpsCount.Add(n)
}

// AddCacheHit incrémente le compteur de cache hits de manière atomique.
func (m *Metrics) AddCacheHit() {
	m.CacheHits.Add(1)
}

// AddTempAllocsAvoided incrémente le compteur d'allocations temporaires *big.Int évitées de manière atomique.
func (m *Metrics) AddTempAllocsAvoided(n int64) {
	m.TempAllocsAvoided.Add(n)
}

// RecordTaskCompleted incrémente les compteurs appropriés quand une tâche se termine.
func (m *Metrics) RecordTaskCompleted(success bool) {
	m.CompletedTasks.Add(1)
	if success {
		m.SuccessfulTasks.Add(1)
	} else {
		m.FailedTasks.Add(1)
	}
}

// FibMatrix représente la matrice 2x2 [[a, b], [c, d]] utilisée pour le calcul de Fibonacci.
type FibMatrix struct {
	a, b, c, d *big.Int
}

// FibCalculator encapsule la logique de calcul, le cache LRU, les pools de ressources,
// la configuration et les métriques agrégées. Il est conçu pour être thread-safe.
type FibCalculator struct {
	lruCache   *lru.Cache[string, *big.Int]
	matrixPool sync.Pool
	bigIntPool sync.Pool
	config     Config
	metrics    *Metrics
}

// NewFibCalculator crée et initialise un nouveau calculateur Fibonacci en fonction de la configuration.
func NewFibCalculator(cfg Config, metrics *Metrics) *FibCalculator {
	fc := &FibCalculator{
		config:  cfg,
		metrics: metrics,
	}
	if cfg.EnableCache {
		var err error
		fc.lruCache, err = lru.New[string, *big.Int](cfg.CacheSize)
		if err != nil {
			log.Fatalf("FATAL: Impossible de créer le cache LRU avec taille %d : %v", cfg.CacheSize, err)
		}
		log.Printf("INFO: Cache LRU partagé activé (taille maximale: %d éléments)", cfg.CacheSize)
		fc.lruCache.Add("0", big.NewInt(0))
		fc.lruCache.Add("1", big.NewInt(1))
		fc.lruCache.Add("2", big.NewInt(1))
	} else {
		log.Println("INFO: Cache désactivé par la configuration.")
	}
	fc.matrixPool = sync.Pool{
		New: func() interface{} {
			return &FibMatrix{a: new(big.Int), b: new(big.Int), c: new(big.Int), d: new(big.Int)}
		},
	}
	fc.bigIntPool = sync.Pool{
		New: func() interface{} { return new(big.Int) },
	}
	return fc
}

// getTempBigInt récupère un *big.Int depuis le pool temporaire partagé.
func (fc *FibCalculator) getTempBigInt() *big.Int {
	bi := fc.bigIntPool.Get().(*big.Int)
	fc.metrics.AddTempAllocsAvoided(1)
	return bi
}

// putTempBigInt remet un *big.Int dans le pool temporaire partagé.
func (fc *FibCalculator) putTempBigInt(bi *big.Int) {
	fc.bigIntPool.Put(bi)
}

// getMatrix récupère une *FibMatrix depuis le pool partagé.
func (fc *FibCalculator) getMatrix() *FibMatrix {
	return fc.matrixPool.Get().(*FibMatrix)
}

// putMatrix remet une *FibMatrix dans le pool partagé.
func (fc *FibCalculator) putMatrix(m *FibMatrix) {
	fc.matrixPool.Put(m)
}

// multiplyMatrices (pas de changement)
func (fc *FibCalculator) multiplyMatrices(m1, m2, result *FibMatrix) {
	t1 := fc.getTempBigInt()
	t2 := fc.getTempBigInt()
	defer fc.putTempBigInt(t1)
	defer fc.putTempBigInt(t2)
	t1.Mul(m1.a, m2.a)
	t2.Mul(m1.b, m2.c)
	result.a.Add(t1, t2)
	t1.Mul(m1.a, m2.b)
	t2.Mul(m1.b, m2.d)
	result.b.Add(t1, t2)
	t1.Mul(m1.c, m2.a)
	t2.Mul(m1.d, m2.c)
	result.c.Add(t1, t2)
	t1.Mul(m1.c, m2.b)
	t2.Mul(m1.d, m2.d)
	result.d.Add(t1, t2)
}

// fastDoubling (Correction V1.6: suppression `totalIterations` et `iterationsDone`)
func (fc *FibCalculator) fastDoubling(ctx context.Context, n int, taskStartTime time.Time) (*big.Int, error) {
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 || n == 2 {
		return big.NewInt(1), nil
	}

	// --- Progression (variables de calcul supprimées car affichage commenté) ---
	// totalIterations := bits.Len(uint(n)) // SUPPRIMÉ V1.6
	// iterationsDone := 0                  // SUPPRIMÉ V1.6
	lastReportTime := taskStartTime // Conservé pour la logique de l'intervalle
	// --- Fin Initialisation Progression ---

	matrix := fc.getMatrix()
	result := fc.getMatrix()
	temp := fc.getMatrix()
	defer fc.putMatrix(matrix)
	defer fc.putMatrix(result)
	defer fc.putMatrix(temp)

	matrix.a.SetInt64(1)
	matrix.b.SetInt64(1)
	matrix.c.SetInt64(1)
	matrix.d.SetInt64(0)
	result.a.SetInt64(1)
	result.b.SetInt64(0)
	result.c.SetInt64(0)
	result.d.SetInt64(1)

	m := n
	for m > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if m&1 != 0 {
			fc.multiplyMatrices(result, matrix, temp)
			result, temp = temp, result
			fc.metrics.AddMatrixOps(1)
		}

		fc.multiplyMatrices(matrix, matrix, temp)
		matrix, temp = temp, matrix
		fc.metrics.AddMatrixOps(1)

		m >>= 1

		// --- Mise à jour et Affichage (commenté) de la Progression ---
		// iterationsDone++ // SUPPRIMÉ V1.6
		now := time.Now()
		// Vérifie si l'intervalle est écoulé (même si rien n'est affiché)
		if now.Sub(lastReportTime) >= ProgressReportInterval || m == 0 {
			// Calculs de progression supprimés en V1.5 et V1.6

			// Ligne d'affichage commentée :
			// fmt.Printf("\r[F(%d)] Progression: ... ", n)

			lastReportTime = now // Met à jour le temps du dernier rapport
		}
		// --- Fin Progression ---
	}

	finalResult := new(big.Int).Set(result.b)
	return finalResult, nil
}

// Calculate (pas de changement)
func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("l'index n doit être non-négatif, reçu %d", n)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	cacheKey := strconv.Itoa(n)
	if fc.config.EnableCache && fc.lruCache != nil {
		if val, ok := fc.lruCache.Get(cacheKey); ok {
			fc.metrics.AddCacheHit()
			return new(big.Int).Set(val), nil
		}
	}
	taskStartTime := time.Now()
	result, err := fc.fastDoubling(ctx, n, taskStartTime)
	if err != nil {
		return nil, err
	}
	if fc.config.EnableCache && fc.lruCache != nil {
		fc.lruCache.Add(cacheKey, new(big.Int).Set(result))
	}
	return result, nil
}

// formatScientific (pas de changement)
func formatScientific(num *big.Int, precision int) string {
	if num.Sign() == 0 {
		return fmt.Sprintf("0.0e+0")
	}
	floatPrec := uint(num.BitLen()) + uint(precision) + 10
	f := new(big.Float).SetPrec(floatPrec).SetInt(num)
	return f.Text('e', precision)
}

// TaskResult encapsule le résultat d'un calcul F(n).
type TaskResult struct {
	N      int
	Result *big.Int
	Err    error
}

// worker (pas de changement)
func worker(id int, ctx context.Context, wg *sync.WaitGroup, calculator *FibCalculator, tasks <-chan int, results chan<- TaskResult) {
	defer wg.Done()
	log.Printf("INFO: Worker %d démarré.", id)
	for {
		select {
		case n, ok := <-tasks:
			if !ok {
				log.Printf("INFO: Worker %d terminé (canal tâches fermé).", id)
				return
			}
			select {
			case <-ctx.Done():
				log.Printf("WARN: Worker %d: Contexte annulé avant de traiter F(%d).", id, n)
				results <- TaskResult{N: n, Result: nil, Err: ctx.Err()}
				calculator.metrics.RecordTaskCompleted(false)
				continue
			default:
				res, err := calculator.Calculate(ctx, n)
				results <- TaskResult{N: n, Result: res, Err: err}
				calculator.metrics.RecordTaskCompleted(err == nil)
			}
		case <-ctx.Done():
			log.Printf("INFO: Worker %d terminé (contexte global annulé: %v).", id, ctx.Err())
			return
		}
	}
}

// main (pas de changement majeur)
func main() {
	cfg := DefaultConfig()

	// --- Validation Config ---
	if len(cfg.NsToCalculate) == 0 {
		log.Fatalf("FATAL: La liste NsToCalculate est vide.")
	}
	if cfg.CacheSize <= 0 && cfg.EnableCache {
		log.Printf("WARN: CacheSize (%d) invalide. Désactivation du cache.", cfg.CacheSize)
		cfg.EnableCache = false
	}
	if cfg.Workers <= 0 {
		log.Printf("WARN: Workers (%d) invalide. Utilisation de runtime.NumCPU() = %d.", cfg.Workers, runtime.NumCPU())
		cfg.Workers = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(cfg.Workers)
	log.Printf("Configuration: %d tâches (F(n)), Timeout=%v, Workers=%d, GOMAXPROCS=%d, Cache=%t, CacheSize=%d, Profiling=%t",
		len(cfg.NsToCalculate), cfg.Timeout, cfg.Workers, runtime.GOMAXPROCS(-1), cfg.EnableCache, cfg.CacheSize, cfg.EnableProfiling)

	metrics := NewMetrics(len(cfg.NsToCalculate))
	var fCpu, fMem *os.File
	var err error

	// --- Profiling Setup ---
	if cfg.EnableProfiling {
		fCpu, err = os.Create("cpu.pprof")
		if err != nil {
			log.Fatalf("FATAL: Création cpu.pprof échouée: %v", err)
		}
		defer fCpu.Close()
		if err := pprof.StartCPUProfile(fCpu); err != nil {
			log.Fatalf("FATAL: Démarrage profilage CPU échoué: %v", err)
		}
		defer pprof.StopCPUProfile()
		log.Println("INFO: Profilage CPU activé -> cpu.pprof")
		fMem, err = os.Create("mem.pprof")
		if err != nil {
			log.Printf("WARN: Création mem.pprof échouée: %v. Profilage mémoire désactivé.", err)
			fMem = nil
		} else {
			defer fMem.Close()
			log.Println("INFO: Profilage Mémoire activé -> mem.pprof")
		}
	}

	fc := NewFibCalculator(cfg, metrics)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// --- Worker Pool Setup & Execution ---
	var wg sync.WaitGroup
	tasks := make(chan int, len(cfg.NsToCalculate))
	results := make(chan TaskResult, len(cfg.NsToCalculate))

	log.Printf("INFO: Démarrage de %d workers...", cfg.Workers)
	for i := 1; i <= cfg.Workers; i++ {
		wg.Add(1)
		go worker(i, ctx, &wg, fc, tasks, results)
	}

	go func() { // Feed tasks
		defer close(tasks)
		log.Println("INFO: Envoi des tâches aux workers...")
		for _, n := range cfg.NsToCalculate {
			select {
			case tasks <- n:
			case <-ctx.Done():
				log.Printf("WARN: Alimentation des tâches interrompue car contexte annulé: %v", ctx.Err())
				return
			}
		}
		log.Println("INFO: Toutes les tâches ont été envoyées.")
	}()

	go func() { // Close results when workers done
		wg.Wait()
		close(results)
		log.Println("INFO: Tous les workers ont terminé.")
	}()

	// --- Collect Results ---
	log.Println("INFO: En attente des résultats des workers...")
	finalResults := make(map[int]*big.Int)
	errorsMap := make(map[int]error)
	for taskResult := range results {
		if taskResult.Err != nil {
			errorsMap[taskResult.N] = taskResult.Err // Store error (context or other)
		} else {
			finalResults[taskResult.N] = taskResult.Result
		}
	}
	metrics.EndTime = time.Now()
	log.Println("INFO: Collecte des résultats terminée.")

	if ctx.Err() != nil {
		log.Printf("WARN: L'opération globale a été terminée prématurément par le contexte: %v (Timeout: %v)", ctx.Err(), cfg.Timeout)
	}

	// --- Display Results & Metrics ---
	totalDuration := metrics.EndTime.Sub(metrics.StartTime)
	fmt.Printf("\n=== Résultats Agrégés pour %d tâches (Timeout: %v, Workers: %d) ===\n",
		metrics.TotalTasks, cfg.Timeout, cfg.Workers)
	fmt.Printf("Temps total d'exécution                     : %v\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("Tâches complétées (succès ou échec géré)    : %d / %d\n", metrics.CompletedTasks.Load(), metrics.TotalTasks)
	fmt.Printf("Tâches réussies                             : %d\n", metrics.SuccessfulTasks.Load())
	fmt.Printf("Tâches échouées (erreur ou timeout indiv.)  : %d\n", metrics.FailedTasks.Load())
	fmt.Printf("Opérations matricielles (total)             : %d\n", metrics.MatrixOpsCount.Load())
	if cfg.EnableCache {
		fmt.Printf("Cache hits (total)                          : %d\n", metrics.CacheHits.Load())
		if fc.lruCache != nil {
			fmt.Printf("Cache LRU taille actuelle / max             : %d / %d\n", fc.lruCache.Len(), cfg.CacheSize)
		}
	} else {
		fmt.Println("Cache                                       : Désactivé")
	}
	fmt.Printf("Allocations *big.Int évitées (total)      : %d\n", metrics.TempAllocsAvoided.Load())

	fmt.Printf("\n--- Détails des Résultats ---\n")
	keys := make([]int, 0, len(cfg.NsToCalculate))
	uniqueNs := make(map[int]struct{})
	for _, n := range cfg.NsToCalculate {
		if _, exists := uniqueNs[n]; !exists {
			keys = append(keys, n)
			uniqueNs[n] = struct{}{}
		}
	}
	sort.Ints(keys)

	const maxDigitsDisplay = 50
	for _, n := range keys {
		if result, ok := finalResults[n]; ok {
			s := result.String()
			numDigits := len(s)
			fmt.Printf("F(%d) [OK] (%d chiffres): ", n, numDigits)
			if numDigits <= maxDigitsDisplay {
				fmt.Printf("%s\n", s)
			} else {
				fmt.Printf("%s...%s (Sci: %s)\n", s[:maxDigitsDisplay/2], s[numDigits-maxDigitsDisplay/2:], formatScientific(result, cfg.Precision))
			}
		} else if errResult, ok := errorsMap[n]; ok {
			errMsg := "ERREUR"
			if errors.Is(errResult, context.Canceled) {
				errMsg = "ANNULÉ (CTX)"
			} else if errors.Is(errResult, context.DeadlineExceeded) {
				errMsg = "TIMEOUT (CTX)"
			}
			fmt.Printf("F(%d) [%s]: %v\n", n, errMsg, errResult)
		} else {
			fmt.Printf("F(%d) [NON TRAITÉ / TIMEOUT GLOBAL]\n", n)
		}
	}

	// --- Final Memory Profile ---
	if cfg.EnableProfiling && fMem != nil {
		log.Println("INFO: Écriture du profil mémoire final (heap)...")
		runtime.GC()
		if err := pprof.WriteHeapProfile(fMem); err != nil {
			log.Printf("WARN: Impossible d'écrire le profil mémoire final: %v", err)
		} else {
			log.Printf("INFO: Profil mémoire final sauvegardé dans '%s'", fMem.Name())
		}
	}
	log.Println("INFO: Programme terminé.")
}
