// -----------------------------------------------------------------------------------------
// Programme : Calcul de la Somme des Nombres de Fibonacci
// Langage : Go (Golang)
//
// Description :
// Ce programme calcule la somme des nombres de Fibonacci jusqu'au nième terme spécifié (n).
// Il utilise la méthode du doublage pour calculer efficacement chaque nombre de Fibonacci.
// L'algorithme est conçu pour exploiter le parallélisme, en répartissant le calcul sur plusieurs
// cœurs du processeur pour accélérer le traitement. Ce programme démontre une approche itérative
// de la méthode du doublage, particulièrement utile pour les calculs de grande envergure.
//
// Le programme crée un fichier "fibonacci_result.txt" dans lequel il enregistre la somme des
// nombres de Fibonacci, le nombre total de calculs effectués, le temps moyen par calcul et le
// temps d'exécution global.
//
// Détails d'implémentation :
// - La méthode `fibDoubling` calcule le nième nombre de Fibonacci en utilisant un algorithme
//   de doublage. Elle repose sur des opérations arithmétiques avancées sur de grands entiers
//   grâce au package "math/big" de Go, afin de garantir une précision infinie pour les calculs
//   même avec des valeurs extrêmement élevées de n.
// - Pour diviser le travail, le programme détermine le nombre de travailleurs en fonction du
//   nombre de cœurs du CPU disponible, permettant ainsi d'optimiser l'utilisation des ressources
//   matérielles.
// - Chaque travailleur calcule une portion de la série de Fibonacci et renvoie un résultat
//   partiel, qui est ensuite additionné pour obtenir le résultat final.
//
// Structure :
// - `fibDoubling(n int) (*big.Int, error)` : Fonction principale pour calculer le nième nombre
//   de Fibonacci en utilisant la méthode de doublage.
// - `fibDoublingHelperIterative(n int) *big.Int` : Fonction auxiliaire itérative qui applique
//   la méthode de doublage.
// - `calcFibonacci(start, end int, partialResult chan<- *big.Int, wg *sync.WaitGroup)` : Fonction
//   qui divise la liste de Fibonacci en segments et calcule la somme des valeurs dans chaque
//   segment.
// - `main()` : Fonction principale qui orchestre les calculs en parallèle, effectue les mesures
//   de temps, et écrit les résultats dans un fichier.
//
// Usage :
// Ce programme est conçu pour des utilisateurs ayant des connaissances en programmation et en
// calculs mathématiques avancés. Il peut être utilisé pour étudier la croissance des nombres de
// Fibonacci et évaluer les performances des algorithmes parallèles.
//
// Avertissements :
// - Ce programme consomme une quantité importante de mémoire et de puissance de calcul en raison
//   des grands nombres de Fibonacci manipulés, particulièrement pour des valeurs élevées de n.
// - Il est conseillé de l'exécuter sur une machine disposant de plusieurs cœurs de CPU pour
//   bénéficier pleinement de l'implémentation concurrente.
//
// -----------------------------------------------------------------------------------------

package main

import (
	"fmt"
	"math/big"
	"math/bits"
	"os"
	"runtime"
	"sync"
	"time"
)

var memo sync.Map

// fibDoubling calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
func fibDoubling(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif") // Vérification des arguments : n doit être un entier positif
	}
	if n > 100000000 {
		return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux et consommation excessive de mémoire") // Limitation pour éviter des calculs extrêmement coûteux
	}
	if n < 2 {
		return big.NewInt(int64(n)), nil // Les deux premiers termes de la suite de Fibonacci sont connus : 0 et 1
	}
	result := fibDoublingHelperIterative(n) // Calcul du nième nombre de Fibonacci en utilisant la méthode de doublage
	return result, nil
}

