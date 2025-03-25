// =============================================================================
// Programme : Calcul optimisé de Fibonacci(n) - Version scientifique simple
// Auteur    : [Votre nom]
// Date      : [Date]
// Version   : 3.1
//
// Description :
// Ce programme implémente le calcul du n-ième nombre de Fibonacci en utilisant
// l'algorithme du doublement parallélisé avec gestion de cache et optimisation mémoire.
// L'affichage utilise la notation scientifique standard sans caractères superscript.
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
)

// Configuration centralise les paramètres configurables
type Configuration struct {
	M               int           // Calcul de Fibonacci(M)
	Timeout         time.Duration // Durée maximale d'exécution
	Precision       int           // Nombre de chiffres significatifs à afficher
	EnableCache     bool          // Active le cache des résultats intermédiaires
	EnableBenchmark bool          // Active les benchmarks comparatifs
}

// DefaultConfig retourne une configuration par défaut
func DefaultConfig() Configuration {
	return Configuration{
		M:               200000000,
		Timeout:         5 * time.Minute,
		Precision:       6,
		EnableCache:     true,
		EnableBenchmark: false,
	}
}

// Metrics conserve les métriques de performance
type Metrics struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalCalculations int64
	CacheHits         int64
	AllocationsSaved  int64
}

// NewMetrics initialise les métriques
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// AddCalculations incrémente le compteur atomiquement
func (m *Metrics) AddCalculations(n int64) {
	atomic.AddInt64(&m.TotalCalculations, n)
}

// AddCacheHit incrémente le compteur de cache hits
func (m *Metrics) AddCacheHit() {
	atomic.AddInt64(&m.CacheHits, 1)
}

// AddAllocationSaved incrémente le compteur d'allocations économisées
func (m *Metrics) AddAllocationSaved() {
	atomic.AddInt64(&m.AllocationsSaved, 1)
}

// FibCalculator encapsule le calcul de Fibonacci avec cache
type FibCalculator struct {
	cache map[int]*big.Int
	mu    sync.RWMutex
	pool  sync.Pool // Pool de big.Int pour réduire les allocations
}

// NewFibCalculator crée un nouveau calculateur avec cache
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		cache: make(map[int]*big.Int),
		pool: sync.Pool{
			New: func() interface{} {
				return new(big.Int)
			},
		},
	}
}

// getFromCache tente de récupérer un résultat depuis le cache
func (fc *FibCalculator) getFromCache(n int) (*big.Int, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	val, ok := fc.cache[n]
	if ok {
		return val, true
	}
	return nil, false
}

// storeInCache stocke un résultat dans le cache
func (fc *FibCalculator) storeInCache(n int, val *big.Int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.cache[n] = val
}

// Calculate retourne F(n) pour n ≥ 0 en utilisant le cache si activé
func (fc *FibCalculator) Calculate(n int, enableCache bool, metrics *Metrics) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être non négatif")
	}

	// Vérification du cache
	if enableCache {
		if val, ok := fc.getFromCache(n); ok {
			metrics.AddCacheHit()
			return val, nil
		}
	}

	// Cas de base
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}

	// Calcul avec doublement parallèle
	result, err := fc.fibDoublingParallel(n, metrics)
	if err != nil {
		return nil, err
	}

	// Mise en cache
	if enableCache {
		fc.storeInCache(n, result)
	}

	return result, nil
}

// fibDoublingParallel implémente l'algorithme de doublement avec parallélisation
func (fc *FibCalculator) fibDoublingParallel(n int, metrics *Metrics) (*big.Int, error) {
	// Récupération de big.Int depuis le pool
	a := fc.pool.Get().(*big.Int).SetInt64(0)
	b := fc.pool.Get().(*big.Int).SetInt64(1)
	defer func() {
		// Remise dans le pool
		fc.pool.Put(a)
		fc.pool.Put(b)
	}()

	// Trouver le bit le plus significatif
	highest := 0
	for i := 31; i >= 0; i-- {
		if n&(1<<i) != 0 {
			highest = i
			break
		}
	}

	// Variables temporaires réutilisables
	temp := fc.pool.Get().(*big.Int)
	twoB := fc.pool.Get().(*big.Int)
	defer func() {
		fc.pool.Put(temp)
		fc.pool.Put(twoB)
	}()

	for i := highest; i >= 0; i-- {
		// Calcul des termes intermédiaires
		twoB.Lsh(b, 1)
		temp.Sub(twoB, a)

		// Canaux pour les résultats parallèles
		cChan := make(chan *big.Int, 1)
		t1Chan := make(chan *big.Int, 1)
		t2Chan := make(chan *big.Int, 1)

		// Goroutines pour les calculs parallèles
		go func(a, temp *big.Int) {
			res := fc.pool.Get().(*big.Int).Mul(a, temp)
			cChan <- res
		}(new(big.Int).Set(a), new(big.Int).Set(temp))

		go func(a *big.Int) {
			res := fc.pool.Get().(*big.Int).Mul(a, a)
			t1Chan <- res
		}(new(big.Int).Set(a))

		go func(b *big.Int) {
			res := fc.pool.Get().(*big.Int).Mul(b, b)
			t2Chan <- res
		}(new(big.Int).Set(b))

		// Récupération des résultats
		c := <-cChan
		t1 := <-t1Chan
		t2 := <-t2Chan

		// Calcul final
		d := fc.pool.Get().(*big.Int).Add(t1, t2)
		defer fc.pool.Put(d)

		if n&(1<<uint(i)) != 0 {
			a.Set(d)
			b.Add(c, d)
		} else {
			a.Set(c)
			b.Set(d)
		}

		// Remise dans le pool
		fc.pool.Put(c)
		fc.pool.Put(t1)
		fc.pool.Put(t2)

		metrics.AddCalculations(1)
	}

	return new(big.Int).Set(a), nil
}

