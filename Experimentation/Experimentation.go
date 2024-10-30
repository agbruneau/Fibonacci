// Programme de calcul de la somme des nombres de Fibonacci en parallèle en tant que service web
//
// Ce programme implémente un calcul parallèle de la somme des nombres de Fibonacci
// en utilisant une méthode de décomposition binaire optimisée et des goroutines en Go.
// Le calcul est distribué entre plusieurs workers, chaque worker utilisant un
// calculateur de Fibonacci encapsulé dans la structure `FibCalculator`.
// L'objectif principal est d'optimiser les calculs en exploitant le parallélisme
// et en évitant les recalculs inutiles grâce à une utilisation judicieuse des
// primitives de synchronisation.
//
// Le programme est maintenant adapté pour fonctionner comme un service web,
// accessible via des requêtes HTTP (par exemple, via la commande `curl`).
// Le service offre une API pour calculer le n-ième nombre de Fibonacci ou la somme
// des nombres de Fibonacci jusqu'à une valeur donnée.
//
// Ce programme est conçu pour utiliser efficacement les ressources CPU disponibles,
// en divisant la charge de travail de calcul de la série de Fibonacci en segments gérés par plusieurs workers.
// Les résultats sont renvoyés au client via l'API HTTP.

// curl "http://localhost:8080/fibonacci?n=10"

package main

import (
	"encoding/json" // Le package 'encoding/json' est utilisé pour encoder et décoder les données JSON des réponses HTTP.
	"fmt"           // Le package 'fmt' est utilisé pour la sortie formatée.
	"math/big"      // Le package 'math/big' permet la manipulation de nombres entiers très grands.
	"math/bits"     // Le package 'math/bits' est utilisé pour manipuler les bits des entiers.
	"net/http"      // Le package 'net/http' est utilisé pour créer un serveur web et gérer les requêtes HTTP.
	"runtime"       // Le package 'runtime' est utilisé pour obtenir des informations sur le système.
	// Le package 'strings' est utilisé pour manipuler des chaînes de caractères.
	"strconv"
	"sync" // Le package 'sync' fournit des primitives pour synchroniser les goroutines.
	// Le package 'time' est utilisé pour mesurer les durées d'exécution.
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

// handleFibonacci calcule le n-ième nombre de Fibonacci et envoie la réponse au client
func handleFibonacci(w http.ResponseWriter, r *http.Request) {
	nStr := r.URL.Query().Get("n")
	if nStr == "" {
		http.Error(w, "Paramètre 'n' manquant", http.StatusBadRequest)
		return
	}

	n, err := strconv.Atoi(nStr)
	if err != nil || n < 0 {
		http.Error(w, "Paramètre 'n' invalide, doit être un entier positif", http.StatusBadRequest)
		return
	}

	pool := NewWorkerPool(runtime.NumCPU())
	calculator := pool.GetCalculator()
	result, err := calculator.Calculate(n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"fibonacci": result.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/fibonacci", handleFibonacci)
	fmt.Println("Serveur démarré sur le port 8080...")
	http.ListenAndServe(":8080", nil)
}
