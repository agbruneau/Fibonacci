/*
 * Matrix-ClaudeAI
 *
 * Langage : Go
 *
 * Description :
 * Ce programme implémente une méthode efficace et concurrente pour le calcul de la suite de Fibonacci,
 * utilisant l'algorithme d'exponentiation rapide des matrices. Pour maximiser l'efficacité, il fait appel à
 * des techniques de parallélisation via un pool de workers. Le calcul se fait de manière itérative en
 * utilisant des structures de données adaptées aux grands nombres (big.Int) pour gérer les grandes valeurs
 * de la suite de Fibonacci.
 *
 * Fonctionnalités :
 * - Calcul d'un nombre de Fibonacci via exponentiation rapide des matrices (logarithmique en complexité).
 * - Utilisation d'un cache pour éviter les recalculs de valeurs déjà obtenues.
 * - Pool de workers pour paralléliser le calcul des segments de la suite de Fibonacci.
 * - Gestion des ressources via des sémaphores pour coordonner les accès concurrents.
 * - Gestion du contexte et de l'annulation pour garantir que les calculs ne dépassent pas les contraintes
 *   temporelles définies.
 *
 * Structure :
 * 1. Matrix2x2 : Représentation d'une matrice 2x2 nécessaire pour le calcul.
 * 2. FibCalculator : Gestion du calcul de Fibonacci en utilisant les matrices.
 * 3. WorkerPool : Gestion d'un pool de workers pour distribuer le calcul des segments de Fibonacci.
 * 4. Fonction Main : Initialisation du pool de workers, distribution du travail et collecte des résultats.
 *
 * Ce programme est conçu pour démontrer l'utilisation des techniques de calcul efficace combinées à la
 * parallélisation, avec une gestion appropriée des ressources et des grandes valeurs numériques.
 */

package main

import (
	"context"     // Gestion du contexte pour l'annulation et les délais d'expiration
	"fmt"         // Utilisé pour l'affichage formaté des résultats et des erreurs
	"math/big"    // Bibliothèque pour manipuler des grands entiers (big.Int) nécessaires pour les grands nombres de Fibonacci
	"runtime"     // Utilisé pour obtenir le nombre de CPU disponibles pour la parallélisation
	"sync"        // Fournit des primitives de synchronisation, comme sync.Mutex et sync.WaitGroup
	"sync/atomic" // Opérations atomiques pour gérer les accès concurrents aux variables partagées
	"time"        // Utilisé pour gérer les délais d'expiration et mesurer la durée des calculs

	"golang.org/x/sync/semaphore" // Implémentation des sémaphores pour gérer l'accès concurrent aux workers
)

// Matrix2x2 représente une matrice 2x2 pour le calcul de Fibonacci
type Matrix2x2 struct {
	a00, a01, a10, a11 *big.Int
}

// NewMatrix2x2 crée une nouvelle matrice 2x2
func NewMatrix2x2() *Matrix2x2 {
	return &Matrix2x2{
		a00: new(big.Int), a01: new(big.Int),
		a10: new(big.Int), a11: new(big.Int),
	}
}

// FibCalculator encapsule la logique de calcul de Fibonacci
type FibCalculator struct {
	cache     sync.Map
	matrix    *Matrix2x2
	tempMat   *Matrix2x2
	resultMat *Matrix2x2
	mutex     sync.Mutex
}

// NewFibCalculator crée une nouvelle instance de FibCalculator
func NewFibCalculator() *FibCalculator {
	fc := &FibCalculator{
		matrix:    NewMatrix2x2(),
		tempMat:   NewMatrix2x2(),
		resultMat: NewMatrix2x2(),
	}
	return fc
}

// multiplyMatrix multiplie deux matrices 2x2
func (fc *FibCalculator) multiplyMatrix(m1, m2, result *Matrix2x2) {
	// Calcul des nouvelles valeurs des éléments de la matrice résultat
	temp00 := new(big.Int).Mul(m1.a00, m2.a00)
	temp00.Add(temp00, new(big.Int).Mul(m1.a01, m2.a10))

	temp01 := new(big.Int).Mul(m1.a00, m2.a01)
	temp01.Add(temp01, new(big.Int).Mul(m1.a01, m2.a11))

	temp10 := new(big.Int).Mul(m1.a10, m2.a00)
	temp10.Add(temp10, new(big.Int).Mul(m1.a11, m2.a10))

	temp11 := new(big.Int).Mul(m1.a10, m2.a01)
	temp11.Add(temp11, new(big.Int).Mul(m1.a11, m2.a11))

	// Mettre à jour les valeurs de la matrice résultat
	result.a00.Set(temp00)
	result.a01.Set(temp01)
	result.a10.Set(temp10)
	result.a11.Set(temp11)
}

