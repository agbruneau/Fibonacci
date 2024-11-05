// Ce programme calcule la somme des n premiers nombres de Fibonacci de manière parallélisée.
// Il utilise des techniques avancées de concurrence en Go et gère les grands nombres.

package main

import (
	"context"  // Pour la gestion du contexte et des timeouts
	"fmt"      // Pour l'affichage formaté
	"log"      // Pour la journalisation des erreurs
	"math/big" // Pour gérer les grands nombres entiers

	// Pour les opérations systèmes
	"runtime" // Pour obtenir des informations sur l'environnement d'exécution
	"strings" // Pour la manipulation de chaînes
	"sync"    // Pour la synchronisation des goroutines
	"time"    // Pour les mesures de temps et timeouts

	"github.com/pkg/errors"
)

// Configuration centralise tous les paramètres configurables du programme.
// Cette structure permet de modifier facilement les paramètres sans changer le code.
type Configuration struct {
	M           int           // M définit la limite supérieure (exclu) du calcul
	NumWorkers  int           // Nombre de workers parallèles
	SegmentSize int           // Taille des segments de calcul pour chaque worker
	Timeout     time.Duration // Durée maximale autorisée pour le calcul complet
}

// DefaultConfig retourne une configuration par défaut avec des valeurs raisonnables.
// Ces valeurs peuvent être ajustées selon les besoins et les ressources disponibles.
func DefaultConfig() Configuration {
	return Configuration{
		M:           100000,           // Calcul jusqu'à F(99999)
		NumWorkers:  runtime.NumCPU(), // Utilise tous les cœurs CPU disponibles
		SegmentSize: 1000,             // Chaque worker traite 1000 nombres à la fois
		Timeout:     5 * time.Minute,  // Le calcul s'arrête après 5 minutes
	}
}

// Metrics garde trace des métriques de performance pendant l'exécution.
// Cette structure est thread-safe grâce à son mutex intégré.
type Metrics struct {
	StartTime         time.Time  // Heure de début du calcul
	EndTime           time.Time  // Heure de fin du calcul
	TotalCalculations int64      // Nombre total de calculs effectués
	mutex             sync.Mutex // Mutex pour protéger les modifications concurrentes
}

// NewMetrics crée une nouvelle instance de Metrics initialisée avec l'heure actuelle.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// IncrementCalculations incrémente le compteur de calculs de manière thread-safe.
// Cette méthode est appelée par les workers pour suivre leur progression.
func (m *Metrics) IncrementCalculations(count int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.TotalCalculations += count
}

// FibCalculator encapsule la logique de calcul des nombres de Fibonacci.
// Il réutilise les variables big.Int pour éviter les allocations mémoire répétées.
type FibCalculator struct {
	fk, fk1             *big.Int   // Variables pour stocker F(k) et F(k+1)
	temp1, temp2, temp3 *big.Int   // Variables temporaires pour les calculs
	mutex               sync.Mutex // Protection pour l'accès concurrent
}

// NewFibCalculator crée une nouvelle instance de calculateur avec ses variables initialisées.
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{
		fk:    new(big.Int),
		fk1:   new(big.Int),
		temp1: new(big.Int),
		temp2: new(big.Int),
		temp3: new(big.Int),
	}
}

// Calculate calcule le n-ième nombre de Fibonacci en utilisant l'algorithme de doublement.
// Cet algorithme est beaucoup plus efficace que l'approche récursive classique.
// La complexité est O(log n) au lieu de O(n) ou O(2^n).
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	// Validation des entrées
	if n < 0 {
		return nil, errors.New("n doit être non-négatif")
	}
	if n > 1000000 {
		return nil, errors.New("n est trop grand, risque de calculs extrêmement coûteux")
	}

	// Protection contre les accès concurrents
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Cas de base pour F(0) et F(1)
	if n <= 1 {
		return big.NewInt(int64(n)), nil
	}

	// Initialisation pour l'algorithme de doublement
	fc.fk.SetInt64(0)  // F(0)
	fc.fk1.SetInt64(1) // F(1)

	// L'algorithme de doublement utilise les formules :
	// F(2k) = F(k)[2F(k+1) - F(k)]
	// F(2k+1) = F(k+1)² + F(k)²
	for i := 63; i >= 0; i-- {
		// Sauvegarde des valeurs actuelles
		fc.temp1.Set(fc.fk)
		fc.temp2.Set(fc.fk1)

		// Calcul de F(2k)
		fc.temp3.Mul(fc.temp2, big.NewInt(2))
		fc.temp3.Sub(fc.temp3, fc.temp1)
		fc.fk.Mul(fc.temp1, fc.temp3)

		// Calcul de F(2k+1)
		fc.fk1.Mul(fc.temp2, fc.temp2)
		fc.temp3.Mul(fc.temp1, fc.temp1)
		fc.fk1.Add(fc.fk1, fc.temp3)

		// Si le i-ème bit de n est 1, on effectue un pas supplémentaire
		if (n & (1 << uint(i))) != 0 {
			fc.temp3.Set(fc.fk1)
			fc.fk1.Add(fc.fk1, fc.fk)
			fc.fk.Set(fc.temp3)
		}
	}

	// Retourne une copie du résultat
	return new(big.Int).Set(fc.fk), nil
}

// WorkerPool gère un pool de calculateurs réutilisables.
// Cela évite de créer/détruire des calculateurs pour chaque calcul.
type WorkerPool struct {
	calculators []*FibCalculator // Tableau des calculateurs disponibles
	current     int              // Index du prochain calculateur à utiliser
	mutex       sync.Mutex       // Protection pour l'accès concurrent
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

// GetCalculator retourne le prochain calculateur disponible de manière circulaire.
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// Result encapsule le résultat d'un calcul avec une potentielle erreur.
type Result struct {
	Value *big.Int // Résultat du calcul
	Error error    // Erreur éventuelle
}

// computeSegment calcule la somme des nombres de Fibonacci pour un segment donné.
// Cette fonction est exécutée en parallèle par plusieurs goroutines.
func computeSegment(ctx context.Context, start, end int, pool *WorkerPool, metrics *Metrics) Result {
	calc := pool.GetCalculator()
	partialSum := new(big.Int)
	segmentSize := end - start + 1

	for i := start; i <= end; i++ {
		// Vérifie si le contexte a expiré (timeout)
		select {
		case <-ctx.Done():
			return Result{Error: ctx.Err()}
		default:
			// Calcule F(i) et l'ajoute à la somme partielle
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
// Par exemple : 123456789 devient "1.2345e8"
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
	// Initialisation
	config := DefaultConfig()
	metrics := NewMetrics()

	// n est la limite supérieure (exclue) pour le calcul
	n := config.M - 1

	// Création d'un contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Initialisation du pool de workers et des canaux
	pool := NewWorkerPool(config.NumWorkers)
	results := make(chan Result, config.NumWorkers)
	var wg sync.WaitGroup

	// Distribution du travail aux workers
	for start := 0; start < n; start += config.SegmentSize {
		end := start + config.SegmentSize - 1
		if end >= n {
			end = n - 1
		}

		// Lancement d'une goroutine pour chaque segment
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			result := computeSegment(ctx, start, end, pool, metrics)
			results <- result
		}(start, end)
	}

	// Goroutine pour fermer le canal results quand tout est terminé
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collecte et agrégation des résultats
	sumFib := new(big.Int)
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
}
