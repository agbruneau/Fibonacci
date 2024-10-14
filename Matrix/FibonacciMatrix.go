// Programme de calcul des nombres de Fibonacci en utilisant la méthode de la matrice de puissance avec mémoïsation et benchmark concurrentiel
//
// Description :
// Ce programme implémente un algorithme pour calculer les nombres de Fibonacci en utilisant la méthode de la matrice de puissance. Cette méthode est particulièrement
// efficace car elle réduit la complexité temporelle à O(log n). Le programme est optimisé avec une mémoïsation basée sur un cache LRU (Least Recently Used) thread-safe,
// ce qui permet de réutiliser les valeurs précédemment calculées et d'améliorer les performances. Le cache est protégé par des verrous RWMutex pour assurer une
// utilisation sécurisée dans un environnement concurrentiel.
//
// Le programme comprend également une fonctionnalité de benchmark qui évalue les performances du calcul des nombres de Fibonacci pour différentes valeurs de n. Pour
// cela, il utilise un pool de workers qui effectuent les calculs de manière parallèle, permettant d'exploiter efficacement les ressources des systèmes multi-cœurs.
// Le benchmark est géré à l'aide d'un contexte (context.Context) qui définit un délai d'expiration pour éviter les exécutions trop longues.
//
// Fonctionnalités principales :
// - Calcul des nombres de Fibonacci par la méthode de la matrice de puissance.
// - Mémoïsation avec un cache LRU pour optimiser les performances et éviter les recalculs.
// - Gestion du cache avec des verrous pour assurer la sécurité dans un contexte concurrentiel.
// - Benchmark concurrentiel utilisant des goroutines et un pool de workers.
// - Gestion du contexte pour limiter le temps d'exécution des benchmarks.
//
// Bibliothèques utilisées :
// - "math/big" pour gérer des entiers de grande taille, nécessaires pour les grands nombres de Fibonacci.
// - "github.com/hashicorp/golang-lru" pour implémenter un cache LRU thread-safe.
// - "sync" et "context" pour la gestion de la concurrence, la synchronisation et la gestion des goroutines.
//
// Auteur :
// Ce programme est conçu pour illustrer l'utilisation de techniques avancées telles que la mémoïsation, la programmation concurrentielle et les méthodes de calcul
// matriciel pour résoudre des problèmes complexes de manière efficace.

package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

const MAX_FIB_VALUE = 500000001 // Valeur maximale de n qui peut être calculée
var maxCacheSize = 1000         // Nombre maximal d'entrées dans le cache

// Initialiser le cache LRU thread-safe
var lruCache *lru.Cache
var cacheMutex sync.RWMutex

func init() {
	// Initialisation du cache LRU thread-safe
	var err error
	lruCache, err = lru.New(maxCacheSize)
	if err != nil {
		panic(fmt.Sprintf("Échec de l'initialisation du cache LRU : %v", err))
	}
}

// fibMatrixPower calcule le nième nombre de Fibonacci en utilisant la méthode de la matrice de puissance
func fibMatrixPower(n int) (*big.Int, error) {
	if n < 0 {
		return nil, errors.New("n doit être un entier positif") // Vérifier que n est positif
	}
	if n < 2 {
		return big.NewInt(int64(n)), nil // Retourner n directement pour les valeurs 0 et 1
	}
	if n > MAX_FIB_VALUE {
		return nil, errors.New("n est trop grand pour cette implémentation") // Vérifier que n ne dépasse pas la valeur maximale autorisée
	}

	// Vérifier si la valeur est déjà dans le cache LRU
	if val, exists := getFromCache(n); exists {
		return val, nil // Retourner la valeur du cache si elle existe
	}

	// Initialiser les matrices de base pour le calcul de Fibonacci
	F := [2][2]*big.Int{
		{big.NewInt(1), big.NewInt(1)}, // F(1,1) et F(1,0)
		{big.NewInt(1), big.NewInt(0)}, // F(0,1) et F(0,0)
	}
	result := matrixPower(F, n-1) // Calculer la puissance de la matrice F^(n-1)

	// La valeur de Fibonacci est dans la case [0][0] de la matrice résultante
	fibValue := new(big.Int).Set(result[0][0])

	// Mettre en cache le résultat dans le cache LRU
	addToCache(n, fibValue) // Ajouter le résultat au cache

	return fibValue, nil
}

// matrixPower calcule la puissance d'une matrice à l'exposant n
func matrixPower(matrix [2][2]*big.Int, n int) [2][2]*big.Int {
	// Initialiser la matrice identité
	result := [2][2]*big.Int{
		{big.NewInt(1), big.NewInt(0)}, // Matrice identité : F(1,0)
		{big.NewInt(0), big.NewInt(1)}, // Matrice identité : F(0,1)
	}

	base := matrix // Définir la matrice de base
	for n > 0 {
		if n%2 == 1 {
			result = matrixMultiply(result, base) // Multiplier le résultat par la base si n est impair
		}
		base = matrixMultiply(base, base) // Élever la base au carré
		n /= 2                            // Diviser n par 2
	}
	return result
}

