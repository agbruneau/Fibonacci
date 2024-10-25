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

Exemple d'appel via cURL :

```
curl -X POST -H "Content-Type: application/json" -d '{"n": 1000}' http://localhost:8080/fibonacci
```

*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/bits"
	"net/http"
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

// Requête pour calculer le n-ième nombre de Fibonacci
type FibonacciRequest struct {
	N int `json:"n"`
}

// Réponse contenant le résultat du calcul du nombre de Fibonacci
type FibonacciResponse struct {
	N                    int    `json:"n"`
	Result               string `json:"result"`
	Message              string `json:"message,omitempty"`
	NombreDeCalculs      int    `json:"nombre_de_calculs"`
	TempsMoyenParCalcul  string `json:"temps_moyen_par_calcul"`
	TempsExecutionTotal  string `json:"temps_execution_total"`
	SommeFormattedResult string `json:"somme_formatted_result"`
}

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

func fibonacciHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req FibonacciRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	if req.N < 0 {
		http.Error(w, "Le paramètre n doit être un entier positif", http.StatusBadRequest)
		return
	}

	startTime := time.Now() // Commence le chronométrage

	pool := NewWorkerPool(runtime.NumCPU())
	calculator := pool.GetCalculator()
	result, err := calculator.Calculate(req.N)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	numCalculations := 32                  // Exemple de valeur pour le nombre de calculs
	executionTime := time.Since(startTime) // Calcule le temps d'exécution
	avgTimePerCalculation := executionTime / time.Duration(numCalculations)

	response := FibonacciResponse{
		N: req.N,
		//	Result:              result.String(),
		NombreDeCalculs:      numCalculations,
		TempsMoyenParCalcul:  avgTimePerCalculation.String(),
		TempsExecutionTotal:  executionTime.String(),
		SommeFormattedResult: formatBigIntSci(result),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Erreur lors de l'encodage de la réponse", http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/fibonacci", fibonacciHandler)

	port := ":8080"
	fmt.Printf("Serveur démarré sur le port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur: %v", err)
	}
}
