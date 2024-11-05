// Service Web pour calculer la somme des n premiers nombres de Fibonacci de manière parallélisée.
// Utilise des techniques avancées de concurrence en Go et gère les grands nombres.
//
// Pour utiliser ce service en ligne de commande avec curl :
//
// Exemple de requête :
// curl -X POST http://localhost:8080/fibonacci -H "Content-Type: application/json" -d '{"m": 1000, "numWorkers": 4, "segmentSize": 100, "timeout": "1m"}'
//
// Exemple de requête avec configuration par défaut :
// curl -X POST http://localhost:8080/fibonacci -H "Content-Type: application/json" -d '{}'
//
// Les paramètres sont tous optionnels et ont des valeurs par défaut :
// - m: nombre de termes à calculer (défaut: 100000)
// - numWorkers: nombre de workers parallèles (défaut: nombre de CPU)
// - segmentSize: taille des segments de calcul (défaut: 1000)
// - timeout: durée maximale en format Go (défaut: "5m")

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Configuration centralise tous les paramètres configurables du programme.
type Configuration struct {
	M           int           `json:"m"`           // M définit la limite supérieure (exclu) du calcul
	NumWorkers  int           `json:"numWorkers"`  // Nombre de workers parallèles
	SegmentSize int           `json:"segmentSize"` // Taille des segments de calcul pour chaque worker
	Timeout     time.Duration `json:"timeout"`     // Durée maximale autorisée pour le calcul complet
}

// APIRequest représente la structure de la requête JSON
type APIRequest struct {
	M           *int   `json:"m,omitempty"`           // Nombre de termes à calculer (optionnel)
	NumWorkers  *int   `json:"numWorkers,omitempty"`  // Nombre de workers parallèles (optionnel)
	SegmentSize *int   `json:"segmentSize,omitempty"` // Taille des segments (optionnel)
	Timeout     string `json:"timeout,omitempty"`     // Durée maximale sous forme de chaîne (optionnel)
}

// APIResponse représente la structure de la réponse JSON
type APIResponse struct {
	Result     string        `json:"result"`          // Résultat du calcul en notation scientifique
	Duration   time.Duration `json:"duration"`        // Durée totale du calcul
	Calculs    int64         `json:"calculations"`    // Nombre total de calculs effectués
	TempsMoyen time.Duration `json:"averageTime"`     // Temps moyen par calcul
	Error      string        `json:"error,omitempty"` // Message d'erreur (le cas échéant)
}

// DefaultConfig retourne une configuration par défaut avec des valeurs raisonnables.
func DefaultConfig() Configuration {
	return Configuration{
		M:           100000,           // Nombre par défaut de termes de Fibonacci à calculer
		NumWorkers:  runtime.NumCPU(), // Nombre de workers par défaut égal au nombre de CPU
		SegmentSize: 1000,             // Taille de segment par défaut
		Timeout:     5 * time.Minute,  // Délai d'attente par défaut de 5 minutes
	}
}

// Metrics garde trace des métriques de performance pendant l'exécution.
type Metrics struct {
	StartTime         time.Time  // Heure de début
	EndTime           time.Time  // Heure de fin
	TotalCalculations int64      // Nombre total de calculs effectués
	mutex             sync.Mutex // Mutex pour garantir l'accès thread-safe aux métriques
}

// NewMetrics crée une nouvelle instance de Metrics initialisée.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()} // Initialisation de l'heure de début à l'instant courant
}

// IncrementCalculations incrémente le compteur de calculs de manière thread-safe.
func (m *Metrics) IncrementCalculations(count int64) {
	m.mutex.Lock()               // Verrouiller pour éviter la concurrence
	defer m.mutex.Unlock()       // Déverrouiller après l'opération
	m.TotalCalculations += count // Incrémenter le compteur de calculs
}

// FibCalculator encapsule la logique de calcul des nombres de Fibonacci.
type FibCalculator struct {
	fk, fk1             *big.Int   // Variables pour les deux derniers nombres de Fibonacci
	temp1, temp2, temp3 *big.Int   // Variables temporaires pour le calcul
	mutex               sync.Mutex // Mutex pour garantir l'accès thread-safe au calculateur
}

// NewFibCalculator crée une nouvelle instance de calculateur.
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		fk:    new(big.Int), // Initialiser fk
		fk1:   new(big.Int), // Initialiser fk1
		temp1: new(big.Int), // Initialiser temp1
		temp2: new(big.Int), // Initialiser temp2
		temp3: new(big.Int), // Initialiser temp3
	}
}