// fibDoublingHelperIterative est une fonction itérative qui utilise la méthode de doublage pour calculer les nombres de Fibonacci
func fibDoublingHelperIterative(n int) *big.Int {
	a := big.NewInt(0) // Initialisation de a avec 0 (F(0))
	b := big.NewInt(1) // Initialisation de b avec 1 (F(1))
	c := new(big.Int)  // Variable auxiliaire pour les calculs

	// Parcours des bits de n, de gauche à droite
	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		// c = 2 * b - a
		c.Lsh(b, 1) // c = b << 1 (c = 2 * b)
		c.Sub(c, a) // c = c - a (c = 2 * b - a)
		c.Mul(c, a) // c = c * a
		// b = a * a + b * b
		b.Mul(a, a)           // b = a * a
		b.Add(b, b.Mul(b, b)) // b = b + (b * b) (b = a^2 + b^2)

		// Si le bit courant est 0, mettre à jour a et b en fonction
		if ((n >> i) & 1) == 0 {
			a.Set(c)
			b.Set(b)
		} else {
			a.Set(b)
			b.Add(c, b) // Si le bit courant est 1, mettre à jour a et b différemment
		}
	}

	return a // Retourne le nième nombre de Fibonacci
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, partialResult chan<- *big.Int, wg *sync.WaitGroup) {
	defer wg.Done() // Indique que cette routine est terminée une fois la fonction terminée

	partialSum := new(big.Int) // Utilisation de new(big.Int) pour éviter les allocations répétées de mémoire
	for i := start; i <= end; i++ {
		fibValue, _ := fibDoubling(i)        // Calcul de F(i)
		partialSum.Add(partialSum, fibValue) // Ajoute F(i) à la somme partielle
	}

	partialResult <- partialSum // Envoie la somme partielle au canal
}

func main() {
	n := 100000000                 // Nombre jusqu'auquel la somme de Fibonacci doit être calculée
	numWorkers := runtime.NumCPU() // Nombre de travailleurs basé sur le nombre de cœurs de CPU disponibles
	segmentSize := n / numWorkers  // Taille de chaque segment à calculer par chaque travailleur
	remaining := n % numWorkers    // Les éléments restants si n n'est pas divisible par numWorkers

	partialResult := make(chan *big.Int, numWorkers) // Taille du tampon du canal ajustée à `numWorkers` pour réduire la consommation de mémoire
	var wg sync.WaitGroup

	startTime := time.Now() // Commence la mesure du temps d'exécution

	// Démarre les travailleurs pour calculer les segments
	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize       // Début du segment
		end := start + segmentSize - 1 // Fin du segment
		if i == numWorkers-1 {
			end += remaining // Ajoute les éléments restants au dernier travailleur
		}

		wg.Add(1)                                        // Indique qu'une nouvelle goroutine va commencer
		go calcFibonacci(start, end, partialResult, &wg) // Lance la fonction de calcul de Fibonacci dans une nouvelle goroutine
	}

	// Ferme le canal une fois que tous les travailleurs ont terminé
	go func() {
		wg.Wait()
		close(partialResult)
	}()

	sumFib := new(big.Int) // Utilisation de new(big.Int) pour éviter les allocations répétées de mémoire
	numCalculations := 0   // Compteur du nombre de calculs effectués
	for partial := range partialResult {
		sumFib.Add(sumFib, partial) // Ajoute la somme partielle à la somme totale
		numCalculations++           // Incrémente le compteur
	}

	executionTime := time.Since(startTime)                                  // Calcule le temps total d'exécution
	avgTimePerCalculation := executionTime / time.Duration(numCalculations) // Calcule le temps moyen par calcul

	// Création du fichier pour écrire les résultats
	file, err := os.Create("fibonacci_result.txt")
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close() // Ferme le fichier à la fin de la fonction

	// Écriture des résultats dans le fichier
	_, err = file.WriteString(fmt.Sprintf("Somme des Fib(%d) = %s\n", n, sumFib.String()))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du résultat dans le fichier:", err)
		return
	}

	_, err = file.WriteString(fmt.Sprintf("Nombre de calculs: %d\n", numCalculations))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du nombre de calculs dans le fichier:", err)
		return
	}

	_, err = file.WriteString(fmt.Sprintf("Temps moyen par calcul: %s\n", avgTimePerCalculation))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du temps moyen par calcul dans le fichier:", err)
		return
	}

	_, err = file.WriteString(fmt.Sprintf("Temps d'exécution: %s\n", executionTime))
	if err != nil {
		fmt.Println("Erreur lors de l'écriture du temps d'exécution dans le fichier:", err)
		return
	}

	// Affichage des résultats dans la console
	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Nombre de calculs: %d\n", numCalculations)
	fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	fmt.Println("Résultat, nombre de calculs, temps moyen par calcul et temps d'exécution écrits dans 'fibonacci_result.txt'.")
}
