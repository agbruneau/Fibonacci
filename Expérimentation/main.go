// =============================================================================
// Programme : Test de Primalité Parallèle pour Grands Nombres en Go
// Auteur    : Adapté par l'IA Gemini depuis la structure Fibonacci v1.6
// Date      : 2025-04-03 // Date de la modification
// Version   : 1.1 // Corrections erreurs compilation (unused var, string.Contains)
//
// Description :
// Version 1.1:
// - Correction de la variable 'displayStr' non utilisée dans l'affichage des résultats.
// - Correction de l'appel à strings.Contains pour la détection d'erreur de conversion.
// - Ajout de l'import du package "strings".
// Version 1.0 (Basé sur Fibonacci v1.6 structure):
// - Teste la primalité (probable) pour une liste de grands nombres (strings).
// - Utilise un pool de workers parallèles et Miller-Rabin (ProbablyPrime).
// - Inclut un cache LRU optionnel et la gestion du timeout.
// =============================================================================

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"runtime/pprof" // Utilisé pour la clé de cache et la détection d'erreur
	"strings"       // ***** AJOUTÉ V1.1 ***** pour strings.Contains
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// --- Structures ---

// Config structure pour les paramètres de configuration.
type Config struct {
	NumbersToTest         []string
	Timeout               time.Duration
	Workers               int
	MillerRabinIterations int
	EnableCache           bool
	CacheSize             int
	EnableProfiling       bool
}

// DefaultConfig retourne la configuration par défaut.
func DefaultConfig() Config {
	exampleNumbers := []string{
		"0", "1", "2", "3", "4", "10", "17",
		"7919",   // Premier
		"104729", // Premier
		"6", "100",
		"104730", // Composé (pair)
		"123456789012345678901234567890123456789", // Très grand, probablement composé
		"170141183460469231731687303715884105727", // 2^127 - 1 (Mersenne prime)
		"7919", // Doublon
		"11111111111111111111111111111111111111111111111111111111111111111111111111111111", // Grand composé
		"104729",         // Doublon
		"invalid-number", // Pour tester la gestion d'erreur
		"170141183460469231731687303715884105727", // Doublon
	}
	numWorkers := runtime.NumCPU()
	return Config{
		NumbersToTest:         exampleNumbers,
		Timeout:               5 * time.Minute,
		Workers:               numWorkers,
		MillerRabinIterations: 20,
		EnableCache:           true,
		CacheSize:             1024,
		EnableProfiling:       false,
	}
}

// Metrics structure pour collecter les métriques agrégées de performance.
type Metrics struct {
	StartTime       time.Time
	EndTime         time.Time
	TotalTasks      int
	CompletedTasks  atomic.Int64
	SuccessfulTasks atomic.Int64
	FailedTasks     atomic.Int64
	CacheHits       atomic.Int64
}

// NewMetrics initialise une nouvelle structure Metrics.
func NewMetrics(totalTasks int) *Metrics {
	return &Metrics{
		StartTime:  time.Now(),
		TotalTasks: totalTasks,
	}
}

