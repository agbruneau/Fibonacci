// Programme Go : Calcul de Fibonacci par la méthode du Doublement avec Mémoïsation et Benchmark
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
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

const MAX_FIB_VALUE = 500000001 // Valeur maximale de n qui peut être calculée
var two = big.NewInt(2)         // Valeur constante 2 en tant que big.Int pour les calculs
var maxCacheSize = 1000         // Nombre maximal d'entrées dans le cache

// Initialiser le cache LRU avec une bibliothèque optimisée
var lruCache *lru.Cache

func init() {
	// Initialisation du cache LRU
	var err error
	lruCache, err = lru.New(maxCacheSize)
	if err != nil {
		// Arrêter le programme si l'initialisation du cache échoue
		panic(fmt.Sprintf("Échec de l'initialisation du cache LRU : %v", err))
	}
}

// fibDoubling calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
func fibDoubling(n int) (*big.Int, error) {
	// Si n est inférieur à 2, retourner directement le résultat correspondant
	if n < 2 {
		return big.NewInt(int64(n)), nil
	} else if n > MAX_FIB_VALUE {
		// Retourner une erreur si n est trop grand pour être calculé raisonnablement
		return nil, errors.New("n est trop grand pour cette implémentation")
	}
	// Pour les petites valeurs de n (inférieures à 93), utiliser un int64 pour un calcul rapide
	if n < 93 {
		return big.NewInt(fibInt64(n)), nil
	}
	// Pour les grandes valeurs, utiliser la fonction de doublage itérative
	result := fibDoublingHelperIterative(n)
	return result, nil
}

// fibInt64 calcule le nième nombre de Fibonacci avec int64 si la valeur est petite
func fibInt64(n int) int64 {
	// Initialiser les deux premiers nombres de Fibonacci : F(0) = 0, F(1) = 1
	a, b := int64(0), int64(1)
	// Calculer le nombre de Fibonacci en utilisant une simple boucle
	for i := 0; i < n; i++ {
		// Vérifier le dépassement de capacité pour éviter des résultats incorrects
		if a > (1<<63-1)-b {
			panic("Dépassement de capacité détecté lors du calcul de Fibonacci avec int64")
		}
		a, b = b, a+b
	}
	return a
}

// fibDoublingHelperIterative est une fonction itérative qui utilise la méthode de doublage pour calculer les nombres de Fibonacci
func fibDoublingHelperIterative(n int) *big.Int {
	// Vérifier si la valeur est déjà dans le cache LRU
	if val, exists := lruCache.Get(n); exists {
		// Retourner la valeur si elle est déjà mise en cache
		return val.(*big.Int)
	}

	// Initialiser les valeurs de base de Fibonacci F(0) = 0 et F(1) = 1
	a, b := big.NewInt(0), big.NewInt(1)
	c, d := new(big.Int), new(big.Int) // Préallouer des variables big.Int pour les réutiliser dans les calculs

	// Déterminer le nombre de bits nécessaires pour représenter n
	bitLength := bits.Len(uint(n))

	// Itérer sur chaque bit du plus significatif au moins significatif
	for i := bitLength - 1; i >= 0; i-- {
		// Utiliser les formules de doublage pour calculer F(2k) et F(2k + 1)
		c.Mul(b, two)                    // c = 2 * F(k+1)
		c.Sub(c, a)                      // c = 2 * F(k+1) - F(k)
		c.Mul(a, c)                      // c = F(k) * (2 * F(k+1) - F(k))
		d.Mul(a, a)                      // d = F(k)^2
		d.Add(d, new(big.Int).Mul(b, b)) // d = F(k)^2 + F(k+1)^2

		// Mettre à jour a et b en fonction du bit actuel de n
		if (n>>i)&1 == 0 {
			a.Set(c) // Si le bit est 0, définir F(2k) sur a
			b.Set(d) // Définir F(2k+1) sur b
		} else {
			a.Set(d)    // Si le bit est 1, définir F(2k+1) sur a
			b.Add(c, d) // Définir F(2k + 2) sur b
		}
	}

	// Mettre en cache le résultat dans le cache LRU
	result := new(big.Int).Set(a)
	lruCache.Add(n, result)

	// Retourner le résultat final
	return result
}

// printError affiche un message d'erreur dans un format cohérent
func printError(n int, err error) {
	// Afficher le message d'erreur pour une valeur de Fibonacci donnée
	fmt.Printf("fibDoubling(%d): %s\n", n, err)
}

// clearMemoization efface efficacement toutes les entrées de la carte de mémoïsation
func clearMemoization() {
	// Réinitialiser le cache LRU en créant une nouvelle instance
	var err error
	lruCache, err = lru.New(maxCacheSize)
	if err != nil {
		// Arrêter le programme si l'initialisation du cache échoue
		panic(fmt.Sprintf("Échec de l'initialisation du cache LRU : %v", err))
	}
}

// benchmarkFib effectue des tests de performance sur les calculs de Fibonacci pour une liste de valeurs
func benchmarkFibWithWorkerPool(ctx context.Context, nValues []int, repetitions int, workerCount int) {
	// Effacer la carte de mémoïsation avant de commencer le benchmark
	clearMemoization()

	// Canal pour gérer les travaux
	jobs := make(chan int, len(nValues)*2) // Utiliser une taille de canal appropriée pour éviter le blocage des goroutines
	var wg sync.WaitGroup

	// Lancer un certain nombre de workers (limité par workerCount)
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done() // Assurez-vous de signaler la fin du travail du worker
			// Chaque worker traite les travaux du canal
			for n := range jobs {
				select {
				case <-ctx.Done():
					// Arrêter l'exécution si le contexte est annulé
					fmt.Printf("Worker %d: contexte annulé, raison: %s\n", workerID, ctx.Err())
					return
				default:
					var totalExecTime time.Duration = 0
					// Répéter le calcul pour obtenir un temps d'exécution moyen
					for i := 0; i < repetitions; i++ {
						start := time.Now()
						_, err := fibDoubling(n)
						if err != nil {
							// Afficher l'erreur si n est trop grand
							printError(n, err)
							continue
						}
						totalExecTime += time.Since(start)
					}
					// Calculer le temps d'exécution moyen
					avgExecTime := totalExecTime / time.Duration(repetitions)
					fmt.Printf("Worker %d: fibDoubling(%d) averaged over %d runs: %s\n", workerID, n, repetitions, avgExecTime)
				}
			}
		}(w)
	}

	// Ajouter des travaux au canal
	for _, n := range nValues {
		jobs <- n
	}
	// Fermer le canal une fois que tous les travaux sont ajoutés
	close(jobs)

	// Attendre la fin des goroutines
	wg.Wait()
}

// Fonction principale pour exécuter les tests de performance
func main() {
	// Définir la liste des valeurs pour lesquelles effectuer les tests de performance
	nValues := []int{100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000, 100000000, 200000000}
	// Nombre de répétitions pour calculer le temps moyen
	repetitions := 250
	// Nombre de goroutines concurrentes
	workerCount := 32
	// Créer un contexte avec annulation possible (timeout de 10 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel() // Annuler le contexte lorsque le benchmark est terminé
	// Exécuter le benchmark
	benchmarkFibWithWorkerPool(ctx, nValues, repetitions, workerCount)
}
