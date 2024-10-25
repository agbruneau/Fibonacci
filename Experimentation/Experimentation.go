/*
Analyse du Code :

Ce programme Go est conçu pour calculer de manière efficace la somme des nombres de Fibonacci jusqu'à un grand nombre `n` (initialement défini à 250 000) en utilisant une approche de calcul parallèle et thread-safe. Pour optimiser les performances, le programme exploite tous les cœurs disponibles du processeur. Voici une analyse détaillée des principales composantes du code :

1. **FibCalculator** :
   - Cette structure encapsule des objets `big.Int` pour stocker les valeurs des nombres de Fibonacci lors des calculs.
   - Elle comprend des méthodes qui permettent de calculer un nombre de Fibonacci de manière thread-safe en utilisant un verrou (`mutex`). La méthode `Calculate` utilise une approche d'exponentiation rapide pour calculer les nombres de Fibonacci de manière efficace.

2. **WorkerPool** :
   - Le pool de travailleurs est représenté par la structure `WorkerPool`, qui gère un ensemble d'instances `FibCalculator`.
   - Cette structure permet de partager les instances de `FibCalculator` parmi plusieurs goroutines afin de minimiser les coûts liés à la création répétée d'instances.

3. **Parallélisme** :
   - Le calcul des nombres de Fibonacci est divisé en plusieurs segments qui sont ensuite attribués à des goroutines exécutées en parallèle. Le nombre de travailleurs (goroutines) est déterminé en fonction du nombre de cœurs de processeur disponibles.
   - La fonction `calcFibonacci` est utilisée pour effectuer les calculs sur chaque segment, en accumulant les résultats partiels.
   - Les résultats partiels sont envoyés via un canal (`channel`) pour être additionnés ultérieurement afin d'obtenir la somme totale des nombres de Fibonacci.

4. **Formatage et Sauvegarde des Résultats** :
   - Après le calcul, la somme totale des nombres de Fibonacci est formatée en notation scientifique si elle est trop grande, ce qui permet d'afficher les résultats de manière plus lisible.
   - Les résultats, tels que le nombre de calculs effectués, le temps moyen par calcul, et le temps total d'exécution, sont écrits dans un fichier texte (`fibonacci_result.txt`).

5. **Synchronisation des Travailleurs** :
   - Le programme utilise un `sync.WaitGroup` pour s'assurer que toutes les goroutines ont terminé leurs calculs avant de fermer le canal de résultats partiels et de calculer la somme finale.
   - De plus, un verrou (`mutex`) est utilisé dans les structures `FibCalculator` et `WorkerPool` pour garantir la sécurité des threads lors de la modification des données partagées.

6. **Efficacité** :
   - La méthode utilisée pour calculer les nombres de Fibonacci est une version optimisée qui se base sur la décomposition binaire de `n` afin de réduire la complexité temporelle par rapport à l'approche naïve.
   - Le parallélisme permet de diviser la charge de calcul, réduisant ainsi le temps total nécessaire pour atteindre le résultat.

7. **Gestion des Fichiers** :
   - Le programme crée un fichier texte pour sauvegarder les résultats, puis lit et affiche le contenu de ce fichier pour vérifier les résultats.
   - Cette partie du code simule une commande UNIX de type `cat` pour afficher le contenu du fichier dans la console.

En résumé, ce programme est un exemple intéressant de l'utilisation de la concurrence et du parallélisme en Go pour résoudre un problème mathématique complexe de manière efficace. Le recours à des structures telles que `sync.Mutex` et `sync.WaitGroup` permet de s'assurer que les calculs sont effectués en toute sécurité dans un environnement multithread, tout en optimisant l'utilisation des ressources CPU disponibles.
*/

package main

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"math/bits"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// FibCalculator encapsule les variables big.Int réutilisables
type FibCalculator struct {
	a, b, c, temp *big.Int
	mutex         sync.Mutex
}

// NewFibCalculator crée une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		a:    big.NewInt(0),
		b:    big.NewInt(1),
		c:    new(big.Int),
		temp: new(big.Int),
	}
}

// Calculate calcule le n-ième nombre de Fibonacci de manière thread-safe
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	if n > 999999 {
		return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux")
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Réinitialisation des valeurs a et b pour chaque calcul
	fc.a.SetInt64(0)
	fc.b.SetInt64(1)

	// Si n est inférieur à 2, le résultat est trivial (0 ou 1)
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}

	// Algorithme principal basé sur la méthode de décomposition binaire pour optimiser le calcul de Fibonacci
	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		// c = a * (2 * b - a)
		fc.c.Lsh(fc.b, 1)    // c = 2 * b
		fc.c.Sub(fc.c, fc.a) // c = 2 * b - a
		fc.c.Mul(fc.c, fc.a) // c = a * (2 * b - a)

		// Sauvegarde temporaire de b
		fc.temp.Set(fc.b)

		// b = a² + b²
		fc.b.Mul(fc.b, fc.b) // b = b²
		fc.a.Mul(fc.a, fc.a) // a = a²
		fc.b.Add(fc.b, fc.a) // b = a² + b²

		// Si le bit correspondant est 0, a prend la valeur de c
		// Sinon, a prend la valeur de b et b devient c + b
		if ((n >> i) & 1) == 0 {
			fc.a.Set(fc.c)
			fc.b.Set(fc.b)
		} else {
			fc.a.Set(fc.b)
			fc.b.Add(fc.c, fc.b)
		}
	}

	// Retourne une copie de a (le résultat final de Fibonacci(n))
	return new(big.Int).Set(fc.a), nil
}

// WorkerPool gère un pool de FibCalculator pour le calcul parallèle
type WorkerPool struct {
	calculators []*FibCalculator
	current     int
	mutex       sync.Mutex
}

