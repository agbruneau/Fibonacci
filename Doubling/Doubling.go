// =============================================================================
// Programme : Calcul ultra-optimisé de Fibonacci(n) en Go
// Auteur    : André-Guy Bruneau // Adapté par l'IA Gemini 2.5 PRo Experimental 03-2025
// Date      : 2025-03-26 // Date de la modification
// Version   : 1.2 // Intégration LRU Cache + String Keys
//
// Description :
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
	"fmt"
	"log"
	"math/big"
	"math/bits" // Importé pour bits.Len
	"os"
	"runtime"
	"runtime/pprof"
	"strconv" // Importé pour convertir int en string pour la clé de cache
	"sync"
	"sync/atomic"
	"time"

	// --- Dépendance Externe ---
	// Nécessite : go get github.com/hashicorp/golang-lru/v2
	lru "github.com/hashicorp/golang-lru/v2"
)

// --- Constantes ---
const (
	// ProgressReportInterval : Fréquence de mise à jour de l'indicateur de progression.
	ProgressReportInterval = 1 * time.Second
)

// Configuration des paramètres
type Config struct {
	N               int           // Calculer Fibonacci(N)
	Timeout         time.Duration // Durée max d'exécution
	Precision       int           // Chiffres significatifs après la virgule pour l'affichage scientifique
	Workers         int           // Nombre de threads CPU à utiliser (GOMAXPROCS)
	EnableCache     bool          // Activer le cache LRU
	CacheSize       int           // Taille maximale du cache LRU (nombre d'éléments)
	EnableProfiling bool          // Activer le profiling CPU/mémoire via pprof
}

// DefaultConfig retourne la configuration par défaut.
func DefaultConfig() Config {
	return Config{
		N:               10000000, // Exemple de grande valeur pour tester la progression
		Timeout:         5 * time.Minute,
		Precision:       10,
		Workers:         runtime.NumCPU(),
		EnableCache:     true,
		CacheSize:       2048,  // Taille par défaut du cache LRU
		EnableProfiling: false, // Mettre à true pour générer les fichiers pprof
	}
}

// Metrics structure pour les métriques de performance.
type Metrics struct {
	StartTime            time.Time
	EndTime              time.Time
	CalculationStartTime time.Time    // Heure de début spécifique au calcul pur
	CalculationEndTime   time.Time    // Heure de fin spécifique au calcul pur
	MatrixOpsCount       atomic.Int64 // Utilisation de atomic.Int64 directement
	CacheHits            atomic.Int64 // Utilisation de atomic.Int64 directement
	TempAllocsAvoided    atomic.Int64 // Utilisation de atomic.Int64 directement
}

// NewMetrics initialise une nouvelle structure Metrics.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// AddMatrixOps incrémente le compteur d'opérations matricielles de manière atomique.
func (m *Metrics) AddMatrixOps(n int64) {
	m.MatrixOpsCount.Add(n)
}

// AddCacheHit incrémente le compteur de cache hits de manière atomique.
func (m *Metrics) AddCacheHit() {
	m.CacheHits.Add(1)
}

// AddTempAllocsAvoided incrémente le compteur d'allocations temporaires évitées de manière atomique.
func (m *Metrics) AddTempAllocsAvoided(n int64) {
	m.TempAllocsAvoided.Add(n)
}

// CalculationDuration retourne la durée du calcul pur.
func (m *Metrics) CalculationDuration() time.Duration {
	// Gère le cas où le calcul n'a pas encore commencé ou fini
	if m.CalculationStartTime.IsZero() || m.CalculationEndTime.IsZero() {
		return 0
	}
	return m.CalculationEndTime.Sub(m.CalculationStartTime)
}

// FibMatrix représente la matrice 2x2 [[a, b], [c, d]] pour le calcul de Fibonacci.
// Utilise des *big.Int directement pour éviter une indirection supplémentaire.
type FibMatrix struct {
	a, b, c, d *big.Int
}

