// Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark Optimisé
//
// Description :
// Ce programme calcule les nombres de Fibonacci en utilisant la méthode du doublement, une technique efficace qui repose sur la propriété de récurrence des nombres de Fibonacci.
// La méthode est optimisée grâce à l'utilisation de la mémoïsation pour éviter les recalculs redondants et améliorer les performances. Pour implémenter cette mémoïsation,
// le programme utilise un cache LRU (Least Recently Used) thread-safe, ce qui garantit que les valeurs les plus récentes et les plus utilisées sont conservées pour un accès rapide.
//
// Le programme prend également en charge le calcul concurrent grâce à l'utilisation de goroutines et d'un pool de workers. Cela permet de diviser les tâches et de traiter
// des calculs de Fibonacci pour différentes valeurs de manière simultanée, améliorant ainsi l'efficacité globale des calculs. Pour synchroniser et protéger les accès au cache,
// des verrous RWMutex sont utilisés pour assurer une lecture et une écriture sécurisées.
//
// En outre, le programme effectue des benchmarks sur le calcul des nombres de Fibonacci pour différentes valeurs de n, en utilisant un pool de workers configurable.
// Le benchmark inclut un mécanisme de répétition pour obtenir des moyennes de temps d'exécution, et il est géré via un contexte avec timeout afin de garantir qu'aucune
// opération ne dépasse le temps imparti.
//
// Fonctionnalités principales :
// - Calcul des nombres de Fibonacci par la méthode du doublement.
// - Mémoïsation avec un cache LRU pour optimiser les performances.
// - Calculs concurrentiels utilisant des goroutines et un pool de workers.
// - Benchmark des performances avec répétition des calculs et gestion du contexte.
//
// Bibliothèques utilisées :
// - "math/big" pour gérer des entiers de grande taille, nécessaires au calcul des grands nombres de Fibonacci.
// - "github.com/hashicorp/golang-lru/simplelru" pour l'implémentation du cache LRU thread-safe.
// - "sync" et "context" pour la gestion de la concurrence et la synchronisation des goroutines.
//
// Auteur :
// Ce programme a été conçu pour illustrer l'efficacité de l'optimisation par mémoïsation et la puissance de la gestion de la concurrence dans Go.

package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/simplelru"
)

const MAX_FIB_VALUE = 500000001 // Valeur maximale de n qui peut être calculée
var maxCacheSize = 1000         // Nombre maximal d'entrées dans le cache

// Initialiser le cache LRU thread-safe
var lruCache *lru.LRU
var cacheMutex sync.RWMutex

func init() {
	// Initialisation du cache LRU thread-safe
	var err error
	lruCache, err = lru.NewLRU(maxCacheSize, nil)
	if err != nil {
		panic(fmt.Sprintf("Échec de l'initialisation du cache LRU : %v", err))
	}
}

// fibDoubling calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
func fibDoubling(n int) (*big.Int, error) {
	if n < 0 {
		return nil, errors.New("n doit être un entier positif") // Vérifier que n est positif
	}
	// Pour n inférieur à 2, retourner n directement
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}
	if n > MAX_FIB_VALUE {
		return nil, errors.New("n est trop grand pour cette implémentation") // Vérifier que n ne dépasse pas la valeur maximale autorisée
	}
	// Calculer le nombre de Fibonacci en utilisant la méthode de doublage
	result := fibDoublingHelperIterative(n)
	return result, nil
}

// fibDoublingHelperIterative est une fonction itérative qui utilise la méthode de doublage pour calculer les nombres de Fibonacci
func fibDoublingHelperIterative(n int) *big.Int {
	// Vérifier si la valeur est déjà dans le cache LRU
	if val, exists := getFromCache(n); exists {
		return val // Retourner la valeur du cache si elle existe
	}

	// Initialiser les valeurs de base de Fibonacci F(0) = 0 et F(1) = 1
	a := big.NewInt(0) // F(0)
	b := big.NewInt(1) // F(1)
	c := new(big.Int)  // Variable temporaire pour les calculs
	d := new(big.Int)  // Variable temporaire pour les calculs

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

	// Mettre en cache le résultat
	result := new(big.Int).Set(a)
	addToCache(n, result) // Ajouter le résultat au cache

	return result
}

// getFromCache récupère une valeur du cache de manière thread-safe
func getFromCache(n int) (*big.Int, bool) {
	cacheMutex.RLock()         // Verrouiller le cache en lecture
	defer cacheMutex.RUnlock() // Déverrouiller à la fin de la fonction
	if val, ok := lruCache.Get(n); ok {
		return val.(*big.Int), true // Retourner la valeur si elle est trouvée
	}
	return nil, false // Retourner nil si la valeur n'est pas trouvée
}

// addToCache ajoute une valeur au cache de manière thread-safe
func addToCache(n int, value *big.Int) {
	cacheMutex.Lock()         // Verrouiller le cache en écriture
	defer cacheMutex.Unlock() // Déverrouiller à la fin de la fonction
	lruCache.Add(n, value)    // Ajouter la valeur au cache
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
	nValues := []int{100000}
	// Nombre de répétitions pour chaque valeur de n
	repetitions := 10
	// Nombre de workers
	workerCount := 4
	// Contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel() // Annuler le contexte après l'exécution
	// Exécuter le benchmark
	benchmarkFibWithWorkerPool(ctx, nValues, repetitions, workerCount)
}