// AddCacheHit incrémente le compteur de cache hits de manière atomique.
func (m *Metrics) AddCacheHit() {
	m.CacheHits.Add(1)
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

// PrimalityTester encapsule la logique de test, le cache LRU, etc.
type PrimalityTester struct {
	lruCache *lru.Cache[string, bool]
	config   Config
	metrics  *Metrics
}

// NewPrimalityTester crée et initialise un nouveau testeur de primalité.
func NewPrimalityTester(cfg Config, metrics *Metrics) *PrimalityTester {
	pt := &PrimalityTester{
		config:  cfg,
		metrics: metrics,
	}
	if cfg.EnableCache {
		var err error
		pt.lruCache, err = lru.New[string, bool](cfg.CacheSize)
		if err != nil {
			log.Fatalf("FATAL: Impossible de créer le cache LRU: %v", err)
		}
		log.Printf("INFO: Cache LRU partagé activé (taille max: %d)", cfg.CacheSize)
		pt.lruCache.Add("0", false)
		pt.lruCache.Add("1", false)
		pt.lruCache.Add("2", true)
		pt.lruCache.Add("3", true)
	} else {
		log.Println("INFO: Cache désactivé.")
	}
	return pt
}

// probablyPrimeInternal (pas de changement)
func probablyPrimeInternal(n *big.Int, k int) bool {
	if n.Cmp(big.NewInt(2)) < 0 {
		return false
	}
	if n.Cmp(big.NewInt(4)) < 0 {
		return true
	}
	if n.Bit(0) == 0 {
		return false
	}
	return n.ProbablyPrime(k)
}

// TestPrimality (pas de changement)
func (pt *PrimalityTester) TestPrimality(ctx context.Context, nStr string) (isPrime bool, err error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	cacheKey := nStr
	if pt.config.EnableCache && pt.lruCache != nil {
		if val, ok := pt.lruCache.Get(cacheKey); ok {
			pt.metrics.AddCacheHit()
			return val, nil
		}
	}
	n := new(big.Int)
	_, success := n.SetString(nStr, 10)
	if !success {
		return false, fmt.Errorf("impossible de convertir '%s' en big.Int", nStr)
	}
	isPrimeResult := probablyPrimeInternal(n, pt.config.MillerRabinIterations)
	if pt.config.EnableCache && pt.lruCache != nil {
		pt.lruCache.Add(cacheKey, isPrimeResult)
	}
	return isPrimeResult, nil
}

// TaskResult encapsule le résultat d'un test de primalité.
type TaskResult struct {
	NStr    string
	IsPrime bool
	Err     error
}

// worker (correction bug logique RecordTaskCompleted)
func worker(id int, ctx context.Context, wg *sync.WaitGroup, tester *PrimalityTester, tasks <-chan string, results chan<- TaskResult) {
	defer wg.Done()
	log.Printf("INFO: Worker %d démarré.", id)
	for {
		select {
		case nStr, ok := <-tasks:
			if !ok {
				log.Printf("INFO: Worker %d terminé (canal tâches fermé).", id)
				return
			}
			select {
			case <-ctx.Done():
				log.Printf("WARN: Worker %d: Contexte annulé avant de tester '%s'.", id, nStr)
				taskErr := ctx.Err()
				results <- TaskResult{NStr: nStr, Err: taskErr}
				// Une tâche annulée avant traitement est considérée comme échouée
				tester.metrics.RecordTaskCompleted(false)
				continue
			default:
				isPrime, err := tester.TestPrimality(ctx, nStr)
				results <- TaskResult{NStr: nStr, IsPrime: isPrime, Err: err}
				// Le test est réussi si err est nil (signifie qu'on a un résultat valide PRIME/COMPOSITE)
				// Les erreurs de contexte qui pourraient survenir PENDANT TestPrimality sont gérées ici.
				// Les erreurs de conversion sont considérées comme un échec de la tâche.
				isSuccess := (err == nil)
				tester.metrics.RecordTaskCompleted(isSuccess)
			}
		case <-ctx.Done():
			log.Printf("INFO: Worker %d terminé (contexte global annulé: %v).", id, ctx.Err())
			return
		}
	}
}

// main (correction affichage et détection erreur conversion)
func main() {
	// --- Configuration & Validation ---
	cfg := DefaultConfig()
	if len(cfg.NumbersToTest) == 0 {
		log.Fatalf("FATAL: NumbersToTest est vide.")
	}
	if cfg.CacheSize <= 0 && cfg.EnableCache {
		cfg.EnableCache = false
		log.Println("WARN: Cache désactivé (taille invalide).")
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
		log.Printf("WARN: Workers invalide, utilise %d.", cfg.Workers)
	}
	if cfg.MillerRabinIterations <= 0 {
		cfg.MillerRabinIterations = 1
		log.Println("WARN: MillerRabinIterations invalide, utilise 1.")
	}
	runtime.GOMAXPROCS(cfg.Workers)
	log.Printf("Configuration: %d tests, Timeout=%v, Workers=%d, GOMAXPROCS=%d, Iterations=%d, Cache=%t, CacheSize=%d, Profiling=%t",
		len(cfg.NumbersToTest), cfg.Timeout, cfg.Workers, runtime.GOMAXPROCS(-1), cfg.MillerRabinIterations, cfg.EnableCache, cfg.CacheSize, cfg.EnableProfiling)

	metrics := NewMetrics(len(cfg.NumbersToTest))
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
			log.Printf("WARN: Création mem.pprof échouée: %v.", err)
			fMem = nil
		} else {
			defer fMem.Close()
			log.Println("INFO: Profilage Mémoire activé -> mem.pprof")
		}
	}

	pt := NewPrimalityTester(cfg, metrics)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// --- Worker Pool Setup & Execution ---
	var wg sync.WaitGroup
	tasks := make(chan string, len(cfg.NumbersToTest))
	results := make(chan TaskResult, len(cfg.NumbersToTest))

	log.Printf("INFO: Démarrage de %d workers...", cfg.Workers)
	for i := 1; i <= cfg.Workers; i++ {
		wg.Add(1)
		go worker(i, ctx, &wg, pt, tasks, results)
	}
	go func() { // Feed tasks
		defer close(tasks)
		log.Println("INFO: Envoi des tâches...")
		for _, nStr := range cfg.NumbersToTest {
			select {
			case tasks <- nStr:
			case <-ctx.Done():
				log.Printf("WARN: Alimentation interrompue: %v", ctx.Err())
				return
			}
		}
		log.Println("INFO: Toutes les tâches envoyées.")
	}()
	go func() { wg.Wait(); close(results); log.Println("INFO: Tous les workers terminés.") }() // Close results

	// --- Collect Results ---
	log.Println("INFO: En attente des résultats...")
	finalResults := make(map[string]bool)
	errorsMap := make(map[string]error)
	for taskResult := range results {
		if taskResult.Err != nil {
			errorsMap[taskResult.NStr] = taskResult.Err
		} else {
			finalResults[taskResult.NStr] = taskResult.IsPrime
		}
	}
	metrics.EndTime = time.Now()
	log.Println("INFO: Collecte terminée.")
	if ctx.Err() != nil {
		log.Printf("WARN: Opération terminée par contexte: %v (Timeout: %v)", ctx.Err(), cfg.Timeout)
	}

	// --- Display Results & Metrics ---
	totalDuration := metrics.EndTime.Sub(metrics.StartTime)
	fmt.Printf("\n=== Résultats Agrégés pour %d tests (Timeout: %v, Workers: %d) ===\n",
		metrics.TotalTasks, cfg.Timeout, cfg.Workers)
	fmt.Printf("Temps total d'exécution       : %v\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("Tâches complétées             : %d / %d\n", metrics.CompletedTasks.Load(), metrics.TotalTasks)
	fmt.Printf("Tâches réussies (résultat OK) : %d\n", metrics.SuccessfulTasks.Load())
	fmt.Printf("Tâches échouées (err/ctx)     : %d\n", metrics.FailedTasks.Load())
	if cfg.EnableCache {
		fmt.Printf("Cache hits (total)            : %d\n", metrics.CacheHits.Load())
		if pt.lruCache != nil {
			fmt.Printf("Cache LRU taille actuelle/max : %d / %d\n", pt.lruCache.Len(), cfg.CacheSize)
		}
	} else {
		fmt.Println("Cache                         : Désactivé")
	}

	fmt.Printf("\n--- Détails des Résultats ---\n")
	displayed := make(map[string]int)
	for _, nStr := range cfg.NumbersToTest {
		count := displayed[nStr]
		displayed[nStr]++
		// *** CORRECTION V1.1: Utilisation de displayStr ici ***
		displayStr := nStr
		if count > 0 {
			displayStr = fmt.Sprintf("%s (%d)", nStr, count+1)
		} // Marque les doublons

		const maxDisplayLen = 60
		trimmedDisplayStr := displayStr // Tronque l'affichage si trop long
		if len(displayStr) > maxDisplayLen {
			// Calcule le point milieu correctement même avec le suffixe "(N)"
			prefixLen := maxDisplayLen / 2
			suffixStart := len(displayStr) - (maxDisplayLen / 2)
			if suffixStart <= prefixLen {
				suffixStart = prefixLen + 1
			} // Évite chevauchement
			trimmedDisplayStr = fmt.Sprintf("%s...%s", displayStr[:prefixLen], displayStr[suffixStart:])
		}

		if isPrime, ok := finalResults[nStr]; ok {
			resultStr := "COMPOSITE"
			if isPrime {
				resultStr = "PROBABLEMENT PREMIER"
			}
			// *** CORRECTION V1.1: Utilisation de trimmedDisplayStr ***
			fmt.Printf("'%s': [%s]\n", trimmedDisplayStr, resultStr)
		} else if errResult, ok := errorsMap[nStr]; ok {
			errMsg := "ERREUR"
			// *** CORRECTION V1.1: Utilisation de strings.Contains ***
			if errors.Is(errResult, context.Canceled) {
				errMsg = "ANNULÉ (CTX)"
			} else if errors.Is(errResult, context.DeadlineExceeded) {
				errMsg = "TIMEOUT (CTX)"
			} else if strings.Contains(errResult.Error(), "impossible de convertir") { // Vérifie le message d'erreur de conversion
				errMsg = "ENTRÉE INVALIDE"
			}
			// *** CORRECTION V1.1: Utilisation de trimmedDisplayStr ***
			fmt.Printf("'%s': [%s] %v\n", trimmedDisplayStr, errMsg, errResult)
		} else {
			// *** CORRECTION V1.1: Utilisation de trimmedDisplayStr ***
			fmt.Printf("'%s': [NON TRAITÉ / TIMEOUT GLOBAL]\n", trimmedDisplayStr)
		}
	}

	// --- Final Memory Profile ---
	if cfg.EnableProfiling && fMem != nil {
		log.Println("INFO: Écriture du profil mémoire final...")
		runtime.GC()
		if err := pprof.WriteHeapProfile(fMem); err != nil {
			log.Printf("WARN: Échec écriture profil mémoire: %v", err)
		} else {
			log.Printf("INFO: Profil mémoire sauvegardé: '%s'", fMem.Name())
		}
	}
	log.Println("INFO: Programme terminé.")
}
