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
	"container/list"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"sync"
	"time"
)

const MAX_FIB_VALUE = 100000001 // Valeur maximale de n qui peut être calculée
var two = big.NewInt(2)         // Valeur constante 2 en tant que big.Int pour les calculs

// Carte de mémoïsation optimisée avec un meilleur contrôle de la concurrence
var memo = &sync.Map{}
var memoMutex = &sync.Mutex{} // Mutex pour fournir un contrôle de concurrence supplémentaire
var maxCacheSize = 1000       // Nombre maximal d'entrées dans le cache

// Implémentation du cache LRU
// Ce cache LRU est utilisé pour stocker les valeurs de Fibonacci les plus récemment calculées afin de réduire les calculs redondants.
type LRUCache struct {
	capacity int
	cache    map[int]*list.Element // Carte pour stocker les références aux éléments de la liste chaînée
	ll       *list.List            // Liste doublement chaînée pour maintenir l'ordre LRU
	mutex    sync.RWMutex          // Mutex en lecture-écriture pour le contrôle de la concurrence
}

type entry struct {
	key   int
	value *big.Int // Stocke la valeur de Fibonacci en tant que big.Int
}

// NewLRUCache crée un nouveau cache LRU avec la capacité donnée
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[int]*list.Element),
		ll:       list.New(),
	}
}

// Get récupère une valeur du cache LRU
func (c *LRUCache) Get(key int) (*big.Int, bool) {
	c.mutex.RLock() // Acquérir un verrou en lecture pour permettre plusieurs lecteurs
	if ele, ok := c.cache[key]; ok {
		value := ele.Value.(*entry).value
		c.mutex.RUnlock() // Libérer le verrou en lecture avant d'acquérir le verrou en écriture pour mettre à jour l'ordre LRU
		c.mutex.Lock()    // Acquérir un verrou en écriture pour déplacer l'élément en tête de liste
		c.ll.MoveToFront(ele)
		c.mutex.Unlock()
		return value, true
	}
	c.mutex.RUnlock()
	return nil, false
}

// Put ajoute une nouvelle valeur au cache LRU
func (c *LRUCache) Put(key int, value *big.Int) {
	c.mutex.Lock() // Acquérir un verrou en écriture pour ajouter/mettre à jour le cache
	defer c.mutex.Unlock()
	if ele, ok := c.cache[key]; ok {
		// Si la clé existe déjà, la déplacer en tête de liste et mettre à jour sa valeur
		c.ll.MoveToFront(ele)
		ele.Value.(*entry).value = value
		return
	}
	// Ajouter une nouvelle entrée en tête de liste
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele
	// Si le cache dépasse sa capacité, supprimer l'entrée la plus ancienne
	if c.ll.Len() > c.capacity {
		c.removeOldest()
	}
}

// removeOldest supprime l'élément le moins récemment utilisé du cache
func (c *LRUCache) removeOldest() {
	if ele := c.ll.Back(); ele != nil {
		c.ll.Remove(ele)
		e := ele.Value.(*entry)
		delete(c.cache, e.key)
	}
}

// Initialiser le cache LRU
var lruCache = NewLRUCache(maxCacheSize)

// fibDoubling calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
func fibDoubling(n int) (*big.Int, error) {
	// Retourner la valeur directement si n est 0 ou 1
	if n < 2 {
		return big.NewInt(int64(n)), nil
	} else if n > MAX_FIB_VALUE {
		// Erreur si la valeur est trop grande pour être calculée en un temps raisonnable
		return nil, errors.New("n est trop grand pour cette implémentation")
	}
	// Calculer la valeur de Fibonacci à l'aide d'une fonction itérative auxiliaire
	result := fibDoublingHelperIterative(n)
	return result, nil
}

// fibDoublingHelperIterative est une fonction itérative qui utilise la méthode de doublage pour calculer les nombres de Fibonacci
func fibDoublingHelperIterative(n int) *big.Int {
	// Vérifier si la valeur est déjà dans le cache LRU
	if val, exists := lruCache.Get(n); exists {
		return val
	}

	// Initialiser les valeurs de base de Fibonacci F(0) = 0 et F(1) = 1
	a, b := big.NewInt(0), big.NewInt(1)
	c, d := new(big.Int), new(big.Int) // Préallouer des variables big.Int pour les réutiliser dans les calculs

	// Déterminer le nombre de bits nécessaires pour représenter n
	bitLength := bits.Len(uint(n))

	// Itérer sur chaque bit du plus significatif au moins significatif
	for i := bitLength - 1; i >= 0; i-- {
		// Utiliser les formules de doublage :
		// F(2k) = F(k) * [2 * F(k+1) - F(k)]
		c.Mul(b, two) // c = 2 * F(k+1)
		c.Sub(c, a)   // c = 2 * F(k+1) - F(k)
		c.Mul(a, c)   // c = F(k) * (2 * F(k+1) - F(k))
		// F(2k + 1) = F(k)^2 + F(k+1)^2
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
	lruCache.Put(n, result)

	return result
}

// printError affiche un message d'erreur dans un format cohérent
func printError(n int, err error) {
	fmt.Printf("fibDoubling(%d): %s\n", n, err)
}

// clearMemoization efface efficacement toutes les entrées de la carte de mémoïsation
func clearMemoization() {
	memoMutex.Lock()
	defer memoMutex.Unlock()
	memo = &sync.Map{} // Remplacer l'ancienne carte de mémo par une nouvelle instance pour un effacement efficace
	lruCache = NewLRUCache(maxCacheSize) // Effacer le cache LRU en créant une nouvelle instance
}

// benchmarkFib effectue des tests de performance sur les calculs de Fibonacci pour une liste de valeurs
func benchmarkFib(nValues []int, repetitions int) {
	// Effacer la carte de mémoïsation avant les tests pour garantir des résultats cohérents
	clearMemoization()

	var wg sync.WaitGroup // WaitGroup pour gérer la concurrence

	for _, n := range nValues {
		// S'assurer que wg.Add(1) est appelé immédiatement avant la goroutine
		wg.Add(1)
		// Lancer une goroutine pour calculer Fibonacci de manière concurrente
		go func(n int) {
			defer wg.Done() // Marquer cette goroutine comme terminée lorsqu'elle se termine

			var totalExecTime time.Duration = 0
			// Répéter le calcul pour un meilleur test de performance
			for i := 0; i < repetitions; i++ {
				start := time.Now()
				_, err := fibDoubling(n)
				if err != nil {
					// Afficher un message d'erreur si n est trop grand
					printError(n, err)
					continue
				}
				// Accumuler le temps d'exécution
				totalExecTime += time.Since(start)
			}
			// Calculer le temps d'exécution moyen
			avgExecTime := totalExecTime / time.Duration(repetitions)
			// Afficher le temps d'exécution moyen pour la valeur donnée de n
			fmt.Printf("fibDoubling(%d) averaged over %d runs: %s\n", n, repetitions, avgExecTime)
		}(n)
	}

	// Attendre que toutes les goroutines soient terminées
	wg.Wait()
}

// Fonction principale pour exécuter les tests de performance
func main() {
	// Définir la liste des valeurs pour lesquelles effectuer les tests de performance du calcul de Fibonacci
	nValues := []int{1000000, 10000000, 100000000} // Liste des valeurs à tester
	// Définir le nombre de répétitions pour une meilleure précision
	repetitions := 3 // Nombre de répétitions pour une meilleure précision
	// Exécuter le test de performance
	benchmarkFib(nValues, repetitions)
}
