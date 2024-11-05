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
	M           *int   `json:"m,omitempty"`
	NumWorkers  *int   `json:"numWorkers,omitempty"`
	SegmentSize *int   `json:"segmentSize,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
}

// APIResponse représente la structure de la réponse JSON
type APIResponse struct {
	Result     string        `json:"result"`
	Duration   time.Duration `json:"duration"`
	Calculs    int64         `json:"calculations"`
	TempsMoyen time.Duration `json:"averageTime"`
	Error      string        `json:"error,omitempty"`
}

// DefaultConfig retourne une configuration par défaut avec des valeurs raisonnables.
func DefaultConfig() Configuration {
	return Configuration{
		M:           100000,
		NumWorkers:  runtime.NumCPU(),
		SegmentSize: 1000,
		Timeout:     5 * time.Minute,
	}
}

// Metrics garde trace des métriques de performance pendant l'exécution.
type Metrics struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalCalculations int64
	mutex             sync.Mutex
}

// NewMetrics crée une nouvelle instance de Metrics initialisée.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// IncrementCalculations incrémente le compteur de calculs de manière thread-safe.
func (m *Metrics) IncrementCalculations(count int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.TotalCalculations += count
}

// FibCalculator encapsule la logique de calcul des nombres de Fibonacci.
type FibCalculator struct {
	fk, fk1             *big.Int
	temp1, temp2, temp3 *big.Int
	mutex               sync.Mutex
}

// NewFibCalculator crée une nouvelle instance de calculateur.
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		fk:    new(big.Int),
		fk1:   new(big.Int),
		temp1: new(big.Int),
		temp2: new(big.Int),
		temp3: new(big.Int),
	}
}

// Calculate calcule le n-ième nombre de Fibonacci.
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, errors.New("n doit être non-négatif")
	}
	if n > 1000000 {
		return nil, errors.New("n est trop grand, risque de calculs extrêmement coûteux")
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if n <= 1 {
		return big.NewInt(int64(n)), nil
	}

	fc.fk.SetInt64(0)
	fc.fk1.SetInt64(1)

	for i := 63; i >= 0; i-- {
		fc.temp1.Set(fc.fk)
		fc.temp2.Set(fc.fk1)

		fc.temp3.Mul(fc.temp2, big.NewInt(2))
		fc.temp3.Sub(fc.temp3, fc.temp1)
		fc.fk.Mul(fc.temp1, fc.temp3)

		fc.fk1.Mul(fc.temp2, fc.temp2)
		fc.temp3.Mul(fc.temp1, fc.temp1)
		fc.fk1.Add(fc.fk1, fc.temp3)

		if (n & (1 << uint(i))) != 0 {
			fc.temp3.Set(fc.fk1)
			fc.fk1.Add(fc.fk1, fc.fk)
			fc.fk.Set(fc.temp3)
		}
	}

	return new(big.Int).Set(fc.fk), nil
}

// WorkerPool gère un pool de calculateurs réutilisables.
type WorkerPool struct {
	calculators []*FibCalculator
	current     int
	mutex       sync.Mutex
}

// NewWorkerPool crée un nouveau pool avec le nombre spécifié de calculateurs.
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
	}
}

// GetCalculator retourne le prochain calculateur disponible.
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// Result encapsule le résultat d'un calcul avec une potentielle erreur.
type Result struct {
	Value *big.Int
	Error error
}

// computeSegment calcule la somme des nombres de Fibonacci pour un segment.
func computeSegment(ctx context.Context, start, end int, pool *WorkerPool, metrics *Metrics) Result {
	calc := pool.GetCalculator()
	partialSum := new(big.Int)
	segmentSize := end - start + 1

	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done():
			return Result{Error: ctx.Err()}
		default:
			fibValue, err := calc.Calculate(i)
			if err != nil {
				return Result{Error: errors.Wrapf(err, "computing Fibonacci(%d)", i)}
			}
			partialSum.Add(partialSum, fibValue)
		}
	}

	metrics.IncrementCalculations(int64(segmentSize))
	return Result{Value: partialSum}
}

// formatBigIntSci formate un grand nombre en notation scientifique.
func formatBigIntSci(n *big.Int) string {
	numStr := n.String()
	numLen := len(numStr)

	if numLen <= 5 {
		return numStr
	}

	significand := numStr[:5]
	exponent := numLen - 1

	formattedNum := significand[:1] + "." + significand[1:]
	formattedNum = strings.TrimRight(strings.TrimRight(formattedNum, "0"), ".")

	return fmt.Sprintf("%se%d", formattedNum, exponent)
}

// handleFibonacci gère les requêtes HTTP pour le calcul de Fibonacci
func handleFibonacci(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Erreur de décodage JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	config := DefaultConfig()

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
			http.Error(w, "Format de timeout invalide: "+err.Error(), http.StatusBadRequest)
			return
		}
		config.Timeout = timeout
	}

	metrics := NewMetrics()
	ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
	defer cancel()

	n := config.M - 1
	pool := NewWorkerPool(config.NumWorkers)
	results := make(chan Result, config.NumWorkers)
	var wg sync.WaitGroup

	for start := 0; start < n; start += config.SegmentSize {
		end := start + config.SegmentSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			result := computeSegment(ctx, start, end, pool, metrics)
			results <- result
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	sumFib := new(big.Int)
	var calcError error

	for result := range results {
		if result.Error != nil {
			calcError = result.Error
			break
		}
		sumFib.Add(sumFib, result.Value)
	}

	metrics.EndTime = time.Now()
	duration := metrics.EndTime.Sub(metrics.StartTime)
	avgTime := duration / time.Duration(metrics.TotalCalculations)

	response := APIResponse{
		Duration:   duration,
		Calculs:    metrics.TotalCalculations,
		TempsMoyen: avgTime,
	}

	if calcError != nil {
		response.Error = calcError.Error()
	} else {
		response.Result = formatBigIntSci(sumFib)
	}

	w.Header().Set("Content-Type", "application/json")
	if calcError != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Erreur d'encodage de la réponse: %v", err)
	}
}

func main() {
	http.HandleFunc("/fibonacci", handleFibonacci)

	port := ":8080"
	fmt.Printf("Serveur démarré sur le port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
