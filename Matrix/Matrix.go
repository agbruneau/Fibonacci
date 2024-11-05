// Package principal du programme
package main

// Import des bibliothèques nécessaires
import (
	"context"  // Pour gérer les contextes et les timeouts
	"fmt"      // Pour l'affichage formaté
	"log"      // Pour la journalisation des erreurs
	"math/big" // Pour gérer les très grands nombres
	"runtime"  // Pour obtenir des informations sur l'environnement d'exécution
	"strings"  // Pour manipuler les chaînes de caractères
	"sync"     // Pour la synchronisation des goroutines
	"time"     // Pour mesurer le temps et gérer les timeouts

	"github.com/pkg/errors" // Pour une meilleure gestion des erreurs
)

// Configuration définit tous les paramètres ajustables du programme
// Cette structure permet de centraliser et modifier facilement les paramètres
type Configuration struct {
	M           int           // Nombre maximum de termes de Fibonacci à calculer
	NumWorkers  int           // Nombre de goroutines de calcul parallèles
	SegmentSize int           // Nombre de calculs par segment pour chaque worker
	Timeout     time.Duration // Temps maximum autorisé pour l'ensemble des calculs
}

// DefaultConfig retourne une configuration par défaut avec des valeurs optimisées
func DefaultConfig() Configuration {
	return Configuration{
		M:           100000,           // Calcule jusqu'à F(99999)
		NumWorkers:  runtime.NumCPU(), // Utilise tous les processeurs disponibles
		SegmentSize: 1000,             // Chaque worker traite 1000 nombres à la fois
		Timeout:     5 * time.Minute,  // Arrête le calcul après 5 minutes
	}
}

// Metrics permet de suivre les performances du programme
// Cette structure est thread-safe (utilisable en parallèle) grâce au mutex
type Metrics struct {
	StartTime         time.Time  // Moment où le calcul commence
	EndTime           time.Time  // Moment où le calcul se termine
	TotalCalculations int64      // Nombre total de nombres de Fibonacci calculés
	mutex             sync.Mutex // Verrou pour protéger les modifications concurrentes
}

// NewMetrics crée et initialise un nouveau compteur de métriques
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// IncrementCalculations augmente le compteur de calculs de façon thread-safe
func (m *Metrics) IncrementCalculations(count int64) {
	m.mutex.Lock()         // Verrouille l'accès aux données
	defer m.mutex.Unlock() // Déverrouille à la fin de la fonction
	m.TotalCalculations += count
}

// Matrix2x2 représente une matrice 2x2 utilisée pour le calcul de Fibonacci
// La méthode matricielle utilise la propriété que:
// [1 1]^n = [F(n+1) F(n)  ]
// [1 0]    [F(n)   F(n-1)]
type Matrix2x2 struct {
	a11, a12, a21, a22 *big.Int // Les 4 éléments de la matrice
}

// NewMatrix2x2 crée une nouvelle matrice 2x2 avec des éléments big.Int
func NewMatrix2x2() *Matrix2x2 {
	return &Matrix2x2{
		a11: new(big.Int), // Élément en position (1,1)
		a12: new(big.Int), // Élément en position (1,2)
		a21: new(big.Int), // Élément en position (2,1)
		a22: new(big.Int), // Élément en position (2,2)
	}
}

// FibCalculator contient tout le nécessaire pour calculer les nombres de Fibonacci
type FibCalculator struct {
	result     *big.Int   // Stocke le résultat du calcul
	baseMatrix *Matrix2x2 // Matrice de base [1 1; 1 0]
	tempMatrix *Matrix2x2 // Matrice temporaire pour les calculs
	powMatrix  *Matrix2x2 // Matrice résultat de l'exponentiation
	mutex      sync.Mutex // Protection pour l'accès concurrent
}

// NewFibCalculator initialise un nouveau calculateur de Fibonacci
func NewFibCalculator() *FibCalculator {
	fc := &FibCalculator{
		result:     new(big.Int),
		baseMatrix: NewMatrix2x2(),
		tempMatrix: NewMatrix2x2(),
		powMatrix:  NewMatrix2x2(),
	}

	// Initialise la matrice de base [[1,1],[1,0]]
	// Cette matrice est la clé de la méthode matricielle
	fc.baseMatrix.a11.SetInt64(1)
	fc.baseMatrix.a12.SetInt64(1)
	fc.baseMatrix.a21.SetInt64(1)
	fc.baseMatrix.a22.SetInt64(0)

	return fc
}

// multiplyMatrices multiplie deux matrices 2x2
// Le résultat est stocké dans la matrice result
func (fc *FibCalculator) multiplyMatrices(m1, m2, result *Matrix2x2) {
	temp1 := new(big.Int) // Variables temporaires pour
	temp2 := new(big.Int) // éviter les allocations répétées

	// Calcul de chaque élément de la matrice résultante
	// selon les règles de multiplication matricielle

	// Calcul de result[1,1] = m1[1,1]*m2[1,1] + m1[1,2]*m2[2,1]
	temp1.Mul(m1.a11, m2.a11)
	temp2.Mul(m1.a12, m2.a21)
	result.a11.Add(temp1, temp2)

	// Calcul de result[1,2] = m1[1,1]*m2[1,2] + m1[1,2]*m2[2,2]
	temp1.Mul(m1.a11, m2.a12)
	temp2.Mul(m1.a12, m2.a22)
	result.a12.Add(temp1, temp2)

	// Calcul de result[2,1] = m1[2,1]*m2[1,1] + m1[2,2]*m2[2,1]
	temp1.Mul(m1.a21, m2.a11)
	temp2.Mul(m1.a22, m2.a21)
	result.a21.Add(temp1, temp2)

	// Calcul de result[2,2] = m1[2,1]*m2[1,2] + m1[2,2]*m2[2,2]
	temp1.Mul(m1.a21, m2.a12)
	temp2.Mul(m1.a22, m2.a22)
	result.a22.Add(temp1, temp2)
}