// Calculate calcule le n-ième nombre de Fibonacci.
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, errors.New("n doit être non-négatif") // Vérifier que n est non-négatif
	}
	if n > 1000001 {
		return nil, errors.New("n est trop grand, risque de calculs extrêmement coûteux") // Limiter la valeur maximale de n
	}

	fc.mutex.Lock()         // Verrouiller pour garantir l'accès exclusif aux variables internes
	defer fc.mutex.Unlock() // Déverrouiller à la fin de l'opération

	if n <= 1 {
		return big.NewInt(int64(n)), nil // Retourner directement le résultat pour n = 0 ou n = 1
	}

	// Initialiser les deux premiers termes de la suite de Fibonacci
	fc.fk.SetInt64(0)
	fc.fk1.SetInt64(1)

	// Utiliser la méthode de doublement pour calculer rapidement le n-ième terme
	for i := 63; i >= 0; i-- {
		// Calculer les termes temporaires selon l'algorithme de doublement
		fc.temp1.Set(fc.fk)
		fc.temp2.Set(fc.fk1)

		fc.temp3.Mul(fc.temp2, big.NewInt(2)) // temp3 = 2 * fk1
		fc.temp3.Sub(fc.temp3, fc.temp1)      // temp3 = 2 * fk1 - fk
		fc.fk.Mul(fc.temp1, fc.temp3)         // fk = fk * temp3

		fc.fk1.Mul(fc.temp2, fc.temp2)   // fk1 = fk1^2
		fc.temp3.Mul(fc.temp1, fc.temp1) // temp3 = fk^2
		fc.fk1.Add(fc.fk1, fc.temp3)     // fk1 = fk1^2 + fk^2

		if (n & (1 << uint(i))) != 0 {
			// Si le bit i est 1, continuer le calcul
			fc.temp3.Set(fc.fk1)
			fc.fk1.Add(fc.fk1, fc.fk) // fk1 = fk1 + fk
			fc.fk.Set(fc.temp3)       // fk = temp3 (ancien fk1)
		}
	}

	return new(big.Int).Set(fc.fk), nil // Retourner le résultat final
}

// WorkerPool gère un pool de calculateurs réutilisables.
type WorkerPool struct {
	calculators []*FibCalculator // Liste des calculateurs disponibles
	current     int              // Index du prochain calculateur à utiliser
	mutex       sync.Mutex       // Mutex pour garantir l'accès thread-safe au pool
}

// NewWorkerPool crée un nouveau pool avec le nombre spécifié de calculateurs.
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator() // Créer un nouveau calculateur pour chaque worker
	}
	return &WorkerPool{
		calculators: calculators,
	}
}

// GetCalculator retourne le prochain calculateur disponible.
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()                                     // Verrouiller pour éviter la concurrence
	defer wp.mutex.Unlock()                             // Déverrouiller après avoir sélectionné le calculateur
	calc := wp.calculators[wp.current]                  // Récupérer le calculateur courant
	wp.current = (wp.current + 1) % len(wp.calculators) // Passer au calculateur suivant
	return calc
}

// Result encapsule le résultat d'un calcul avec une potentielle erreur.
type Result struct {
	Value *big.Int // Valeur calculée
	Error error    // Erreur potentielle
}

// computeSegment calcule la somme des nombres de Fibonacci pour un segment.
func computeSegment(ctx context.Context, start, end int, pool *WorkerPool, metrics *Metrics) Result {
	calc := pool.GetCalculator()   // Obtenir un calculateur du pool
	partialSum := new(big.Int)     // Initialiser la somme partielle
	segmentSize := end - start + 1 // Taille du segment à calculer

	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done():
			// Si le contexte est annulé, retourner une erreur
			return Result{Error: ctx.Err()}
		default:
			// Calculer le n-ième terme de Fibonacci
			fibValue, err := calc.Calculate(i)
			if err != nil {
				return Result{Error: errors.Wrapf(err, "computing Fibonacci(%d)", i)}
			}
			partialSum.Add(partialSum, fibValue) // Ajouter le terme à la somme partielle
		}
	}

	metrics.IncrementCalculations(int64(segmentSize)) // Mettre à jour le compteur de calculs
	return Result{Value: partialSum}                  // Retourner la somme partielle
}

