/*
Programme de calcul parallèle de la somme des nombres de Fibonacci jusqu'à un certain terme.

Description :
Ce programme calcule la somme des nombres de Fibonacci jusqu'à un certain nombre n de manière efficace et parallèle. Il utilise la technique de la mémoïsation pour éviter les recalculs inutiles des valeurs intermédiaires et répartit les calculs entre plusieurs goroutines pour optimiser le temps d'exécution. Les goroutines effectuent des calculs de segments de la séquence de Fibonacci, puis les résultats partiels sont agrégés pour obtenir la somme finale.

Fonctionnement :
- La mémoïsation est utilisée pour stocker les valeurs intermédiaires déjà calculées, afin de réduire le temps nécessaire pour les recalculer.
- Le programme divise le calcul en plusieurs segments, chacun géré par une goroutine, afin de tirer parti de la puissance des systèmes multi-cœurs.
- Les segments sont déterminés en fonction du nombre total de termes de Fibonacci à calculer et du nombre de workers (goroutines).
- Une fois que chaque goroutine termine son segment, elle envoie son résultat partiel via un canal, où les résultats sont ensuite combinés pour obtenir la somme finale.

Fonctionnalités principales :
- Calcul parallèle des termes de la suite de Fibonacci pour optimiser les performances.
- Utilisation d'un mécanisme de mémoïsation thread-safe pour éviter les recalculs des valeurs déjà connues.
- Gestion des goroutines à l'aide de WaitGroup pour garantir la synchronisation et la bonne gestion des ressources.
- Écriture du résultat final dans un fichier texte ainsi que l'affichage du temps d'exécution dans le terminal.

Bibliothèques utilisées :
- "math/big" : Utilisée pour gérer des entiers de grande taille, car les valeurs de Fibonacci peuvent rapidement dépasser la capacité des entiers standards.
- "sync" : Utilisée pour la synchronisation des goroutines, en particulier avec sync.Map pour la mémoïsation et sync.WaitGroup pour coordonner l'exécution des goroutines.
- "time" : Utilisée pour mesurer le temps d'exécution total du programme.
- "os" : Utilisée pour la gestion des fichiers, notamment pour créer et écrire dans le fichier résultat.

Usage :
- Le programme est conçu pour être exécuté directement, et il calcule par défaut la somme des nombres de Fibonacci jusqu'au millionième terme.
- Le résultat est écrit dans un fichier nommé 'fibonacci_result.txt'.

Auteur :
Ce programme est conçu pour démontrer l'utilisation de la programmation parallèle et de la mémoïsation en Go afin de résoudre des problèmes impliquant des calculs intensifs de manière efficace.
*/

// Mémoïsation pour stocker les valeurs intermédiaires de Fibonacci
package main

import (
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"
)

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
		// Continuer la séquence à partir de la valeur trouvée
		a, b = b, new(big.Int).Add(a, b)
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
	n := 1000000    // Par exemple, calculer jusqu'au 1 000 000ème nombre de Fibonacci
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

		// Ajouter une tâche au WaitGroup et lancer une goroutine pour calculer ce segment
		wg.Add(1)
		go calcFibonacci(start, end, partialResult, &wg)
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
