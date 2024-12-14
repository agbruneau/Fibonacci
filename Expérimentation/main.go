// go run main.go -m 200000 -workers 8 -segment 5000 -timeout 10m
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Configuration du programme
type Configuration struct {
	M           int           // Limite supérieure du calcul
	NumWorkers  int           // Nombre de workers
	SegmentSize int           // Taille du segment de calcul
	Timeout     time.Duration // Durée maximale du calcul
}

// Metrics de performance
type Metrics struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalCalculations int64
	mutex             sync.Mutex
}

// Créer une nouvelle instance de Metrics
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// Incrémenter le compteur de calculs
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

// Créer une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		fk:    new(big.Int),
		fk1:   new(big.Int),
		temp1: new(big.Int),
		temp2: new(big.Int),
		temp3: new(big.Int),
	}
}

// Calculate calcule le n-ième nombre de Fibonacci
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, errors.New("n doit être non-négatif")
	}
	if n > 1000001 {
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

// Pool de calculateurs
type WorkerPool struct {
	calculators []*FibCalculator
	current     int
	mutex       sync.Mutex
}

// Créer un nouveau pool de worker
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
	}
}

// Retourner le prochain calculateur disponible
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// Resultat d'un calcul
type Result struct {
	Value *big.Int
	Error error
}

// Pool d'allocation de big.Int pour réduire le nombre d'allocation
var bigIntPool = sync.Pool{
	New: func() interface{} {
		return new(big.Int)
	},
}

// Obtenir un big.Int depuis le pool
func getBigIntFromPool() *big.Int {
	return bigIntPool.Get().(*big.Int)
}

// Remettre un big.Int au pool
func putBigIntToPool(bi *big.Int) {
	bi.SetInt64(0)
	bigIntPool.Put(bi)
}

// computeSegment calcule la somme des nombres de Fibonacci pour un segment donné.
func computeSegment(ctx context.Context, start, end int, pool *WorkerPool, metrics *Metrics, results chan<- Result) {
	calc := pool.GetCalculator()
	partialSum := getBigIntFromPool()
	defer putBigIntToPool(partialSum)

	segmentSize := end - start + 1
	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done():
			results <- Result{Error: ctx.Err()}
			return
		default:
			fibValue, err := calc.Calculate(i)
			if err != nil {
				results <- Result{Error: errors.Wrapf(err, "computing Fibonacci(%d)", i)}
				return
			}
			partialSum.Add(partialSum, fibValue)
		}
	}
	metrics.IncrementCalculations(int64(segmentSize))
	results <- Result{Value: partialSum}

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

// La fonction main orchestre tout le processus de calcul.
func main() {
	// Configuration du programme par flags
	config := Configuration{}
	flag.IntVar(&config.M, "m", 100000, "Limite supérieure du calcul")
	flag.IntVar(&config.NumWorkers, "workers", runtime.NumCPU(), "Nombre de workers")
	flag.IntVar(&config.SegmentSize, "segment", 1000, "Taille des segments de calcul")
	flag.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Durée maximale du calcul")
	flag.Parse()

	//Initialisation des metrics
	metrics := NewMetrics()
	// n est la limite supérieure (exclue) pour le calcul
	n := config.M - 1

	// Initialisation du context avec timeout et du canal d'annulation
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Initialisation du pool de workers et des canaux
	pool := NewWorkerPool(config.NumWorkers)
	results := make(chan Result, config.NumWorkers)
	tasks := make(chan struct{ start, end int }, config.NumWorkers)
	var wg sync.WaitGroup

	//Lancement des workers
	for i := 0; i < config.NumWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				computeSegment(ctx, task.start, task.end, pool, metrics, results)
			}
		}()
	}
	// Distribution du travail aux workers
	for start := 0; start < n; start += config.SegmentSize {
		end := start + config.SegmentSize - 1
		if end >= n {
			end = n - 1
		}
		tasks <- struct{ start, end int }{start: start, end: end}
	}
	close(tasks)

	// Goroutine pour fermer le canal results quand tout est terminé
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collecte et agrégation des résultats
	sumFib := getBigIntFromPool()
	defer putBigIntToPool(sumFib)
	hasErrors := false

	for result := range results {
		if result.Error != nil {
			log.Printf("Erreur durant le calcul: %v", result.Error)
			hasErrors = true
			continue
		}
		sumFib.Add(sumFib, result.Value)
	}

	if hasErrors {
		log.Printf("Des erreurs sont survenues pendant le calcul")
	}

	// Calcul des métriques finales
	metrics.EndTime = time.Now()
	duration := metrics.EndTime.Sub(metrics.StartTime)
	avgTime := duration / time.Duration(metrics.TotalCalculations)

	// Affichage des résultats
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Nombre de workers: %d\n", config.NumWorkers)
	fmt.Printf("  Taille des segments: %d\n", config.SegmentSize)
	fmt.Printf("  Valeur de m: %d\n", config.M)

	fmt.Printf("\nPerformance:\n")
	fmt.Printf("  Temps total d'exécution: %v\n", duration)
	fmt.Printf("  Nombre de calculs: %d\n", metrics.TotalCalculations)
	fmt.Printf("  Temps moyen par calcul: %v\n", avgTime)

	fmt.Printf("\nRésultat:\n")
	fmt.Printf("  Somme des Fibonacci(0..%d): %s\n", config.M, formatBigIntSci(sumFib))
	os.Exit(0)
}
