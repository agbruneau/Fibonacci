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

// fibMatrixPower calcule le nième nombre de Fibonacci en utilisant la méthode de la matrice de puissance
func fibMatrixPower(n int) (*big.Int, error) {
	// Si n est inférieur à 2, retourner directement le résultat correspondant
	if n < 2 {
		return big.NewInt(int64(n)), nil
	} else if n > MAX_FIB_VALUE {
		// Retourner une erreur si n est trop grand pour être calculé raisonnablement
		return nil, errors.New("n est trop grand pour cette implémentation")
	}

	// Vérifier si la valeur est déjà dans le cache LRU
	if val, exists := lruCache.Get(n); exists {
		// Retourner la valeur si elle est déjà mise en cache
		return val.(*big.Int), nil
	}

	// Initialiser les matrices de base
	F := [2][2]*big.Int{
		{big.NewInt(1), big.NewInt(1)},
		{big.NewInt(1), big.NewInt(0)},
	}
	result := matrixPower(F, n-1)

	// La valeur de Fibonacci est dans la case [0][0] de la matrice résultante
	fibValue := result[0][0]

	// Mettre en cache le résultat dans le cache LRU
	lruCache.Add(n, fibValue)

	// Retourner le résultat final
	return fibValue, nil
}

// matrixPower calcule la puissance d'une matrice à l'exposant n
func matrixPower(matrix [2][2]*big.Int, n int) [2][2]*big.Int {
	// Initialiser la matrice identité (matrice neutre pour la multiplication)
	result := [2][2]*big.Int{
		{big.NewInt(1), big.NewInt(0)},
		{big.NewInt(0), big.NewInt(1)},
	}

	// Utiliser l'exponentiation rapide pour calculer la puissance de la matrice
	base := matrix
	for n > 0 {
		// Si le bit courant est 1, multiplier le résultat par la base
		if n%2 == 1 {
			result = matrixMultiply(result, base)
		}
		// Multiplier la base par elle-même (exponentiation rapide)
		base = matrixMultiply(base, base)
		n /= 2
	}

	return result
}

// matrixMultiply multiplie deux matrices 2x2
func matrixMultiply(a, b [2][2]*big.Int) [2][2]*big.Int {
	// Effectuer la multiplication matricielle en utilisant des big.Int pour éviter les dépassements
	return [2][2]*big.Int{
		{
			// Calculer la valeur en [0][0]
			new(big.Int).Add(new(big.Int).Mul(a[0][0], b[0][0]), new(big.Int).Mul(a[0][1], b[1][0])),
			// Calculer la valeur en [0][1]
			new(big.Int).Add(new(big.Int).Mul(a[0][0], b[0][1]), new(big.Int).Mul(a[0][1], b[1][1])),
		},
		{
			// Calculer la valeur en [1][0]
			new(big.Int).Add(new(big.Int).Mul(a[1][0], b[0][0]), new(big.Int).Mul(a[1][1], b[1][0])),
			// Calculer la valeur en [1][1]
			new(big.Int).Add(new(big.Int).Mul(a[1][0], b[0][1]), new(big.Int).Mul(a[1][1], b[1][1])),
		},
	}
}

// printError affiche un message d'erreur dans un format cohérent
func printError(n int, err error) {
	// Afficher le message d'erreur pour une valeur de Fibonacci donnée
	fmt.Printf("fibMatrixPower(%d): %s\n", n, err)
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
						_, err := fibMatrixPower(n)
						if err != nil {
							// Afficher l'erreur si n est trop grand
							printError(n, err)
							continue
						}
						totalExecTime += time.Since(start)
					}
					// Calculer le temps d'exécution moyen
					avgExecTime := totalExecTime / time.Duration(repetitions)
					fmt.Printf("Worker %d: fibMatrixPower(%d) averaged over %d runs: %s\n", workerID, n, repetitions, avgExecTime)
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
