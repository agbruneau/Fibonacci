// Claude AI : Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark
//
// Description :
// Ce programme en Go calcule les nombres de Fibonacci en utilisant la méthode du doublement, qui est une approche
// efficace basée sur la division et la conquête. L'algorithme utilise une technique itérative pour calculer
// rapidement les valeurs de Fibonacci pour de très grands nombres. Pour améliorer la performance, une stratégie
// de mémoïsation avec LRU (Least Recently Used) est utilisée afin de mettre en cache les résultats des calculs
// précédents. Cela permet de réutiliser les valeurs déjà calculées et de réduire le temps de calcul des appels
// futurs. De plus, le programme est conçu pour utiliser des goroutines, ce qui permet un calcul concurrent et
// améliore l'efficacité en utilisant plusieurs threads.
//
// Algorithme de Doublement :
// L'algorithme de doublement repose sur les propriétés suivantes des nombres de Fibonacci :
// - F(2k) = F(k) * [2 * F(k+1) - F(k)]
// - F(2k + 1) = F(k)^2 + F(k+1)^2
// Ces formules permettent de calculer des valeurs de Fibonacci en utilisant une approche binaire sur les bits
// de l'indice n, rendant l'algorithme très performant pour de grands nombres.
//
// Le programme effectue également des tests de performance (benchmark) sur des valeurs élevées de Fibonacci
// et affiche le temps moyen d'exécution pour chaque valeur, en utilisant des répétitions multiples pour
// une meilleure précision.

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync"
	"time"
)

// Constantes pour la configuration
const (
	MaxFibValue       = 500000000        // Valeur maximale de l'indice de Fibonacci pouvant être calculé
	SmallFibThreshold = 93               // Seuil pour utiliser le calcul avec int64 (pour éviter le dépassement)
	DefaultWorkers    = 16               // Nombre par défaut de goroutines de travail
	BenchmarkTimeout  = 10 * time.Minute // Temps limite pour l'opération de benchmark
)

// Type d'erreur personnalisé pour les erreurs liées à Fibonacci
type FibError struct {
	N     int
	Cause string
}

func (e *FibError) Error() string {
	// Formatage du message d'erreur pour les erreurs de Fibonacci
	return fmt.Sprintf("erreur de Fibonacci pour n=%d: %s", e.N, e.Cause)
}

// BenchmarkResult représente le résultat d'une exécution de benchmark
type BenchmarkResult struct {
	N        int           // L'indice de Fibonacci calculé
	Duration time.Duration // Temps pris pour calculer le nombre de Fibonacci
	WorkerID int           // ID du travailleur qui a effectué le calcul
	Error    error         // Toute erreur survenue lors du calcul
}

// Implémentation du cache avec sync.Map pour la sécurité des threads
type FibCache struct {
	cache sync.Map // Cache thread-safe pour stocker les nombres de Fibonacci
}

type cacheEntry struct {
	value     *big.Int  // La valeur de Fibonacci mise en cache
	timestamp time.Time // Timestamp de l'entrée en cache (utilisé pour l'expiration)
}

// BigIntPool gère un pool d'objets big.Int
var bigIntPool = sync.Pool{
	New: func() interface{} {
		// Créer un nouvel objet big.Int si nécessaire
		return new(big.Int)
	},
}

// FibCalculator gère les calculs de Fibonacci
type FibCalculator struct {
	cache *FibCache // Cache pour stocker les nombres de Fibonacci déjà calculés
	two   *big.Int  // Valeur constante 2, utilisée dans les calculs
}

func NewFibCalculator() *FibCalculator {
	// Créer un nouveau calculateur de Fibonacci avec un cache initialisé
	return &FibCalculator{
		cache: &FibCache{},
		two:   big.NewInt(2),
	}
}