// Calculate calcule le n-ième nombre de Fibonacci
func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être positif")
	}

	// Vérifier si le contexte a été annulé
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Vérifier si la valeur est déjà en cache
	if cachedValue, ok := fc.cache.Load(n); ok {
		return new(big.Int).Set(cachedValue.(*big.Int)), nil
	}

	// Mutex pour empêcher les écritures concurrentes
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Initialiser la matrice de base pour la multiplication
	fc.matrix.a00.SetInt64(1)
	fc.matrix.a01.SetInt64(1)
	fc.matrix.a10.SetInt64(1)
	fc.matrix.a11.SetInt64(0)

	// Initialiser la matrice résultat comme matrice identité (neutre pour la multiplication)
	fc.resultMat.a00.SetInt64(1)
	fc.resultMat.a01.SetInt64(0)
	fc.resultMat.a10.SetInt64(0)
	fc.resultMat.a11.SetInt64(1)

	// Exponentiation rapide de la matrice pour calculer Fibonacci en temps logarithmique
	power := n - 1
	for power > 0 {
		// Si le bit actuel est à 1, multiplier la matrice résultat par la matrice de base
		if power&1 == 1 {
			fc.multiplyMatrix(fc.resultMat, fc.matrix, fc.tempMat)
			fc.resultMat, fc.tempMat = fc.tempMat, fc.resultMat
		}
		// Élever la matrice de base au carré
		fc.multiplyMatrix(fc.matrix, fc.matrix, fc.tempMat)
		fc.matrix, fc.tempMat = fc.tempMat, fc.matrix
		power >>= 1

		// Vérifier le contexte périodiquement pour l'annulation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	// Stocker le résultat dans le cache et le retourner
	result := new(big.Int).Set(fc.resultMat.a00)
	fc.cache.Store(n, result)
	return result, nil
}

// WorkerPool gère un pool de workers pour le calcul parallèle
type WorkerPool struct {
	calculators []*FibCalculator
	sem         *semaphore.Weighted
	current     uint64
}

// NewWorkerPool crée un nouveau pool de workers
func NewWorkerPool(size int) *WorkerPool {
	calculators := make([]*FibCalculator, size)
	for i := range calculators {
		calculators[i] = NewFibCalculator()
	}
	return &WorkerPool{
		calculators: calculators,
		sem:         semaphore.NewWeighted(int64(size)),
	}
}

// GetCalculator obtient un calculateur du pool de façon à équilibrer la charge
func (wp *WorkerPool) GetCalculator() *FibCalculator {
	current := atomic.AddUint64(&wp.current, 1)
	return wp.calculators[current%uint64(len(wp.calculators))]
}

// ProcessSegment traite un segment de calculs Fibonacci
func (wp *WorkerPool) ProcessSegment(ctx context.Context, start, end int, results chan<- *big.Int) error {
	// Acquérir un "permis" pour un worker avant de démarrer un calcul
	if err := wp.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer wp.sem.Release(1)

	// Obtenir un calculateur du pool
	calc := wp.GetCalculator()
	sum := new(big.Int)

	// Calculer la somme des nombres de Fibonacci dans le segment donné
	for i := start; i <= end; i++ {
		// Vérifier si le contexte a été annulé
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fibValue, err := calc.Calculate(ctx, i)
			if err != nil {
				return err
			}
			sum.Add(sum, fibValue)
		}
	}

	// Envoyer la somme partielle sur le canal des résultats
	select {
	case results <- sum:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	// Créer un contexte avec une limite de temps
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	n := 100000                   // Limite de la suite de Fibonacci
	numCPU := runtime.NumCPU()    // Nombre de CPU disponibles
	pool := NewWorkerPool(numCPU) // Créer un pool de workers

	// Calculer la taille optimale des segments à traiter par les workers
	segmentSize := 1000
	if n > 10000 {
		segmentSize = n / (numCPU * 4)
	}

	results := make(chan *big.Int, numCPU)
	var wg sync.WaitGroup
	var errCount uint64

	startTime := time.Now()

	// Distribuer le travail entre les workers
	for start := 0; start < n; start += segmentSize {
		end := start + segmentSize - 1
		if end >= n {
			end = n - 1
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			if err := pool.ProcessSegment(ctx, s, e, results); err != nil {
				fmt.Printf("Erreur segment %d-%d: %v\n", s, e, err)
				atomic.AddUint64(&errCount, 1)
			}
		}(start, end)
	}

	// Goroutine pour fermer le canal des résultats une fois que tous les workers ont fini
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collecter les résultats de chaque segment
	finalSum := new(big.Int)
	for partialSum := range results {
		finalSum.Add(finalSum, partialSum)
	}

	// Afficher les statistiques
	duration := time.Since(startTime)
	fmt.Printf("Temps total: %v\n", duration)
	fmt.Printf("Erreurs: %d\n", atomic.LoadUint64(&errCount))
	fmt.Printf("Résultat: %s\n", formatBigInt(finalSum))
	fmt.Printf("Performance moyenne: %v par calcul\n", duration/time.Duration(n))
}

// formatBigInt formate un grand nombre en notation scientifique
func formatBigInt(n *big.Int) string {
	str := n.String()
	length := len(str)
	if length <= 10 {
		return str
	}
	// Retourner le nombre en notation scientifique pour les grands chiffres
	return fmt.Sprintf("%s.%se%d", str[0:1], str[1:10], length-1)
}