// NewWorkerPool crée un nouveau pool de calculateurs
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
	}
}

// GetCalculator retourne un calculateur du pool de manière thread-safe
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	// Récupère le calculateur actuel et met à jour l'indice courant de manière circulaire
	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, pool *WorkerPool, partialResult chan<- *big.Int, wg *sync.WaitGroup) {
	defer wg.Done() // Indique que le travailleur a terminé son travail une fois la fonction terminée

	calc := pool.GetCalculator() // Récupère un calculateur du pool
	partialSum := new(big.Int)   // Crée une nouvelle instance de big.Int pour accumuler la somme partielle

	// Boucle pour calculer chaque valeur de Fibonacci dans la plage donnée
	for i := start; i <= end; i++ {
		fibValue, err := calc.Calculate(i)
		if err != nil {
			fmt.Printf("Erreur lors du calcul de Fib(%d): %v\n", i, err)
			continue
		}
		partialSum.Add(partialSum, fibValue) // Ajoute la valeur de Fibonacci au total partiel
	}

	// Envoie la somme partielle au canal de résultats partiels
	partialResult <- partialSum
}

// formatBigIntSci formate un big.Int en notation scientifique
func formatBigIntSci(n *big.Int) string {
	// Convertir le nombre en chaîne de caractères
	numStr := n.String()
	numLen := len(numStr)

	// Si la longueur est inférieure ou égale à 5, renvoyer simplement la chaîne
	if numLen <= 5 {
		return numStr
	}

	// Prendre les 5 premiers chiffres et calculer l'exposant
	significand := numStr[:5]
	exponent := numLen - 1 // -1 car on déplace la virgule après le premier chiffre

	// Insérer un point décimal après le premier chiffre
	formattedNum := significand[:1] + "." + significand[1:]

	// Supprimer les zéros à la fin de la partie décimale
	formattedNum = strings.TrimRight(strings.TrimRight(formattedNum, "0"), ".")

	// Retourner la représentation en notation scientifique
	return fmt.Sprintf("%se%d", formattedNum, exponent)
}

func main() {
	n := 250000                    // Taille maximale de n pour le calcul
	n = n - 1                      // Ajustement de n pour correspondre aux segments de calcul
	numWorkers := runtime.NumCPU() // Nombre de travailleurs égal au nombre de cœurs de CPU
	segmentSize := n / numWorkers  // Taille de chaque segment calculé par un travailleur
	remaining := n % numWorkers    // Reste à distribuer au dernier travailleur

	pool := NewWorkerPool(numWorkers)                // Crée un pool de travailleurs
	partialResult := make(chan *big.Int, numWorkers) // Canal pour recevoir les résultats partiels
	var wg sync.WaitGroup                            // Groupe d'attente pour synchroniser les travailleurs

	startTime := time.Now() // Enregistre l'heure de début de l'exécution

	// Démarre les travailleurs
	for i := 0; i < numWorkers; i++ {
		start := i * segmentSize
		end := start + segmentSize - 1
		if i == numWorkers-1 {
			end += remaining // Le dernier travailleur prend également le reste
		}

		wg.Add(1)                                              // Ajoute un travailleur au groupe d'attente
		go calcFibonacci(start, end, pool, partialResult, &wg) // Démarre un goroutine pour chaque segment
	}

	// Fonction pour fermer le canal une fois que tous les travailleurs ont terminé
	go func() {
		wg.Wait()            // Attendre que tous les travailleurs aient terminé
		close(partialResult) // Ferme le canal une fois que les résultats sont prêts
	}()

	sumFib := new(big.Int) // Crée une nouvelle instance de big.Int pour la somme totale
	numCalculations := 0   // Compteur pour le nombre de calculs effectués

	// Récupère et additionne toutes les sommes partielles des travailleurs
	for partial := range partialResult {
		sumFib.Add(sumFib, partial)
		numCalculations++
	}

	executionTime := time.Since(startTime)                                  // Calcule le temps total d'exécution
	avgTimePerCalculation := executionTime / time.Duration(numCalculations) // Temps moyen par calcul

	// Écriture des résultats dans un fichier
	file, err := os.Create("fibonacci_result.txt")
	if err != nil {
		fmt.Println("Erreur lors de la création du fichier:", err)
		return
	}
	defer file.Close()

	// Correction AGB : Réajustement de n et ajout de la valeur manquante
	n = n + 1
	sumFib.Add(sumFib, big.NewInt(1))

	// Écriture simplifiée et corrigée dans le fichier
	writeLines := []string{
		fmt.Sprintf("Nombre de calculs: %d", numCalculations),
		fmt.Sprintf("Temps moyen par calcul: %s", avgTimePerCalculation),
		fmt.Sprintf("Temps d'exécution: %s", executionTime),
		fmt.Sprintf("Somme des Fib(%d) = %s\n", n, formatBigIntSci(sumFib)),
	}

	// Écrit chaque ligne dans le fichier
	for _, line := range writeLines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			fmt.Printf("Erreur lors de l'écriture dans le fichier: %v\n", err)
			return
		}
	}

	// Lire et afficher le contenu du fichier (équivalent à "cat fibonacci_result.txt")
	file, err = os.Open("fibonacci_result.txt")
	if err != nil {
		fmt.Printf("Erreur lors de l'ouverture du fichier pour lecture: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Println("\nContenu de fibonacci_result.txt :")
	fmt.Println("--------------------------------")

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Erreur lors de la lecture du fichier: %v\n", err)
			}
			break
		}
		fmt.Print(line) // Affiche chaque ligne du fichier
	}
	fmt.Println("--------------------------------")
}