// FibCalculator encapsule la logique de calcul, le cache, les pools et les métriques.
type FibCalculator struct {
	lruCache   *lru.Cache[string, *big.Int] // Cache LRU (thread-safe) avec clé string
	matrixPool sync.Pool                    // Pool pour réutiliser les structures FibMatrix
	bigIntPool sync.Pool                    // Pool pour réutiliser les *big.Int temporaires
	config     Config
	metrics    *Metrics
}

// NewFibCalculator crée et initialise un nouveau calculateur Fibonacci.
func NewFibCalculator(cfg Config) *FibCalculator {
	fc := &FibCalculator{
		// lruCache initialisé ci-dessous
		config:  cfg,
		metrics: NewMetrics(),
	}

	// Initialisation du cache LRU s'il est activé
	if cfg.EnableCache {
		var err error
		// Utilise une clé string et une valeur *big.Int. La bibliothèque LRU gère la synchronisation.
		fc.lruCache, err = lru.New[string, *big.Int](cfg.CacheSize)
		if err != nil {
			// Devrait seulement arriver si CacheSize <= 0, géré par config validation ou défauts.
			log.Fatalf("FATAL: Impossible de créer le cache LRU : %v", err)
		}
		log.Printf("INFO: Cache LRU activé (taille: %d)", cfg.CacheSize)

		// Pré-remplissage du cache avec les cas de base
		// Utilise les clés string correspondantes.
		fc.lruCache.Add("0", big.NewInt(0))
		fc.lruCache.Add("1", big.NewInt(1))
		fc.lruCache.Add("2", big.NewInt(1))
	} else {
		log.Println("INFO: Cache désactivé.")
	}

	// Initialisation du pool pour FibMatrix
	fc.matrixPool = sync.Pool{
		New: func() interface{} {
			return &FibMatrix{
				a: new(big.Int), b: new(big.Int),
				c: new(big.Int), d: new(big.Int),
			}
		},
	}

	// Initialisation du pool pour les *big.Int temporaires
	fc.bigIntPool = sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}

	return fc
}

// getTempBigInt récupère un *big.Int du pool temporaire.
func (fc *FibCalculator) getTempBigInt() *big.Int {
	bi := fc.bigIntPool.Get().(*big.Int)
	fc.metrics.AddTempAllocsAvoided(1)
	return bi
}

// putTempBigInt remet un *big.Int dans le pool temporaire.
func (fc *FibCalculator) putTempBigInt(bi *big.Int) {
	fc.bigIntPool.Put(bi)
}

// getMatrix récupère une *FibMatrix du pool.
func (fc *FibCalculator) getMatrix() *FibMatrix {
	m := fc.matrixPool.Get().(*FibMatrix)
	return m
}

// putMatrix remet une *FibMatrix dans le pool.
func (fc *FibCalculator) putMatrix(m *FibMatrix) {
	fc.matrixPool.Put(m)
}

// multiplyMatrices multiplie deux matrices 2x2 (m1 * m2 = result).
// Utilise **deux** *big.Int temporaires du pool pour minimiser les allocations.
// ATTENTION : result NE DOIT PAS être le même pointeur que m1 ou m2.
func (fc *FibCalculator) multiplyMatrices(m1, m2, result *FibMatrix) {
	t1 := fc.getTempBigInt()
	t2 := fc.getTempBigInt()
	defer fc.putTempBigInt(t1)
	defer fc.putTempBigInt(t2)

	// Calculs (inchangés)
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

	// Le comptage se fait dans fastDoubling
}