// matrixPower calcule la puissance n-ième de la matrice de base
// Utilise l'algorithme d'exponentiation rapide (complexity O(log n))
func (fc *FibCalculator) matrixPower(n int) {
	// Initialise la matrice résultat à la matrice identité
	fc.powMatrix.a11.SetInt64(1)
	fc.powMatrix.a12.SetInt64(0)
	fc.powMatrix.a21.SetInt64(0)
	fc.powMatrix.a22.SetInt64(1)

	// Crée une copie de la matrice de base pour les calculs
	base := NewMatrix2x2()
	base.a11.Set(fc.baseMatrix.a11)
	base.a12.Set(fc.baseMatrix.a12)
	base.a21.Set(fc.baseMatrix.a21)
	base.a22.Set(fc.baseMatrix.a22)

	// Algorithme d'exponentiation rapide
	// Au lieu de multiplier n fois, on utilise la décomposition binaire de n
	// Par exemple, pour n=13 (1101 en binaire), on calcule:
	// M^13 = M^8 * M^4 * M^1
	for n > 0 {
		if n&1 == 1 { // Si le bit actuel est 1
			// Multiplie le résultat par la puissance actuelle
			fc.multiplyMatrices(fc.powMatrix, base, fc.tempMatrix)
			fc.powMatrix, fc.tempMatrix = fc.tempMatrix, fc.powMatrix
		}
		// Carré la puissance actuelle
		fc.multiplyMatrices(base, base, fc.tempMatrix)
		base, fc.tempMatrix = fc.tempMatrix, base
		n >>= 1 // Passe au bit suivant
	}
}

// Calculate calcule le n-ième nombre de Fibonacci
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	// Vérifie que n est valide
	if n < 0 {
		return nil, errors.New("n doit être non-négatif")
	}
	if n > 1000001 {
		return nil, errors.New("n est trop grand, risque de calculs extrêmement coûteux")
	}

	// Protection contre les accès concurrents
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Cas de base pour F(0) et F(1)
	if n <= 1 {
		return big.NewInt(int64(n)), nil
	}

	// Utilise l'exponentiation matricielle pour n > 1
	fc.matrixPower(n - 1)

	// F(n) est l'élément [1,1] de la matrice résultante
	return new(big.Int).Set(fc.powMatrix.a11), nil
}

// WorkerPool gère un ensemble de calculateurs réutilisables
type WorkerPool struct {
	calculators []*FibCalculator // Tableau des calculateurs disponibles
	current     int              // Index du prochain calculateur à utiliser
	mutex       sync.Mutex       // Protection pour l'accès concurrent
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

// GetCalculator retourne le prochain calculateur disponible
// de manière circulaire (round-robin)
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	calc := wp.calculators[wp.current]
	wp.current = (wp.current + 1) % len(wp.calculators)
	return calc
}

// Result encapsule le résultat d'un calcul avec une potentielle erreur
type Result struct {
	Value *big.Int // Le résultat du calcul
	Error error    // L'erreur éventuelle
}

// computeSegment calcule la somme des nombres de Fibonacci pour un segment donné
func computeSegment(ctx context.Context, start, end int, pool *WorkerPool, metrics *Metrics) Result {
	calc := pool.GetCalculator() // Obtient un calculateur du pool
	partialSum := new(big.Int)   // Pour stocker la somme partielle
	segmentSize := end - start + 1

	// Calcule chaque nombre de Fibonacci dans le segment
	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done(): // Vérifie si le timeout est atteint
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

// formatBigIntSci formate un grand nombre en notation scientifique
func formatBigIntSci(n *big.Int) string {
	numStr := n.String()
	numLen := len(numStr)

	if numLen <= 5 { // Pour les petits nombres, pas de notation scientifique
		return numStr
	}

	// Extrait les 5 premiers chiffres pour la mantisse
	significand := numStr[:5]
	exponent := numLen - 1

	// Formate en notation scientifique (ex: 1.2345e6)
	formattedNum := significand[:1] + "." + significand[1:]
	formattedNum = strings.TrimRight(strings.TrimRight(formattedNum, "0"), ".")

	return fmt.Sprintf("%se%d", formattedNum, exponent)
}

// main est le point d'entrée du programme
func main() {
	// Initialisation
	config := DefaultConfig()
	metrics := NewMetrics()
	n := config.M - 1

	// Crée un contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Initialise le pool de workers et les canaux
	pool := NewWorkerPool(config.NumWorkers)
	results := make(chan Result, config.NumWorkers)
	var wg sync.WaitGroup

	// Distribue le travail aux workers
	for start := 0; start < n; start += config.SegmentSize {
		end := start + config.SegmentSize - 1
		if end >= n {
			end = n - 1
		}

		// Lance une goroutine pour chaque segment
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

	// Collecte et agrège les résultats
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

	// Calcul et affichage des métriques finales
	metrics.EndTime = time.Now()
	duration := metrics.EndTime.Sub(metrics.StartTime)
	avgTime := duration / time.Duration(metrics.TotalCalculations)

	// Affiche les résultats
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
