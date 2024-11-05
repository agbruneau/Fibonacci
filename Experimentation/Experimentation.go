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
	Calculs    int64        `json:"calculations"`
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

[Le reste du code reste identique jusqu'aux fonctions de calcul]

// handleFibonacci gère les requêtes HTTP pour le calcul de Fibonacci
func handleFibonacci(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Décode la requête JSON
	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Erreur de décodage JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Création de la configuration en partant des valeurs par défaut
	config := DefaultConfig()

	// Mise à jour de la configuration avec les valeurs de la requête
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

	// Initialisation des métriques et du contexte
	metrics := NewMetrics()
	ctx, cancel := context.WithTimeout(r.Context(), config.Timeout)
	defer cancel()

	// Calcul
	n := config.M - 1
	pool := NewWorkerPool(config.NumWorkers)
	results := make(chan Result, config.NumWorkers)
	var wg sync.WaitGroup

	// Distribution du travail
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

	// Collecte des résultats
	sumFib := new(big.Int)
	var calcError error

	for result := range results {
		if result.Error != nil {
			calcError = result.Error
			break
		}
		sumFib.Add(sumFib, result.Value)
	}

	// Préparation de la réponse
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

	// Envoi de la réponse
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