// fastDoubling calcule Fibonacci(n) avec l'algorithme de doublement matriciel optimisé,
// affiche la progression et respecte l'annulation via le contexte.
// Retourne le résultat et une erreur (nil si succès, ctx.Err() si annulé).
func (fc *FibCalculator) fastDoubling(ctx context.Context, n int, calcStartTime time.Time) (*big.Int, error) {
	// Cas de base gérés avant l'appel (dans Calculate) ou ici pour robustesse
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 || n == 2 {
		return big.NewInt(1), nil
	}

	// --- Initialisation pour la progression (inchangée) ---
	totalIterations := bits.Len(uint(n))
	iterationsDone := 0
	lastReportTime := calcStartTime
	// --- Fin Initialisation Progression ---

	// Récupération des matrices depuis le pool (inchangé)
	matrix := fc.getMatrix()
	result := fc.getMatrix()
	temp := fc.getMatrix()
	defer fc.putMatrix(matrix)
	defer fc.putMatrix(result)
	defer fc.putMatrix(temp)

	// Initialisation des matrices (inchangée)
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
		// --- Vérification du contexte (inchangée) ---
		select {
		case <-ctx.Done():
			log.Printf("\nINFO: Calcul interrompu (%v).", ctx.Err())
			fmt.Println()
			return nil, ctx.Err()
		default:
			// Continue
		}
		// --- Fin Vérification du contexte ---

		// --- Logique Fast Doubling (inchangée) ---
		if m&1 != 0 {
			fc.multiplyMatrices(result, matrix, temp)
			result, temp = temp, result
			fc.metrics.AddMatrixOps(1)
		}

		fc.multiplyMatrices(matrix, matrix, temp)
		matrix, temp = temp, matrix
		fc.metrics.AddMatrixOps(1)

		m >>= 1
		// --- Fin Logique Fast Doubling ---

		// --- Mise à jour et Affichage Progression (inchangée) ---
		iterationsDone++
		now := time.Now()
		if now.Sub(lastReportTime) >= ProgressReportInterval || m == 0 {
			elapsed := now.Sub(calcStartTime)
			var progress float64
			if totalIterations > 0 {
				progress = float64(iterationsDone) / float64(totalIterations) * 100.0
			} else {
				progress = 100.0
			}
			fmt.Printf("\rProgress: %.2f%% (%d/%d bits), Elapsed: %v      ",
				progress, iterationsDone, totalIterations, elapsed.Round(time.Millisecond))
			lastReportTime = now
		}
		// --- Fin Progression ---
	}

	fmt.Println() // Saut de ligne après progression

	// Le résultat F(n) se trouve dans result.b
	// Crée une copie pour le retour (inchangé)
	finalResult := new(big.Int).Set(result.b)
	return finalResult, nil
}

// Calculate gère le cache LRU et lance le calcul via fastDoubling.
// Accepte un context.Context pour l'annulation.
func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("l'index n doit être non-négatif, reçu %d", n)
	}

	// 0. Vérification initiale du contexte (inchangée)
	select {
	case <-ctx.Done():
		log.Printf("WARN: Contexte annulé avant même le début du calcul pour F(%d): %v", n, ctx.Err())
		return nil, ctx.Err()
	default:
		// Continue
	}

	// Clé pour le cache LRU (string)
	cacheKey := strconv.Itoa(n)

	// 1. Vérification du cache LRU (si activé)
	// Pas besoin de mutex externe, la bibliothèque LRU est thread-safe.
	if fc.config.EnableCache && fc.lruCache != nil {
		if val, ok := fc.lruCache.Get(cacheKey); ok {
			fc.metrics.AddCacheHit()
			now := time.Now()
			fc.metrics.CalculationStartTime = now // Marque comme instantané
			fc.metrics.CalculationEndTime = now
			// Retourne une copie pour éviter que l'appelant modifie la valeur cachée accidentellement
			// (même si Get retourne un pointeur, la bonne pratique est de copier)
			return new(big.Int).Set(val), nil
		}
	}

	// 2. Lancement du calcul via Fast Doubling (inchangé)
	fc.metrics.CalculationStartTime = time.Now()
	result, err := fc.fastDoubling(ctx, n, fc.metrics.CalculationStartTime)
	if err != nil {
		return nil, fmt.Errorf("le calcul fastDoubling a échoué: %w", err)
	}
	fc.metrics.CalculationEndTime = time.Now()

	// 3. Mise en cache du résultat dans LRU (si activé et calcul réussi)
	// Pas besoin de mutex externe. Add est thread-safe.
	if fc.config.EnableCache && fc.lruCache != nil {
		// Met en cache une copie pour la sécurité (même si fastDoubling retourne une nouvelle instance)
		fc.lruCache.Add(cacheKey, new(big.Int).Set(result))
	}

	// Retourne le résultat calculé
	return result, nil
}

