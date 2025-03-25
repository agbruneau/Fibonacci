// =============================================================================
// Programme : Calcul ultra-optimisé de Fibonacci(n) en Go
// Auteur    : [Votre nom]
// Date      : [Date]
// Version   : 4.0
//
// Description :
// Cette version intègre des optimisations avancées pour le calcul de Fibonacci :
// - Algorithme de doublement matriciel optimisé
// - Mémoire pré-allouée et recyclée
// - Parallélisation fine des opérations
// - Optimisation des opérations big.Int
// =============================================================================

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Configuration des paramètres
type Config struct {
	N               int           // Fibonacci(N)
	Timeout         time.Duration // Durée max d'exécution
	Precision       int           // Chiffres significatifs
	Workers         int           // Nombre de workers parallèles
	EnableCache     bool          // Cache intermédiaire
	EnableProfiling bool          // Profiling CPU/mémoire
}

func DefaultConfig() Config {
	return Config{
		N:               200000000,
		Timeout:         5 * time.Minute,
		Precision:       6,
		Workers:         runtime.NumCPU(),
		EnableCache:     true,
		EnableProfiling: false,
	}
}

// Structure pour les métriques de performance
type Metrics struct {
	StartTime   time.Time
	EndTime     time.Time
	OpsCount    int64
	CacheHits   int64
	MemSaved    int64
	ParallelOps int64
}

func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

func (m *Metrics) AddOps(n int64) {
	atomic.AddInt64(&m.OpsCount, n)
}

func (m *Metrics) AddCacheHit() {
	atomic.AddInt64(&m.CacheHits, 1)
}

func (m *Metrics) AddMemSaved(n int64) {
	atomic.AddInt64(&m.MemSaved, n)
}

func (m *Metrics) AddParallelOp() {
	atomic.AddInt64(&m.ParallelOps, 1)
}

// Pool de matrices pour Fibonacci
type FibMatrix struct {
	a, b, c, d *big.Int
}

type FibCalculator struct {
	cache      map[int]*big.Int
	mu         sync.RWMutex
	matrixPool sync.Pool
	config     Config
	metrics    *Metrics
}

func NewFibCalculator(cfg Config) *FibCalculator {
	fc := &FibCalculator{
		cache:   make(map[int]*big.Int),
		config:  cfg,
		metrics: NewMetrics(),
	}

	fc.matrixPool = sync.Pool{
		New: func() interface{} {
			return &FibMatrix{
				a: new(big.Int),
				b: new(big.Int),
				c: new(big.Int),
				d: new(big.Int),
			}
		},
	}

	// Pré-allocation pour les cas de base
	if cfg.EnableCache {
		fc.cache[0] = big.NewInt(0)
		fc.cache[1] = big.NewInt(1)
		fc.cache[2] = big.NewInt(1)
	}

	return fc
}

// multiplyMatrices multiplie deux matrices 2x2 de façon optimisée
func (fc *FibCalculator) multiplyMatrices(m1, m2 *FibMatrix, result *FibMatrix) {
	// Canal pour les résultats partiels
	resChan := make(chan struct{}, 4)

	// Calculs parallélisés
	go func() {
		result.a.Mul(m1.a, m2.a)
		result.a.Add(result.a, new(big.Int).Mul(m1.b, m2.c))
		resChan <- struct{}{}
		fc.metrics.AddParallelOp()
	}()

	go func() {
		result.b.Mul(m1.a, m2.b)
		result.b.Add(result.b, new(big.Int).Mul(m1.b, m2.d))
		resChan <- struct{}{}
		fc.metrics.AddParallelOp()
	}()

	go func() {
		result.c.Mul(m1.c, m2.a)
		result.c.Add(result.c, new(big.Int).Mul(m1.d, m2.c))
		resChan <- struct{}{}
		fc.metrics.AddParallelOp()
	}()

	go func() {
		result.d.Mul(m1.c, m2.b)
		result.d.Add(result.d, new(big.Int).Mul(m1.d, m2.d))
		resChan <- struct{}{}
		fc.metrics.AddParallelOp()
	}()

	// Attendre la fin des calculs
	for i := 0; i < 4; i++ {
		<-resChan
	}
}

