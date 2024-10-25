// -----------------------------------------------------------------------------------------
// Programme : Service Web pour Calcul de la Somme des Nombres de Fibonacci
// Langage : Go (Golang)
//
// Description :
// Ce programme expose un service Web qui calcule la somme des nombres de Fibonacci jusqu'au nième terme spécifié (n).
// Il utilise la méthode du doublage pour calculer efficacement chaque nombre de Fibonacci.
// L'algorithme est conçu pour exploiter le parallélisme, en répartissant le calcul sur plusieurs
// cœurs du processeur pour accélérer le traitement. Ce service Web démontre une approche itérative
// de la méthode du doublage, particulièrement utile pour les calculs de grande envergure.
//
// Le service répond aux requêtes HTTP POST avec un JSON spécifiant la valeur de n, et renvoie la somme des
// nombres de Fibonacci, le nombre total de calculs effectués, le temps moyen par calcul et le
// temps d'exécution global (en secondes).
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
// - `main()` : Fonction principale qui expose l'API REST pour répondre aux requêtes HTTP.
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
	"encoding/json" // Package pour encoder et décoder des objets JSON dans les requêtes HTTP
	"fmt"           // Package pour la gestion des formats d'entrée/sortie, utilisé ici pour la gestion des erreurs
	"log"           // Package pour l'enregistrement de messages de journalisation (logs)
	"math/big"      // Package pour manipuler des grands entiers (BigInt), nécessaire pour le calcul des grands nombres de Fibonacci
	"math/bits"     // Package pour la manipulation des bits, utilisé ici pour le parcours des bits d'un entier
	"net/http"      // Package pour implémenter un serveur HTTP, gérant les requêtes et réponses HTTP
	"runtime"       // Package pour obtenir des informations sur le système, comme le nombre de cœurs de CPU disponibles
	"sync"          // Package pour la synchronisation entre goroutines, utilisé ici avec les WaitGroup et Map
	"time"          // Package pour mesurer la durée des opérations, utile pour calculer le temps d'exécution
)

var memo sync.Map

// fibDoubling calcule le nième nombre de Fibonacci en utilisant la méthode de doublage
func fibDoubling(n int) (*big.Int, error) {
	// Vérification des arguments : n doit être un entier positif
	if n < 0 {
		return nil, fmt.Errorf("n doit être un entier positif")
	}
	// Limitation pour éviter des calculs extrêmement coûteux
	if n > 10000000000 {
		return nil, fmt.Errorf("n est trop grand, risque de calculs extrêmement coûteux et consommation excessive de mémoire")
	}
	// Les deux premiers termes de la suite de Fibonacci sont connus : 0 et 1
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}
	// Calcul du nième nombre de Fibonacci en utilisant la méthode de doublage
	result := fibDoublingHelperIterative(n)
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
			// Si le bit courant est 1, mettre à jour a et b différemment
			a.Set(b)
			b.Add(c, b)
		}
	}

	// Retourne le nième nombre de Fibonacci
	return a
}

// calcFibonacci calcule une portion de la liste de Fibonacci entre start et end
func calcFibonacci(start, end int, partialResult chan<- *big.Int, wg *sync.WaitGroup) {
	defer wg.Done() // Indique que cette routine est terminée une fois la fonction terminée

	partialSum := new(big.Int) // Utilisation de new(big.Int) pour éviter les allocations répétées de mémoire
	for i := start; i <= end; i++ {
		fibValue, err := fibDoubling(i) // Calcul de F(i)
		if err != nil {
			log.Printf("Erreur lors du calcul de Fibonacci pour i=%d: %v", i, err)
			continue
		}
		if fibValue != nil {
			partialSum.Add(partialSum, fibValue) // Ajoute F(i) à la somme partielle seulement si fibValue n'est pas nil
		}
	}

	// Envoie la somme partielle au canal
	partialResult <- partialSum
}

// handleFibonacci est le gestionnaire HTTP pour la requête POST de calcul de Fibonacci
func handleFibonacci(w http.ResponseWriter, r *http.Request) {
	// Vérifie que la méthode est POST
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Décodage du corps de la requête JSON
	var request struct {
		N int `json:"n"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Erreur de décodage JSON", http.StatusBadRequest)
		return
	}

	n := request.N
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

	executionTime := time.Since(startTime).Seconds()                  // Calcule le temps total d'exécution en secondes
	avgTimePerCalculation := executionTime / float64(numCalculations) // Calcule le temps moyen par calcul en secondes

	// Prépare la réponse JSON
	response := struct {
		Sum                   string  `json:"sum"`
		NumCalculations       int     `json:"num_calculations"`
		AvgTimePerCalculation float64 `json:"avg_time_per_calculation_in_second"`
		ExecutionTime         float64 `json:"execution_time_in_second"`
	}{
		Sum:                   sumFib.String(),
		NumCalculations:       numCalculations,
		AvgTimePerCalculation: avgTimePerCalculation,
		ExecutionTime:         executionTime,
	}

	// Encodage de la réponse en JSON et écriture dans la réponse HTTP
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Erreur d'encodage JSON", http.StatusInternalServerError)
		return
	}
}

// Fonction principale qui démarre le serveur HTTP
func main() {
	http.HandleFunc("/fibonacci", handleFibonacci)  // Associe la fonction handleFibonacci au chemin /fibonacci
	log.Println("Serveur démarré sur le port 8080") // Affiche un message pour indiquer que le serveur a démarré
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur : %v", err) // Affiche une erreur fatale si le serveur ne démarre pas
	}
}
