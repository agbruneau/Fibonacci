package main

import (
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"
)

// Mémoïsation pour stocker les valeurs intermédiaires de Fibonacci
var memo sync.Map

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, partialResult chan<- *big.Int, wg *sync.WaitGroup) {
	defer wg.Done() // Indiquer la fin du travail pour cette goroutine

	// Initialiser les premiers nombres de Fibonacci
	a, b := big.NewInt(0), big.NewInt(1)

	// Utiliser la mémoïsation pour récupérer les valeurs déjà calculées
	if val, exists := memo.Load(start); exists {
		// Si la valeur de départ est déjà dans la mémoïsation, l'utiliser
		a = val.(*big.Int)
		a, b = b, new(big.Int).Add(a, b) // Continuer la séquence à partir de la valeur trouvée
	} else {
		// Calculer les premiers termes si nécessaire
		for i := 0; i < start; i++ {
			a, b = b, new(big.Int).Add(a, b) // Progression normale de Fibonacci
		}
	}

	// Calculer la sous-liste de Fibonacci et accumuler le résultat partiel
	partialSum := big.NewInt(0)
	for i := start; i <= end; i++ {
		a, b = b, new(big.Int).Add(a, b) // Calculer le terme suivant
		partialSum.Add(partialSum, a)    // Ajouter le terme au résultat partiel
	}

	// Stocker le dernier résultat dans la mémoïsation pour une utilisation future
	memo.Store(end, new(big.Int).Set(a))

	// Envoyer le résultat partiel au canal pour être combiné plus tard
	partialResult <- partialSum
}

func main() {
	// Taille de la suite de Fibonacci que nous voulons calculer (n)
	n := 1000000    // Par exemple, 1 000 000ème nombre de Fibonacci
	numWorkers := 4 // Nombre de goroutines pour effectuer le travail

	// Déterminer la taille du segment pour chaque goroutine
	segmentSize := n / numWorkers
	remaining := n % numWorkers // Nombre restant pour le dernier segment si n n'est pas divisible par numWorkers

	// Canal pour récupérer les résultats partiels calculés
	partialResult := make(chan *big.Int, numWorkers*2) // Taille du canal pour éviter les blocages

	// WaitGroup pour synchroniser les goroutines
	var wg sync.WaitGroup

	// Mesurer le temps de début
	startTime := time.Now()

	// Démarrer plusieurs goroutines pour calculer les segments
	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 { // S'assurer que le dernier segment couvre le reste
			end += remaining
		}

		wg.Add(1)                                        // Indiquer qu'une nouvelle goroutine est en cours
		go calcFibonacci(start, end, partialResult, &wg) // Lancer la goroutine pour calculer ce segment
	}

	// Attendre la fin des goroutines
	go func() {
		wg.Wait()            // Attendre que toutes les goroutines aient terminé leur travail
		close(partialResult) // Fermer le canal une fois toutes les goroutines terminées
	}()

	// Récupérer et combiner les résultats partiels des calculs de Fibonacci
	sumFib := big.NewInt(0)
	for partial := range partialResult {
		sumFib.Add(sumFib, partial) // Ajouter chaque résultat partiel à la somme totale
	}

	// Calculer le temps total écoulé
	executionTime := time.Since(startTime)

	// Ouvrir (ou créer) un fichier pour y écrire le résultat
	file, err := os.Create("fibonacci_result.txt")
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close() // Fermer le fichier à la fin, même en cas d'erreur

	// Écrire le résultat dans le fichier
	_, err = file.WriteString(fmt.Sprintf("Somme des Fib(%d) = %s\n", n, sumFib.String()))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du résultat dans le fichier:", err)
		return
	}

	// Afficher uniquement le temps d'exécution dans le terminal
	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Println("Résultat et temps d'exécution écrits dans 'fibonacci_result.txt'.")
}
