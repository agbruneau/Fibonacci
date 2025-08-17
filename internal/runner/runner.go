// Package runner orchestre l'exécution concurrente des algorithmes de benchmark.
package runner

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/user/fibbench/internal/fibonacci"
	"github.com/user/fibbench/internal/metrics"
	"golang.org/x/sync/errgroup"
)

const progressRefreshRate = 100 * time.Millisecond

// Result encapsule le résultat d'une exécution d'algorithme.
type Result struct {
	Name     string
	Key      fibonacci.AlgorithmKey
	Value    *big.Int
	Duration time.Duration
	Err      error
}

// progressData est utilisé pour communiquer la progression.
type progressData struct {
	name string
	pct  float64
}

// Runner est responsable de la configuration et de l'exécution du benchmark.
type Runner struct {
	N       int
	Timeout time.Duration
	Algos   []fibonacci.Algorithm
	intPool *sync.Pool
}

// NewRunner crée une nouvelle instance de Runner configurée.
func NewRunner(n int, timeout time.Duration, algoKeys []fibonacci.AlgorithmKey) (*Runner, error) {
	var algos []fibonacci.Algorithm
	for _, key := range algoKeys {
		algo, err := fibonacci.Get(key)
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la récupération de l'algorithme %s: %w", key, err)
		}
		algos = append(algos, algo)
	}

	return &Runner{
		N:       n,
		Timeout: timeout,
		Algos:   algos,
		intPool: fibonacci.NewIntPool(),
	}, nil
}

// Run exécute le benchmark de manière concurrente en utilisant errgroup.
func (r *Runner) Run(ctx context.Context) ([]Result, error) {
	slog.Info("Démarrage du benchmark", "N", r.N, "timeout", r.Timeout, "algorithms_count", len(r.Algos))

	// Configuration du contexte avec timeout.
	runCtx := ctx
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}

	// Utilisation de errgroup pour gérer le cycle de vie des goroutines et l'annulation.
	g, gCtx := errgroup.WithContext(runCtx)

	resultsCh := make(chan Result, len(r.Algos))
	progressCh := make(chan progressData, len(r.Algos)*50) // Tamponné

	// 1. Lancement du présentateur de progression.
	g.Go(func() error {
		// S'arrête lorsque progressCh est fermé ou que gCtx est annulé.
		r.runPresenter(gCtx, progressCh)
		return nil
	})

	// 2. Lancement des workers (Fan-out).
	// Utilisation d'un WaitGroup pour les workers afin de contrôler la fermeture de progressCh.
	var workersWg sync.WaitGroup
	for _, algo := range r.Algos {
		algo := algo
		workersWg.Add(1)
		g.Go(func() error {
			defer workersWg.Done()
			result := r.runTask(gCtx, algo, progressCh)
			select {
			case resultsCh <- result:
				return nil
			case <-gCtx.Done():
				return gCtx.Err()
			}
		})
	}

	// 3. Goroutine de clôture pour fermer progressCh lorsque tous les workers ont fini.
	go func() {
		workersWg.Wait()
		close(progressCh)
	}()

	// 4. Attente de la fin de toutes les goroutines (workers + présentateur).
	err := g.Wait()
	close(resultsCh)

	// 5. Collecte et tri des résultats.
	results := r.collectResults(resultsCh)

	if err != nil {
		// Journalisation des erreurs globales (timeout/annulation).
		slog.Info("Le benchmark s'est terminé avec une interruption", "error", err)
		return results, err
	}

	slog.Info("Benchmark terminé avec succès")
	return results, nil
}

// runTask exécute un algorithme spécifique, gère la progression locale et l'instrumentation.
func (r *Runner) runTask(ctx context.Context, algo fibonacci.Algorithm, progressAggregatorCh chan<- progressData) Result {
	slog.Debug("Lancement de la tâche", "algorithm", algo.Name)

	// Instrumentation (métriques).
	recordMetrics := metrics.RecordStart(string(algo.Key))

	// Canal de progression local.
	localProgCh := make(chan float64, 10)

	// Goroutine de relais pour envoyer la progression sans bloquer le calcul.
	var relayWg sync.WaitGroup
	relayWg.Add(1)
	go func() {
		defer relayWg.Done()
		for p := range localProgCh {
			data := progressData{name: algo.Name, pct: p}
			select {
			case progressAggregatorCh <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Exécution du calcul.
	start := time.Now()
	v, err := algo.Impl.Calculate(ctx, localProgCh, r.N, r.intPool)
	duration := time.Since(start)

	// Synchronisation finale.
	close(localProgCh)
	relayWg.Wait()

	// Enregistrement des métriques.
	recordMetrics(err)

	slog.Debug("Tâche terminée", "algorithm", algo.Name, "duration", duration, "error", err)

	return Result{
		Name:     algo.Name,
		Key:      algo.Key,
		Value:    v,
		Duration: duration,
		Err:      err,
	}
}

// collectResults lit tous les résultats du canal et les trie par performance.
func (r *Runner) collectResults(resultsCh <-chan Result) []Result {
	results := make([]Result, 0, len(r.Algos))
	for result := range resultsCh {
		results = append(results, result)
		if result.Err != nil {
			r.logTaskError(result)
		}
	}

	// Tri : succès en premier, puis par durée croissante.
	sort.Slice(results, func(i, j int) bool {
		if results[i].Err == nil && results[j].Err != nil {
			return true
		}
		if results[i].Err != nil && results[j].Err == nil {
			return false
		}
		return results[i].Duration < results[j].Duration
	})
	return results
}

// logTaskError journalise les erreurs spécifiques aux tâches.
func (r *Runner) logTaskError(result Result) {
	durationStr := result.Duration.Round(time.Microsecond).String()

	// Les erreurs de contexte sont attendues et journalisées en WARN.
	if result.Err == context.DeadlineExceeded || result.Err == context.Canceled {
		slog.Warn("Tâche annulée ou délai dépassé",
			"algorithm", result.Name,
			"duration", durationStr,
			"reason", result.Err)
	} else {
		// Les autres erreurs sont journalisées en ERROR.
		slog.Error("Erreur lors de l'exécution de la tâche",
			"algorithm", result.Name,
			"duration", durationStr,
			"error", result.Err)
	}
}