// fastDoubling calcule Fibonacci(n) avec l'algorithme de doublement matriciel
func (fc *FibCalculator) fastDoubling(n int) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}

	matrix := fc.matrixPool.Get().(*FibMatrix)
	defer fc.matrixPool.Put(matrix)

	// Initialisation de la matrice
	matrix.a.SetInt64(1)
	matrix.b.SetInt64(1)
	matrix.c.SetInt64(1)
	matrix.d.SetInt64(0)

	result := fc.matrixPool.Get().(*FibMatrix)
	defer fc.matrixPool.Put(result)

	result.a.SetInt64(1)
	result.b.SetInt64(0)
	result.c.SetInt64(0)
	result.d.SetInt64(1)

	temp := fc.matrixPool.Get().(*FibMatrix)
	defer fc.matrixPool.Put(temp)

	for k := uint(0); k < uint(unsafe.Sizeof(n)*8); k++ {
		if n&(1<<k) != 0 {
			fc.multiplyMatrices(result, matrix, temp)
			result, temp = temp, result
			fc.metrics.AddOps(1)
		}

		if 1<<k > uint(n) {
			break
		}

		fc.multiplyMatrices(matrix, matrix, temp)
		matrix, temp = temp, matrix
		fc.metrics.AddOps(1)
	}

	return result.b
}

// Calculate gère le cache et appelle l'algorithme optimisé
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être positif")
	}

	// Vérification du cache
	if fc.config.EnableCache {
		if val, ok := fc.getFromCache(n); ok {
			fc.metrics.AddCacheHit()
			return val, nil
		}
	}

	// Cas spéciaux
	if n <= 2 {
		return big.NewInt(int64(n)), nil
	}

	// Calcul optimisé
	result := fc.fastDoubling(n)

	// Mise en cache
	if fc.config.EnableCache {
		fc.storeInCache(n, result)
	}

	return result, nil
}

func (fc *FibCalculator) getFromCache(n int) (*big.Int, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	val, ok := fc.cache[n]
	return val, ok
}

func (fc *FibCalculator) storeInCache(n int, val *big.Int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.cache[n] = val
}

// formatScientific formate en notation scientifique standard
func formatScientific(num *big.Int, precision int) string {
	s := num.String()
	if len(s) <= 1 {
		return s
	}

	if precision < 1 {
		precision = 1
	}
	if len(s)-1 < precision {
		precision = len(s) - 1
	}

	return fmt.Sprintf("%s.%se%d", s[:1], s[1:1+precision], len(s)-1)
}

func main() {
	cfg := DefaultConfig()
	runtime.GOMAXPROCS(cfg.Workers)

	// Démarrer le profiling si activé
	if cfg.EnableProfiling {
		// Ici vous pourriez ajouter du code de profiling avec pprof
		log.Println("Profiling activé - utilisez pprof pour analyser")
	}

	fc := NewFibCalculator(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Canal pour le résultat
	resultChan := make(chan *big.Int, 1)
	errChan := make(chan error, 1)

	go func() {
		start := time.Now()
		res, err := fc.Calculate(cfg.N)
		if err != nil {
			errChan <- err
			return
		}
		fc.metrics.EndTime = time.Now()
		log.Printf("Calcul terminé en %v", fc.metrics.EndTime.Sub(start))
		resultChan <- res
	}()

	var result *big.Int
	select {
	case <-ctx.Done():
		log.Fatalf("Timeout après %v", cfg.Timeout)
	case err := <-errChan:
		log.Fatalf("Erreur: %v", err)
	case result = <-resultChan:
		// Succès
	}

	// Affichage des résultats
	fmt.Printf("\n=== Résultats Fibonacci(%d) ===\n", cfg.N)
	fmt.Printf("Temps total: %v\n", fc.metrics.EndTime.Sub(fc.metrics.StartTime))
	fmt.Printf("Opérations: %d\n", fc.metrics.OpsCount)
	fmt.Printf("Cache hits: %d\n", fc.metrics.CacheHits)
	fmt.Printf("Opérations parallèles: %d\n", fc.metrics.ParallelOps)
	fmt.Printf("Mémoire économisée: %d allocations\n", fc.metrics.MemSaved)

	fmt.Printf("\nRésultat: %s\n", formatScientific(result, cfg.Precision))

	if cfg.EnableProfiling {
		// Ici vous pourriez sauvegarder les données de profiling
		log.Println("Données de profiling disponibles")
	}
}