// formatScientific formate un *big.Int en notation scientifique avec une précision donnée.
// (inchangé)
func formatScientific(num *big.Int, precision int) string {
	if num.Sign() == 0 {
		return "0.0e+0"
	}
	floatPrec := uint(num.BitLen()) + uint(precision) + 10 // Marge de sécurité
	f := new(big.Float).SetPrec(floatPrec).SetInt(num)
	return f.Text('e', precision)
}

func main() {
	cfg := DefaultConfig()
	// --- Validation simple de la configuration ---
	if cfg.CacheSize <= 0 && cfg.EnableCache {
		log.Printf("WARN: CacheSize (%d) invalide, désactivation du cache.", cfg.CacheSize)
		cfg.EnableCache = false
	}
	if cfg.N < 0 {
		log.Fatalf("FATAL: N (%d) ne peut pas être négatif.", cfg.N)
	}

	runtime.GOMAXPROCS(cfg.Workers)
	log.Printf("Configuration: N=%d, Timeout=%v, Workers=%d, Cache=%t, CacheSize=%d, Profiling=%t, Précision Affichage=%d",
		cfg.N, cfg.Timeout, cfg.Workers, cfg.EnableCache, cfg.CacheSize, cfg.EnableProfiling, cfg.Precision)

	var fCpu, fMem *os.File
	var err error

	// --- Configuration du Profiling (inchangée) ---
	if cfg.EnableProfiling {
		fCpu, err = os.Create("cpu.pprof")
		if err != nil {
			log.Fatalf("FATAL: Impossible de créer le fichier de profil CPU: %v", err)
		}
		defer fCpu.Close()
		if err := pprof.StartCPUProfile(fCpu); err != nil {
			log.Fatalf("FATAL: Impossible de démarrer le profil CPU: %v", err)
		}
		defer pprof.StopCPUProfile()
		log.Println("INFO: Profiling CPU activé. Fichier: cpu.pprof")

		fMem, err = os.Create("mem.pprof")
		if err != nil {
			log.Printf("WARN: Impossible de créer le fichier de profil mémoire: %v. Profiling mémoire désactivé.", err)
			fMem = nil
		} else {
			defer fMem.Close()
			log.Println("INFO: Profiling Mémoire activé. Fichier: mem.pprof (sera écrit à la fin)")
		}
	}

	// --- Initialisation du calculateur ---
	fc := NewFibCalculator(cfg) // Utilise la config validée

	// --- Contexte pour le Timeout (inchangé) ---
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// --- Lancement du calcul dans une goroutine séparée (inchangé) ---
	resultChan := make(chan *big.Int, 1)
	errChan := make(chan error, 1)

	go func() {
		log.Printf("INFO: Démarrage du calcul de Fibonacci(%d)... (Timeout: %v)", cfg.N, cfg.Timeout)
		res, err := fc.Calculate(ctx, cfg.N) // Passe le contexte

		if err != nil {
			// Si l'erreur N'EST PAS context.Canceled ou context.DeadlineExceeded,
			// et que le contexte lui-même n'est pas encore Done (évite race condition au log)
			// alors c'est une autre erreur de calcul qu'il faut signaler.
			if !(err == context.Canceled || err == context.DeadlineExceeded || ctx.Err() != nil) {
				errChan <- fmt.Errorf("erreur interne dans fc.Calculate: %w", err)
			}
			// Si c'était une erreur de contexte, main s'en chargera via select.
			return
		}

		// Si succès
		fc.metrics.EndTime = time.Now()
		resultChan <- res
	}()

	// --- Attente du résultat, de l'erreur ou du timeout (inchangé) ---
	var result *big.Int
	log.Println("INFO: En attente du résultat ou du timeout...")

	select {
	case <-ctx.Done():
		log.Printf("FATAL: Opération annulée ou timeout (%v) dépassé. Raison: %v", cfg.Timeout, ctx.Err())
		// Tentative écriture profil mémoire (inchangé)
		if cfg.EnableProfiling && fMem != nil {
			log.Println("INFO: Tentative d'écriture du profil mémoire après timeout/annulation...")
			runtime.GC()
			if err := pprof.WriteHeapProfile(fMem); err != nil {
				log.Printf("WARN: Impossible d'écrire le profil mémoire dans %s: %v", fMem.Name(), err)
			} else {
				log.Printf("INFO: Profil mémoire sauvegardé dans %s", fMem.Name())
			}
		}
		os.Exit(1)

	case err := <-errChan:
		log.Fatalf("FATAL: Erreur interne lors du calcul: %v", err)

	case result = <-resultChan:
		calculationDuration := fc.metrics.CalculationDuration()
		log.Printf("INFO: Calcul terminé avec succès. Durée calcul pur: %v", calculationDuration.Round(time.Millisecond))
	}

	// --- Affichage des résultats et métriques (si succès) (inchangé) ---
	if result != nil {
		fmt.Printf("\n=== Résultats Fibonacci(%d) ===\n", cfg.N)
		totalDuration := fc.metrics.EndTime.Sub(fc.metrics.StartTime)
		calculationDuration := fc.metrics.CalculationDuration()

		fmt.Printf("Temps total d'exécution                     : %v\n", totalDuration.Round(time.Millisecond))
		fmt.Printf("Temps de calcul pur (fastDoubling)          : %v\n", calculationDuration.Round(time.Millisecond))
		fmt.Printf("Opérations matricielles (multiplications)   : %d\n", fc.metrics.MatrixOpsCount.Load())
		if cfg.EnableCache {
			fmt.Printf("Cache hits                                  : %d\n", fc.metrics.CacheHits.Load())
			// Pourrait ajouter fc.lruCache.Len() pour voir la taille actuelle vs max
			if fc.lruCache != nil {
				fmt.Printf("Cache LRU taille actuelle/max               : %d/%d\n", fc.lruCache.Len(), cfg.CacheSize)
			}
		} else {
			fmt.Println("Cache                                       : Désactivé")
		}
		fmt.Printf("Allocations *big.Int évitées (via pool)   : %d\n", fc.metrics.TempAllocsAvoided.Load())

		fmt.Printf("\nRésultat F(%d) :\n", cfg.N)
		fmt.Printf("  Notation scientifique (~%d chiffres) : %s\n", cfg.Precision, formatScientific(result, cfg.Precision))

		const maxDigitsDisplay = 100
		s := result.String()
		numDigits := len(s)
		fmt.Printf("  Nombre total de chiffres décimaux      : %d\n", numDigits)
		if numDigits <= 2*maxDigitsDisplay {
			fmt.Printf("  Valeur exacte                          : %s\n", s)
		} else {
			fmt.Printf("  Premiers %d chiffres                   : %s...\n", maxDigitsDisplay, s[:maxDigitsDisplay])
			fmt.Printf("  Derniers %d chiffres                   : ...%s\n", maxDigitsDisplay, s[numDigits-maxDigitsDisplay:])
		}

	} else if ctx.Err() == nil {
		log.Println("WARN: Le résultat final est nil, mais aucune erreur de contexte détectée (état inattendu).")
	}

	// --- Écriture du Profil Mémoire (si succès & activé) (inchangé) ---
	if cfg.EnableProfiling && fMem != nil && result != nil {
		log.Println("INFO: Écriture du profil mémoire (heap)...")
		runtime.GC()
		if err := pprof.WriteHeapProfile(fMem); err != nil {
			log.Printf("WARN: Impossible d'écrire le profil mémoire dans %s: %v", fMem.Name(), err)
		} else {
			log.Printf("INFO: Profil mémoire sauvegardé dans %s", fMem.Name())
		}
	}

	log.Println("INFO: Programme terminé.")
}
