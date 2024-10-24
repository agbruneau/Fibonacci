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
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}
	result := fibDoublingHelperIterative(n)
	return result, nil
}

// fibDoublingHelperIterative est une fonction itérative qui utilise la méthode de doublage pour calculer les nombres de Fibonacci
func fibDoublingHelperIterative(n int) *big.Int {
	a := big.NewInt(0)
	b := big.NewInt(1)
	c := new(big.Int)

	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		c.Lsh(b, 1)
		c.Sub(c, a)
		c.Mul(c, a)
		b.Mul(a, a)
		b.Add(b, b.Mul(b, b))

		if ((n >> i) & 1) == 0 {
			a.Set(c)
			b.Set(b)
		} else {
			a.Set(b)
			b.Add(c, b)
		}
	}

	return a
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, partialResult chan<- *big.Int, wg *sync.WaitGroup) {
	defer wg.Done()

	partialSum := big.NewInt(0)
	for i := start; i <= end; i++ {
		fibValue, _ := fibDoubling(i)
		partialSum.Add(partialSum, fibValue)
	}

	partialResult <- partialSum
}

func main() {
	n := 100000000
	numWorkers := runtime.NumCPU()
	segmentSize := n / numWorkers
	remaining := n % numWorkers

	partialResult := make(chan *big.Int, numWorkers*2)
	var wg sync.WaitGroup

	startTime := time.Now()

	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 {
			end += remaining
		}

		wg.Add(1)
		go calcFibonacci(start, end, partialResult, &wg)
	}

	go func() {
		wg.Wait()
		close(partialResult)
	}()

	sumFib := big.NewInt(0)
	numCalculations := 0
	for partial := range partialResult {
		sumFib.Add(sumFib, partial)
		numCalculations++
	}

	executionTime := time.Since(startTime)
	avgTimePerCalculation := executionTime / time.Duration(numCalculations)

	file, err := os.Create("fibonacci_result.txt")
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close()

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

	fmt.Printf("Temps d'exécution: %s\n", executionTime)
	fmt.Printf("Nombre de calculs: %d\n", numCalculations)
	fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
	fmt.Println("Résultat, nombre de calculs, temps moyen par calcul et temps d'exécution écrits dans 'fibonacci_result.txt'.")
}