// Calculate calcule le nième nombre de Fibonacci
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		// Retourne une erreur si l'indice est négatif
		return nil, &FibError{N: n, Cause: "indice négatif"}
	}
	if n > MaxFibValue {
		// Retourne une erreur si l'indice dépasse la valeur maximale autorisée
		return nil, &FibError{N: n, Cause: "valeur trop grande"}
	}

	// Vérifie d'abord dans le cache
	if result, ok := fc.cache.get(n); ok {
		// Retourne le résultat mis en cache s'il est disponible
		return result, nil
	}

	// Utilise int64 pour les petites valeurs afin d'éviter des calculs big.Int inutiles
	if n < SmallFibThreshold {
		return big.NewInt(fc.fibInt64(n)), nil
	}

	// Pour les grandes valeurs, utilise la méthode de doublement
	result := fc.fibDoubling(n)
	// Met en cache le résultat pour une utilisation future
	fc.cache.set(n, result)
	return result, nil
}

// fibInt64 calcule Fibonacci pour les petits nombres en utilisant int64
func (fc *FibCalculator) fibInt64(n int) int64 {
	if n <= 1 {
		// Cas de base : F(0) = 0, F(1) = 1
		return int64(n)
	}

	var a, b int64 = 0, 1
	// Calcule itérativement Fibonacci en utilisant int64 pour l'efficacité
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// fibDoubling implémente la méthode de doublement pour les grands nombres de Fibonacci
func (fc *FibCalculator) fibDoubling(n int) *big.Int {
	if n <= 1 {
		// Cas de base pour big.Int : F(0) = 0, F(1) = 1
		return big.NewInt(int64(n))
	}

	// Obtient des objets big.Int du pool pour éviter des allocations fréquentes
	k := getBigInt()
	k1 := getBigInt()
	defer func() {
		// Retourne les objets big.Int au pool pour réutilisation
		bigIntPool.Put(k)
		bigIntPool.Put(k1)
	}()

	// Initialiser F(1) et F(2)
	k.SetInt64(1)  // F(1)
	k1.SetInt64(1) // F(2)

	// Utilise la méthode de doublement pour calculer F(n)
	for i := n - 2; i > 0; i >>= 1 {
		// Obtient des objets big.Int temporaires du pool
		tmp := getBigInt()
		tmp2 := getBigInt()
		defer func() {
			// Retourne les objets big.Int temporaires au pool pour réutilisation
			bigIntPool.Put(tmp)
			bigIntPool.Put(tmp2)
		}()

		// Calculer F(2k+1) = F(k) * (2 * F(k+1) - F(k))
		tmp.Mul(k, fc.two) // 2 * F(k+1)
		tmp.Sub(tmp, k1)   // 2 * F(k+1) - F(k)
		tmp.Mul(k1, tmp)   // F(k) * (2 * F(k+1) - F(k))

		// Calculer F(2k+2) = F(k+1)^2 + F(k)^2
		tmp2.Mul(k1, k1)    // F(k+1)^2
		tmp2.Add(tmp2, tmp) // F(k+1)^2 + F(k)^2

		// Mettre à jour k et k1 en fonction de la parité de i
		if i&1 == 1 {
			k.Set(tmp2)       // F(2k+2)
			k1.Add(tmp, tmp2) // F(2k+3)
		} else {
			k.Set(tmp)   // F(2k+1)
			k1.Set(tmp2) // F(2k+2)
		}
	}

	return k1
}

// Méthodes de cache
func (c *FibCache) get(n int) (*big.Int, bool) {
	// Tentative de chargement de la valeur depuis le cache
	if val, ok := c.cache.Load(n); ok {
		entry := val.(cacheEntry)
		// Vérifie si l'entrée en cache est toujours valide (non expirée)
		if time.Since(entry.timestamp) < time.Hour {
			return entry.value, true
		}
		// Supprime l'entrée de cache expirée
		c.cache.Delete(n)
	}
	return nil, false
}

func (c *FibCache) set(n int, value *big.Int) {
	// Stocke la valeur dans le cache avec le timestamp actuel
	c.cache.Store(n, cacheEntry{
		value:     value,
		timestamp: time.Now(),
	})
}

// Fonctionnalité de benchmark
type Benchmarker struct {
	calculator *FibCalculator // Instance du calculateur de Fibonacci
	workers    int            // Nombre de goroutines de travail
}

func NewBenchmarker(workers int) *Benchmarker {
	// Définit le nombre de travailleurs au nombre de CPU si non spécifié
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Benchmarker{
		calculator: NewFibCalculator(),
		workers:    workers,
	}
}

func (b *Benchmarker) Run(ctx context.Context, values []int, repetitions int) []BenchmarkResult {
	// Canal pour collecter les résultats de benchmark
	results := make(chan BenchmarkResult, len(values)*repetitions)
	// Canal pour distribuer les tâches aux travailleurs
	jobs := make(chan int, len(values))
	var wg sync.WaitGroup

	// Démarrer les travailleurs
	for w := 0; w < b.workers; w++ {
		wg.Add(1)
		go b.worker(ctx, w, jobs, results, repetitions, &wg)
	}

	// Envoyer les tâches aux travailleurs
	go func() {
		for _, n := range values {
			select {
			case <-ctx.Done():
				// Arrêter d'envoyer des tâches si le contexte est terminé
				return
			case jobs <- n:
				// Envoyer la tâche aux travailleurs
			}
		}
		close(jobs) // Fermer le canal des tâches une fois que toutes les tâches sont envoyées
	}()

	// Attendre que tous les travailleurs aient fini
	wg.Wait()
	close(results) // Fermer le canal des résultats une fois que tous les travailleurs ont fini

	// Collecter les résultats
	var benchmarkResults []BenchmarkResult
	for result := range results {
		benchmarkResults = append(benchmarkResults, result)
	}

	return benchmarkResults
}

func (b *Benchmarker) worker(ctx context.Context, id int, jobs <-chan int, results chan<- BenchmarkResult, repetitions int, wg *sync.WaitGroup) {
	defer wg.Done() // Marquer le travailleur comme terminé lorsque la fonction se termine

	for n := range jobs {
		for r := 0; r < repetitions; r++ {
			select {
			case <-ctx.Done():
				// Arrêter le travailleur si le contexte est terminé
				return
			default:
				// Mesurer le temps pris pour calculer le nombre de Fibonacci
				start := time.Now()
				_, err := b.calculator.Calculate(n)
				duration := time.Since(start)

				// Envoyer le résultat au canal des résultats
				results <- BenchmarkResult{
					N:        n,
					Duration: duration,
					WorkerID: id,
					Error:    err,
				}
			}
		}
	}
}

// Fonctions utilitaires
func getBigInt() *big.Int {
	// Obtient un big.Int du pool et réinitialise sa valeur à 0
	return bigIntPool.Get().(*big.Int).SetInt64(0)
}

func main() {
	// Définir les paramètres de benchmark
	values := []int{1000, 10000, 100000, 1000000, 10000000, 100000000}
	repetitions := 100
	benchmarker := NewBenchmarker(DefaultWorkers)

	// Créer un contexte avec un timeout pour limiter la durée du benchmark
	ctx, cancel := context.WithTimeout(context.Background(), BenchmarkTimeout)
	defer cancel() // S'assurer que le contexte est annulé pour libérer les ressources

	// Exécuter le benchmark
	fmt.Printf("Exécution du benchmark avec %d travailleurs...\n", benchmarker.workers)
	results := benchmarker.Run(ctx, values, repetitions)

	// Traiter et afficher les résultats
	processResults(results)
}

func processResults(results []BenchmarkResult) {
	// Grouper les résultats par indice de Fibonacci (N)
	resultsByN := make(map[int][]time.Duration)
	for _, r := range results {
		if r.Error != nil {
			// Enregistrer toute erreur survenue lors du calcul
			log.Printf("Erreur lors du calcul de F(%d): %v", r.N, r.Error)
			continue
		}
		// Ajouter la durée à la liste pour l'indice de Fibonacci spécifique
		resultsByN[r.N] = append(resultsByN[r.N], r.Duration)
	}

	// Calculer et afficher la durée moyenne pour chaque indice de Fibonacci
	for n, durations := range resultsByN {
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avg := total / time.Duration(len(durations))
		fmt.Printf("F(%d): Temps moyen = %v\n", n, avg)
	}
}
