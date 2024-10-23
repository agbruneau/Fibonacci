// Ce programme en Go calcule les nombres de Fibonacci en utilisant une combinaison de la méthode de doublage et de parallélisation afin d'optimiser les performances.
//
// Objectif :
// Ce programme est conçu pour illustrer des techniques avancées d'optimisation de calcul à l'aide de la mémoïsation et de la parallélisation. Le calcul des nombres de Fibonacci, en particulier pour des indices élevés, peut être très coûteux en termes de temps et de ressources. En combinant plusieurs méthodes de calcul avancées, ce programme vise à réduire de manière significative le temps de traitement tout en utilisant efficacement la mémoire disponible.
//
// Techniques employées :
// 1. **Méthode de doublage** : Cette technique est une forme d'optimisation mathématique qui permet de calculer les nombres de Fibonacci en exploitant une récurrence basée sur les bits binaires de l'indice. Cela permet de réduire le nombre d'opérations nécessaires et de minimiser le coût du calcul.
//    - En parcourant les bits de l'indice n, la méthode de doublage divise les calculs en opérations successives qui utilisent les relations entre F(2k) et F(2k+1).
// 2. **Mémoïsation avec cache LRU (Least Recently Used)** : Le programme utilise un cache de type LRU pour stocker les valeurs déjà calculées de la suite de Fibonacci. Cela évite les recalculs redondants et améliore les performances globales du programme.
//    - Le cache est géré de manière thread-safe en utilisant des verrous (RWMutex) pour garantir que plusieurs goroutines puissent lire sans conflits tout en protégeant les opérations d'écriture.
// 3. **Parallélisation avec goroutines et pool de workers** : Pour tirer parti des systèmes multi-cœurs modernes, le programme utilise des goroutines et un pool de workers. Cela permet d'effectuer plusieurs calculs de manière simultanée, réduisant ainsi le temps d'exécution global.
//    - Un ensemble de workers exécute des tâches parallèles et les synchronise à l'aide de `WaitGroup`.
//
// Benchmark :
// Le programme inclut un benchmark qui permet d'évaluer les performances des différentes méthodes d'optimisation mises en œuvre. Les benchmarks sont exécutés sur des valeurs prédéfinies de n, avec plusieurs répétitions pour calculer la moyenne du temps d'exécution.
// - Un contexte avec timeout est utilisé pour limiter la durée des tests de performance, garantissant que le programme ne s'exécute pas indéfiniment en cas de problèmes.
//
// Usage :
// - Le programme commence par initialiser le cache LRU et configure un contexte d'exécution avec timeout pour s'assurer que les calculs ne dépassent pas une durée raisonnable.
// - Ensuite, il lance des goroutines pour exécuter le calcul des nombres de Fibonacci en parallèle, puis combine les résultats.
// - Enfin, il affiche les résultats des benchmarks, y compris les temps d'exécution moyens pour chaque valeur calculée.
//
// Conclusion :
// Ce programme est un exemple d'optimisation avancée en Go pour le calcul intensif, utilisant la mémoïsation, la parallélisation et des techniques algorithmiques efficaces. En utilisant le cache LRU et des goroutines, le programme montre comment maximiser les performances tout en minimisant les temps de calcul pour des valeurs élevées de la suite de Fibonacci.

package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

const MAX_FIB_VALUE = 500000001

var maxCacheSize = 1000

// Initialiser le cache LRU thread-safe
var lruCache *lru.Cache
var cacheMutex sync.RWMutex

func init() {
	// Initialisation du cache LRU avec une taille maximale prédéfinie
	var err error
	lruCache, err = lru.New(maxCacheSize)
	if err != nil {
		panic(fmt.Sprintf("Échec de l'initialisation du cache LRU : %v", err))
	}
}

// fibDoubling calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
func fibDoubling(n int) (*big.Int, error) {
	// Retourne une erreur si n est un entier négatif
	if n < 0 {
		return nil, errors.New("n doit être un entier positif")
	}
	// Retourne directement n si n est inférieur à 2 (F(0) = 0, F(1) = 1)
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}
	// Retourne une erreur si n dépasse la valeur maximale autorisée
	if n > MAX_FIB_VALUE {
		return nil, errors.New("n est trop grand pour cette implémentation")
	}
	// Utilise la fonction itérative pour calculer le nombre de Fibonacci
	result := fibDoublingHelperIterative(n)
	return result, nil
}