// matrixMultiply multiplie deux matrices 2x2 en réutilisant les big.Int pour éviter les allocations
func matrixMultiply(a, b [2][2]*big.Int) [2][2]*big.Int {
	// Préallouer les big.Int pour le résultat
	result := [2][2]*big.Int{
		{new(big.Int), new(big.Int)},
		{new(big.Int), new(big.Int)},
	}

	// Calculer les éléments de la matrice résultante
	// result[0][0] = a[0][0] * b[0][0] + a[0][1] * b[1][0]
	mul00 := new(big.Int).Mul(a[0][0], b[0][0])
	mul01 := new(big.Int).Mul(a[0][1], b[1][0])
	result[0][0].Add(mul00, mul01)

	// result[0][1] = a[0][0] * b[0][1] + a[0][1] * b[1][1]
	mul02 := new(big.Int).Mul(a[0][0], b[0][1])
	mul03 := new(big.Int).Mul(a[0][1], b[1][1])
	result[0][1].Add(mul02, mul03)

	// result[1][0] = a[1][0] * b[0][0] + a[1][1] * b[1][0]
	mul10 := new(big.Int).Mul(a[1][0], b[0][0])
	mul11 := new(big.Int).Mul(a[1][1], b[1][0])
	result[1][0].Add(mul10, mul11)

	// result[1][1] = a[1][0] * b[0][1] + a[1][1] * b[1][1]
	mul12 := new(big.Int).Mul(a[1][0], b[0][1])
	mul13 := new(big.Int).Mul(a[1][1], b[1][1])
	result[1][1].Add(mul12, mul13)

	return result
}

// getFromCache récupère une valeur du cache de manière thread-safe
func getFromCache(n int) (*big.Int, bool) {
	cacheMutex.RLock()         // Verrouiller le cache en lecture pour éviter les conflits
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

// clearMemoization efface efficacement toutes les entrées de la carte de mémoïsation
func clearMemoization() {
	cacheMutex.Lock()         // Verrouiller le cache en écriture
	defer cacheMutex.Unlock() // Déverrouiller à la fin de la fonction
	lruCache.Purge()          // Vider le cache
}

// printError affiche un message d'erreur dans un format cohérent
func printError(n int, err error) {
	fmt.Printf("fibMatrixPower(%d): %s\n", n, err) // Afficher l'erreur avec la valeur de n
}

// benchmarkFibWithWorkerPool effectue des tests de performance sur les calculs de Fibonacci pour une liste de valeurs
func benchmarkFibWithWorkerPool(ctx context.Context, nValues []int, repetitions int, workerCount int) {
	clearMemoization() // Effacer les valeurs précédemment mémorisées avant le benchmark

	// Canal pour gérer les travaux
	jobs := make(chan int)
	var wg sync.WaitGroup

	// Lancer les workers
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done() // Décrémenter le compteur du WaitGroup lorsque le worker termine
			for n := range jobs {
				select {
				case <-ctx.Done():
					// Arrêter si le contexte est annulé
					fmt.Printf("Worker %d: contexte annulé, raison: %s\n", workerID, ctx.Err())
					return
				default:
					totalExecTime := time.Duration(0)
					// Calculer le nombre de Fibonacci plusieurs fois pour obtenir une moyenne
					for i := 0; i < repetitions; i++ {
						start := time.Now() // Commencer le chronométrage
						_, err := fibMatrixPower(n)
						if err != nil {
							printError(n, err) // Afficher l'erreur si elle se produit
							continue
						}
						totalExecTime += time.Since(start) // Accumuler le temps d'exécution
					}
					// Calculer le temps d'exécution moyen
					avgExecTime := totalExecTime / time.Duration(repetitions)
					fmt.Printf("Worker %d: fibMatrixPower(%d) moyenne sur %d exécutions: %s\n", workerID, n, repetitions, avgExecTime)
				}
			}
		}(w)
	}

	// Lancer une goroutine pour envoyer les travaux
	go func() {
		defer close(jobs) // Fermer le canal des travaux une fois tous les travaux envoyés
		for _, n := range nValues {
			select {
			case <-ctx.Done():
				// Arrêter si le contexte est annulé
				return
			case jobs <- n:
				// Envoyer le travail au canal des jobs
			}
		}
	}()

	// Attendre la fin des workers
	wg.Wait()
}

// Fonction principale pour exécuter les tests de performance
func main() {
	nValues := []int{1000, 10000, 100000, 1000000, 10000000, 100000000}      // Valeurs pour lesquelles calculer Fibonacci
	repetitions := 100                                                       // Nombre de répétitions pour chaque valeur de n
	workerCount := 16                                                        // Nombre de workers à lancer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Contexte avec un délai d'expiration de 10 minutes
	defer cancel()                                                           // Annuler le contexte après exécution
	benchmarkFibWithWorkerPool(ctx, nValues, repetitions, workerCount)       // Exécuter le benchmark
}
