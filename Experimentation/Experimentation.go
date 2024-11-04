// DoublingParallel-Fibonacci Fast Calculation in Go
// Ce programme est une version traduite et adaptée d'un code en C# pour calculer un nombre de Fibonacci en utilisant la méthode de "Doubling".
// Il exploite une pile de tuples pour générer les valeurs et utilise sync.Pool pour optimiser la gestion des ressources.

package main

import (
	"fmt"
	"math/big"
	"time"
)

// Déclaration des structures utilisées
type Tuple struct {
	val1 int
	val2 int
}

type StackElement struct {
	index  int
	values Tuple
}

// Déclaration des variables globales
var fiboKnown []TupleBig
var stack []StackElement
var number int

// TupleBig encapsule l'indice et la valeur du nombre de Fibonacci
type TupleBig struct {
	index int
	value *big.Int
}

// FibCalculator encapsule les variables big.Int réutilisables
type FibCalculator struct {
	a, b, c, temp *big.Int
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

// ReadNumber permet de lire l'entrée de l'utilisateur
func ReadNumber() {
	fmt.Println("Note: F(0) = 0, F(1) = 1, F(2) = 1, n >= 2")
	fmt.Println("Entrez n, où n est le n-ième nombre de Fibonacci: ")
	fmt.Scan(&number)
	if number < 2 {
		ReadNumber()
	}
}

// CreateStack construit la pile d'éléments nécessaires au calcul
func CreateStack(stack *[]StackElement) {
	for i := len(*stack) - 1; i >= 0; i-- {
		element := (*stack)[i]
		if contains(*stack, element.values.val1) {
			continue
		}
		*stack = append(*stack, StackElement{index: element.values.val1, values: Tuple{val1: element.values.val1 / 2, val2: element.values.val1/2 + 1}})
		if contains(*stack, element.values.val2) {
			continue
		}
		*stack = append(*stack, StackElement{index: element.values.val2, values: Tuple{val1: element.values.val2 / 2, val2: element.values.val2/2 + 1}})
		CreateStack(stack)
	}
}

// contains vérifie si la pile contient un élément avec l'indice donné
func contains(stack []StackElement, index int) bool {
	for _, el := range stack {
		if el.index == index {
			return true
		}
	}
	return false
}

// CreateFibonacci génère les valeurs de Fibonacci en utilisant la méthode "Doubling"
func CreateFibonacci() {
	for i := len(stack) - 1; i >= 0; i-- {
		el := stack[i]
		val1 := findValue(el.values.val1)
		val2 := findValue(el.values.val2)
		var value *big.I// DoublingParallel-ChatGPTCanvas
		//
		// Ce programme implémente un calcul parallèle de la somme des nombres de Fibonacci
		// en utilisant une méthode de décomposition binaire optimisée et des goroutines en Go.
		// Le calcul est distribué entre plusieurs workers, chaque worker utilisant un
		// calculateur de Fibonacci encapsulé dans la structure `FibCalculator`.
		// L'objectif principal est d'optimiser les calculs en exploitant le parallélisme
		// et en évitant les recalculs inutiles grâce à une utilisation judicieuse des
		// primitives de synchronisation.
		//
		// Le programme comprend les composants suivants :
		// 1. `FibCalculator` : Structure encapsulant les variables nécessaires pour le calcul
		//    des nombres de Fibonacci de manière thread-safe, en utilisant de grandes valeurs entières (`math/big`).
		// 2. `WorkerPool` : Structure gérant un pool de calculateurs de Fibonacci, permettant
		//    d'allouer des ressources de calcul aux différentes tâches parallèles.
		// 3. `calcFibonacci` : Fonction qui calcule une portion des nombres de Fibonacci entre
		//    deux bornes et accumule les résultats partiels.
		// 4. `main` : Fonction principale qui initialise les paramètres de calcul, divise la charge
		//    de travail entre les goroutines, synchronise les résultats et mesure le temps d'exécution.
		//
		// Ce programme est conçu pour utiliser efficacement les ressources CPU disponibles,
		// en divisant la charge de travail de calcul de la série de Fibonacci en segments gérés par plusieurs workers.
		// Les résultats sont accumulés et affichés avec des statistiques de performance, telles que
		// le temps moyen par calcul et le temps d'exécution total.
		
		package main
		
		import (
			"fmt"       // Le package 'fmt' est utilisé pour la sortie formatée, comme 'Println' ou 'Printf' pour afficher des informations dans la console.
			"math/big"  // Le package 'math/big' permet la manipulation de nombres entiers très grands, ici utilisé pour calculer des valeurs de Fibonacci potentiellement très élevées.
			"math/bits" // Le package 'math/bits' est utilisé pour manipuler les bits des entiers, par exemple pour trouver la longueur binaire d'un nombre, ce qui est utile dans l'optimisation du calcul de Fibonacci.
			"runtime"   // Le package 'runtime' est utilisé pour obtenir des informations sur le système, comme le nombre de processeurs disponibles, afin d'optimiser le nombre de workers.
			"strings"   // Le package 'strings' est utilisé pour manipuler des chaînes de caractères, par exemple pour formater un 'big.Int' en notation scientifique.
			"sync"      // Le package 'sync' fournit des primitives pour synchroniser les goroutines, comme 'Mutex' pour les sections critiques et 'WaitGroup' pour attendre la fin de plusieurs goroutines.
			"time"      // Le package 'time' est utilisé pour mesurer les durées d'exécution et calculer le temps pris par des opérations spécifiques.
		)
		
		// FibCalculator encapsule les variables big.Int réutilisables
		type FibCalculator struct {
			a, b, c, temp *big.Int
			mutex         sync.Mutex
		}
		
		// NewFibCalculator crée une nouvelle instance de FibCalculator
		func NewFibCalculator() *FibCalculator {
			// Initialisation des valeurs de Fibonacci (a = 0, b = 1, c et temp sont des tampons)
			return &FibCalculator{
				a:    big.NewInt(0),
				b:    big.NewInt(1),
				c:    new(big.Int),
				temp: new(big.Int),
			}
		}
		
		// Calculate calcule le n-ième nombre de Fibonacci de manière thread-safe
		func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
			// Vérification de la validité de n (doit être positif)
			if n < 0 {
				return nil, fmt.Errorf("n doit être un entier positif")
			}
			if n > 1000000 {
				return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux")
			}
		
			// Verrouillage pour garantir que le calcul est thread-safe
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
		
				// Sauvegarde temporaire de b (pour utilisation ultérieure)
				fc.temp.Set(fc.b)
		
				// b = a² + b²
				fc.b.Mul(fc.b, fc.b) // b = b²
				fc.a.Mul(fc.a, fc.a) // a = a²
				fc.b.Add(fc.b, fc.a) // b = a² + b²
		
				// Si le bit correspondant est 0, a prend la valeur de c
				// Sinon, a prend la valeur de b et b devient c + b
				if ((n >> i) & 1) == 0 {
					fc.a.Set(fc.c) // a = c
					fc.b.Set(fc.b) // b reste inchangé
				} else {
					fc.a.Set(fc.b)       // a = b
					fc.b.Add(fc.c, fc.b) // b = c + b
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
			// Initialisation du pool avec le nombre de calculateurs spécifié
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
			// Verrouillage pour garantir un accès thread-safe au pool de calculateurs
			wp.mutex.Lock()
			defer wp.mutex.Unlock()
		
			// Récupère le calculateur actuel et met à jour l'indice courant de manière circulaire
			calc := wp.calculators[wp.current]
			wp.current = (wp.current + 1) % len(wp.calculators)
			return calc
		}
		
		// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
		func calcFibonacci(start, end int, pool *WorkerPool, partialResult chan<- *big.Int) {
			// Récupère un calculateur du pool
			calc := pool.GetCalculator()
			partialSum := new(big.Int)
		
			// Calcule la somme des valeurs de Fibonacci entre start et end
			for i := start; i <= end; i++ {
				fibValue, err := calc.Calculate(i)
				if err != nil {
					fmt.Printf("Erreur lors du calcul de Fib(%d): %v\n", i, err)
					continue
				}
				partialSum.Add(partialSum, fibValue) // Ajoute la valeur de Fibonacci à la somme partielle
			}
		
			// Envoie le résultat partiel au canal
			partialResult <- partialSum
		}
		
		// formatBigIntSci formate un big.Int en notation scientifique
		func formatBigIntSci(n *big.Int) string {
			numStr := n.String()
			numLen := len(numStr)
		
			// Si le nombre est petit, le retourner directement
			if numLen <= 5 {
				return numStr
			}
		
			// Formater le nombre en notation scientifique
			significand := numStr[:5]
			exponent := numLen - 1
		
			// Crée une représentation significative et supprime les zéros inutiles
			formattedNum := significand[:1] + "." + significand[1:]
			formattedNum = strings.TrimRight(strings.TrimRight(formattedNum, "0"), ".")
		
			return fmt.Sprintf("%se%d", formattedNum, exponent)
		}
		
		func main() {
			// Initialisation des paramètres pour le calcul
			n := 100000
			n = n - 1
			numWorkers := runtime.NumCPU()                   // Utilise le nombre de CPU disponibles
			segmentSize := n / (numWorkers * 2)              // Taille de chaque segment à traiter par un worker
			pool := NewWorkerPool(numWorkers)                // Création du pool de calculateurs
			taskChannel := make(chan [2]int, numWorkers*4)   // Canal pour les segments de travail
			partialResult := make(chan *big.Int, numWorkers) // Canal pour les résultats partiels
			var wg sync.WaitGroup
		
			// Initialiser les segments de travail et les envoyer au canal de tâches
			for i := 0; i < n; i += segmentSize {
				end := i + segmentSize - 1
				if end >= n {
					end = n - 1
				}
				taskChannel <- [2]int{i, end} // Envoie le segment de travail au canal
			}
			close(taskChannel)
		
			// Lancer les goroutines du pool pour traiter les tâches
			startTime := time.Now() // Enregistre l'heure de début pour mesurer la durée
			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					// Traite les segments jusqu'à ce que le canal soit fermé
					for segment := range taskChannel {
						calcFibonacci(segment[0], segment[1], pool, partialResult)
					}
				}()
			}
		
			// Fonction pour fermer le canal une fois que tous les travailleurs ont terminé
			go func() {
				wg.Wait()
				close(partialResult)
			}()
		
			sumFib := new(big.Int)
			count := 0
		
			// Récupérer et additionner les résultats partiels
			for partial := range partialResult {
				sumFib.Add(sumFib, partial)
				count++
			}
		
			executionTime := time.Since(startTime)                        // Calcule le temps total d'exécution
			avgTimePerCalculation := executionTime / time.Duration(count) // Temps moyen par calcul
		
			// Afficher les résultats
			fmt.Printf("Nombre de workers: %d\n", numWorkers)
			fmt.Printf("Temps moyen par calcul: %s\n", avgTimePerCalculation)
			fmt.Printf("Temps d'exécution: %s\n", executionTime)
			fmt.Printf("Somme des Fibonacci: %s\n", formatBigIntSci(sumFib))
		}
		nt
		if el.index%2 == 0 {
			value = new(big.Int).Mul(val1, new(big.Int).Sub(new(big.Int).Mul(big.NewInt(2), val2), val1))
		} else {
			temp1 := new(big.Int).Mul(val1, val1)
			temp2 := new(big.Int).Mul(val2, val2)
			value = new(big.Int).Add(temp1, temp2)
		}
		fiboKnown = append(fiboKnown, TupleBig{index: el.index, value: value})
	}
}