// formatBigIntSci formate un grand nombre en notation scientifique.
func formatBigIntSci(n *big.Int) string {
	numStr := n.String()  // Convertir le nombre en chaîne de caractères
	numLen := len(numStr) // Longueur du nombre

	if numLen <= 5 {
		return numStr // Si la longueur est inférieure à 5, retourner directement la chaîne
	}

	significand := numStr[:5] // Les 5 premiers chiffres significatifs
	exponent := numLen - 1    // Exposant basé sur la longueur

	formattedNum := significand[:1] + "." + significand[1:]                     // Formater le nombre avec un point décimal
	formattedNum = strings.TrimRight(strings.TrimRight(formattedNum, "0"), ".") // Supprimer les zéros et points inutiles

	return fmt.Sprintf("%se%d", formattedNum, exponent) // Retourner le nombre en notation scientifique
}

// handleFibonacci gère les requêtes HTTP pour le calcul de Fibonacci
func handleFibonacci(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed) // Vérifier que la méthode est POST
		return
	}

	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Erreur de décodage JSON: "+err.Error(), http.StatusBadRequest) // Gérer les erreurs de décodage JSON
		return
	}

	config := DefaultConfig() // Charger la configuration par défaut

	// Mettre à jour la configuration avec les valeurs fournies par l'utilisateur
	if req.M != nil {
		config.M = *req.M
	}
	if req.NumWorkers != nil {
		config.NumWorkers = *req.NumWorkers
	}
	if req.SegmentSize != nil {
		config.SegmentSize = *req.SegmentSize
	}
	if req.Timeout != "" {
		timeout, err := time.ParseDuration(req.Timeout)
		if err != nil {
			http.Error(w, "Format de timeout invalide: "+err.Error(), http.StatusBadRequest) // Gérer les erreurs de format de timeout
			return
		}
		config.Timeout = timeout
	}

	metrics := NewMetrics()                                         // Initialiser les métriques
	ctx, cancel := context.WithTimeout(r.Context(), config.Timeout) // Créer un contexte avec délai d'attente
	defer cancel()

	n := config.M - 1
	pool := NewWorkerPool(config.NumWorkers)        // Créer un pool de calculateurs
	results := make(chan Result, config.NumWorkers) // Canal pour recevoir les résultats
	var wg sync.WaitGroup

	// Lancer des goroutines pour calculer les segments en parallèle
	for start := 0; start < n; start += config.SegmentSize {
		end := start + config.SegmentSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()                                          // Décrémenter le compteur quand la goroutine se termine
			result := computeSegment(ctx, start, end, pool, metrics) // Calculer le segment
			results <- result                                        // Envoyer le résultat au canal
		}(start, end)
	}

	// Attendre la fin de toutes les goroutines et fermer le canal de résultats
	go func() {
		wg.Wait()
		close(results)
	}()

	sumFib := new(big.Int) // Initialiser la somme totale des termes de Fibonacci
	var calcError error

	// Lire les résultats des goroutines
	for result := range results {
		if result.Error != nil {
			calcError = result.Error
			break
		}
		sumFib.Add(sumFib, result.Value) // Ajouter la valeur partielle à la somme totale
	}

	metrics.EndTime = time.Now()                                   // Enregistrer l'heure de fin
	duration := metrics.EndTime.Sub(metrics.StartTime)             // Calculer la durée totale du calcul
	avgTime := duration / time.Duration(metrics.TotalCalculations) // Calculer le temps moyen par calcul

	// Construire la réponse API
	response := APIResponse{
		Duration:   duration,
		Calculs:    metrics.TotalCalculations,
		TempsMoyen: avgTime,
	}

	if calcError != nil {
		response.Error = calcError.Error() // Enregistrer l'erreur si une erreur est survenue
	} else {
		response.Result = formatBigIntSci(sumFib) // Formater le résultat final
	}

	w.Header().Set("Content-Type", "application/json") // Définir le type de contenu de la réponse
	if calcError != nil {
		w.WriteHeader(http.StatusInternalServerError) // Si une erreur est survenue, retourner un code d'erreur HTTP
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Erreur d'encodage de la réponse: %v", err) // Enregistrer toute erreur survenue lors de l'encodage de la réponse
	}
}

func main() {
	http.HandleFunc("/fibonacci", handleFibonacci) // Associer la route /fibonacci au gestionnaire

	port := ":8080"
	fmt.Printf("Serveur démarré sur le port %s\n", port) // Afficher un message pour indiquer que le serveur est démarré
	log.Fatal(http.ListenAndServe(port, nil))            // Lancer le serveur HTTP sur le port 8080
}