// fibDoublingHelperIterative calcule les nombres de Fibonacci en utilisant la méthode de doublage
func fibDoublingHelperIterative(n int) *big.Int {
	// Vérifie si la valeur est déjà présente dans le cache LRU
	if val, exists := getFromCache(n); exists {
		return val
	}

	// Initialiser les valeurs de base de Fibonacci : F(0) = 0, F(1) = 1
	a := big.NewInt(0)
	b := big.NewInt(1)
	c := new(big.Int) // Variable temporaire pour les calculs
	d := new(big.Int) // Variable temporaire pour les calculs

	// Parcourir les bits de n du plus significatif au moins significatif
	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		// Calculer c = a * (2 * b - a)
		c.Lsh(b, 1) // c = 2 * b
		c.Sub(c, a) // c = 2 * b - a
		c.Mul(c, a) // c = a * (2 * b - a)

		// Calculer d = a^2 + b^2
		d.Mul(a, a)           // d = a^2
		d.Add(d, b.Mul(b, b)) // d = a^2 + b^2

		// Mettre à jour a et b en fonction du bit actuel de n
		if ((n >> i) & 1) == 0 {
			a.Set(c) // Si le bit est 0, a = c et b = d
			b.Set(d)
		} else {
			a.Set(d) // Si le bit est 1, a = d et b = c + d
			b.Add(c, d)
		}
	}

	// Stocker le résultat dans le cache pour une utilisation future
	result := new(big.Int).Set(a)
	addToCache(n, result)
	return result
}

// getFromCache récupère une valeur du cache de manière thread-safe
func getFromCache(n int) (*big.Int, bool) {
	cacheMutex.RLock()         // Verrouiller le cache en lecture
	defer cacheMutex.RUnlock() // Déverrouiller à la fin de la fonction
	if val, ok := lruCache.Get(n); ok {
		return val.(*big.Int), true // Retourner la valeur si elle est trouvée dans le cache
	}
	return nil, false // Retourner nil si la valeur n'est pas trouvée
}

// addToCache ajoute une valeur au cache de manière thread-safe
func addToCache(n int, value *big.Int) {
	cacheMutex.Lock()         // Verrouiller le cache en écriture
	defer cacheMutex.Unlock() // Déverrouiller à la fin de la fonction
	lruCache.Add(n, value)    // Ajouter la valeur au cache
}

// benchmarkFibWithWorkerPool effectue des tests de performance en utilisant un pool de workers
func benchmarkFibWithWorkerPool(ctx context.Context, nValues []int, repetitions int, workerCount int) {
	// Effacer le cache avant de commencer le benchmark
	clearMemoization()

	// Canal pour les travaux
	jobs := make(chan int)
	// Canal pour les résultats
	results := make(chan string)
	var wg sync.WaitGroup

	// Lancer les workers
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for n := range jobs {
				select {
				case <-ctx.Done():
					// Arrêter si le contexte est annulé
					return
				default:
					totalExecTime := time.Duration(0)
					// Effectuer les répétitions pour calculer la moyenne du temps d'exécution
					for i := 0; i < repetitions; i++ {
						start := time.Now() // Début du chronométrage
						_, err := fibDoubling(n)
						if err != nil {
							printError(n, err) // Afficher l'erreur si elle se produit
							continue
						}
						totalExecTime += time.Since(start) // Ajouter la durée de l'exécution
					}
					// Calculer le temps d'exécution moyen
					avgExecTime := totalExecTime / time.Duration(repetitions)
					// Envoyer le résultat au canal des résultats
					result := fmt.Sprintf("Worker %d: fibDoubling(%d) moyenne sur %d exécutions: %s", workerID, n, repetitions, avgExecTime)
					select {
					case results <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}(w)
	}

	// Lancer une goroutine pour collecter les résultats
	go func() {
		wg.Wait()      // Attendre que tous les workers aient terminé
		close(results) // Fermer le canal des résultats
	}()

	// Envoyer les travaux
	go func() {
		for _, n := range nValues {
			select {
			case jobs <- n: // Envoyer la valeur de n au canal des travaux
			case <-ctx.Done():
				close(jobs) // Fermer le canal des travaux si le contexte est annulé
				return
			}
		}
		close(jobs) // Fermer le canal des travaux une fois tous les travaux envoyés
	}()

	// Afficher les résultats
	for res := range results {
		fmt.Println(res) // Afficher chaque résultat
	}
}

// Fonction principale pour exécuter les tests de performance
func main() {
	// Liste des valeurs de n pour le benchmark
	nValues := []int{100000000}
	// Nombre de répétitions pour chaque valeur de n
	repetitions := 10
	// Nombre de workers
	workerCount := 16
	// Contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel() // Annuler le contexte après l'exécution
	// Exécuter le benchmark
	benchmarkFibWithWorkerPool(ctx, nValues, repetitions, workerCount)
}

// clearMemoization efface le cache LRU de manière thread-safe
func clearMemoization() {
	cacheMutex.Lock()         // Verrouiller le cache en écriture
	defer cacheMutex.Unlock() // Déverrouiller à la fin de la fonction
	lruCache.Purge()          // Vider le cache
}

// printError affiche un message d'erreur dans un format cohérent
func printError(n int, err error) {
	fmt.Printf("fibDoubling(%d): %s\n", n, err) // Formater et afficher l'erreur
}