// findValue recherche une valeur dans fiboKnown par l'indice
func findValue(index int) *big.Int {
	for _, el := range fiboKnown {
		if el.index == index {
			return el.value
		}
	}
	return big.NewInt(0)
}

// FormatBigIntToScientific converts a big.Int to scientific notation with 5 significant digits
func FormatBigIntToScientific(n *big.Int) string {
	str := n.String()
	if len(str) <= 5 {
		return str
	}
	return fmt.Sprintf("%s.%se%d", str[:1], str[1:5], len(str)-1)
}

func main() {
	// Initialiser les valeurs connues de Fibonacci
	fiboKnown = []TupleBig{
		{index: 0, value: big.NewInt(0)},
		{index: 1, value: big.NewInt(1)},
		{index: 2, value: big.NewInt(1)},
	}

	ReadNumber()
	start := time.Now()

	// Initialiser la pile de travail
	stack = []StackElement{{index: number, values: Tuple{val1: number / 2, val2: number/2 + 1}}}
	CreateStack(&stack)

	// Calculer les valeurs de Fibonacci
	CreateFibonacci()
	end := time.Since(start)

	// Afficher le résultat
	result := findValue(number)
	formattedResult := FormatBigIntToScientific(result)
	fmt.Printf("Le %d-ième nombre de Fibonacci est: %s. Calcul effectué en %v\n", number, formattedResult, end)
}