// formatBigIntScientific formate un big.Int en notation scientifique standard
func formatBigIntScientific(n *big.Int, precision int) string {
	s := n.String()
	if len(s) <= 1 {
		return s
	}

	// Ajustement de la précision
	if precision < 1 {
		precision = 1
	}
	if len(s)-1 < precision {
		precision = len(s) - 1
	}

	significand := s[:1]
	if precision > 0 {
		significand += "." + s[1:1+precision]
	}
	exponent := len(s) - 1
	return fmt.Sprintf("%se%d", significand, exponent)
}

// runBenchmark exécute des benchmarks comparatifs
func runBenchmark(fc *FibCalculator, n int, metrics *Metrics) {
	fmt.Println("\nBenchmark comparatif:")

	// Test avec cache
	start := time.Now()
	fc.Calculate(n, true, metrics)
	withCache := time.Since(start)

	// Test sans cache
	start = time.Now()
	fc.Calculate(n, false, metrics)
	withoutCache := time.Since(start)

	fmt.Printf("Avec cache: %v\n", withCache)
	fmt.Printf("Sans cache: %v\n", withoutCache)
	fmt.Printf("Gain: %.2f%%\n",
		float64(withoutCache.Nanoseconds()-withCache.Nanoseconds())/float64(withoutCache.Nanoseconds())*100)
}

func main() {
	// Configuration
	runtime.GOMAXPROCS(runtime.NumCPU())
	config := DefaultConfig()
	metrics := NewMetrics()
	fc := NewFibCalculator()

	// Contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Calcul principal
	resultChan := make(chan *big.Int, 1)
	errorChan := make(chan error, 1)

	go func() {
		fib, err := fc.Calculate(config.M, config.EnableCache, metrics)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- fib
	}()

	var fibResult *big.Int
	select {
	case <-ctx.Done():
		log.Fatalf("Délai d'exécution dépassé : %v", ctx.Err())
	case err := <-errorChan:
		log.Fatalf("Erreur lors du calcul : %v", err)
	case fibResult = <-resultChan:
		// Calcul terminé avec succès
	}

	// Benchmark si activé
	if config.EnableBenchmark {
		runBenchmark(fc, config.M/1000, metrics) // Test sur une valeur plus petite
	}

	// Finalisation des métriques
	metrics.EndTime = time.Now()
	duration := metrics.EndTime.Sub(metrics.StartTime)

	// Affichage des résultats
	fmt.Printf("\nConfiguration :\n")
	fmt.Printf("  Valeur de M             : %d\n", config.M)
	fmt.Printf("  Timeout                 : %v\n", config.Timeout)
	fmt.Printf("  Précision affichage     : %d chiffres\n", config.Precision)
	fmt.Printf("  Cache activé            : %v\n", config.EnableCache)
	fmt.Printf("  Nombre de cœurs utilisés: %d\n", runtime.NumCPU())

	fmt.Printf("\nPerformance :\n")
	fmt.Printf("  Temps total d'exécution : %v\n", duration)
	fmt.Printf("  Nombre de calculs       : %d\n", metrics.TotalCalculations)
	fmt.Printf("  Cache hits              : %d\n", metrics.CacheHits)
	fmt.Printf("  Allocations économisées : %d\n", metrics.AllocationsSaved)

	// Affichage du résultat
	formattedResult := formatBigIntScientific(fibResult, config.Precision)
	fmt.Printf("\nRésultat :\n")
	fmt.Printf("  Fibonacci(%d) : %s\n", config.M, formattedResult)
}
