package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru/simplelru"
)

// Métriques de performance
// La structure Metrics permet de collecter des statistiques de performance, telles que les succès de cache, les échecs, et le temps total passé dans les différentes phases de calcul.
type Metrics struct {
	CacheHits       uint64        // Nombre de hits du cache
	CacheMisses     uint64        // Nombre de ratés du cache
	SegmentsCreated uint64        // Nombre de segments créés
	TimeInDoubling  time.Duration // Temps passé dans les calculs utilisant la méthode de doublement
	TimeInSegments  time.Duration // Temps passé dans le calcul des segments
}

// Structure pour les résultats de segment en streaming
// Cette structure représente un segment de la séquence de Fibonacci, et inclut les valeurs calculées et l'état du segment.
type SegmentResult struct {
	Index    int        // L'index de départ de ce segment dans la séquence de Fibonacci
	Values   []*big.Int // Les valeurs calculées pour ce segment
	Complete bool       // Indique si le calcul du segment est complet
	Error    error      // Toute erreur survenue lors du calcul de ce segment
}

// Configuration du calculateur de Fibonacci
// Contient les paramètres utilisés pour le calcul parallèle de la séquence de Fibonacci.
type Config struct {
	InitialSegmentSize int  // Taille initiale des segments de calcul
	MinSegmentSize     int  // Taille minimale des segments de calcul
	MaxSegmentSize     int  // Taille maximale des segments de calcul
	WorkerCount        int  // Nombre de workers parallèles utilisés pour le calcul
	CacheSize          int  // Taille du cache LRU utilisé pour mémoriser les valeurs calculées
	AdaptiveSegments   bool // Indique si la taille des segments doit être ajustée dynamiquement
}

// Structure principale optimisée pour le calcul des nombres de Fibonacci
// Utilise la méthode de doublement pour optimiser le calcul parallèle.
type DoublingFibCalculator struct {
	config     Config       // Configuration du calculateur
	cache      *lru.LRU     // Cache LRU pour mémoriser les valeurs déjà calculées
	cacheMutex sync.RWMutex // Mutex pour protéger l'accès au cache en lecture/écriture
	metrics    *Metrics     // Métriques de performance pour le suivi des opérations
}

// Configuration par défaut
// Retourne la configuration par défaut pour le calcul parallèle des nombres de Fibonacci.
func DefaultConfig() Config {
	return Config{
		InitialSegmentSize: 1000,
		MinSegmentSize:     100,
		MaxSegmentSize:     5000,
		WorkerCount:        runtime.NumCPU(),
		CacheSize:          10000,
		AdaptiveSegments:   true,
	}
}

// Créer un nouveau calculateur utilisant la méthode de doublement
func NewDoublingCalculator(config Config) (*DoublingFibCalculator, error) {
	// Initialiser le cache LRU
	cache, err := lru.NewLRU(config.CacheSize, nil)
	if err != nil {
		return nil, fmt.Errorf("échec de l'initialisation du cache: %v", err)
	}

	// Retourner une instance de DoublingFibCalculator
	return &DoublingFibCalculator{
		config:  config,
		cache:   cache,
		metrics: &Metrics{},
	}, nil
}

// Méthode de doublement optimisée pour calculer un seul nombre de Fibonacci
// Utilise une méthode de doublement pour calculer efficacement le n-ième nombre de Fibonacci.
func (calc *DoublingFibCalculator) fibDoubling(n int) (*big.Int, error) {
	startTime := time.Now()
	defer func() {
		calc.metrics.TimeInDoubling += time.Since(startTime)
	}()

	// Vérifier si l'entrée est valide
	if n < 0 {
		return nil, errors.New("n doit être positif")
	}
	if n < 2 {
		// Les deux premiers nombres de Fibonacci sont 0 et 1
		return big.NewInt(int64(n)), nil
	}

	// Vérifier le cache pour éviter un calcul redondant
	if val, exists := calc.getFromCache(n); exists {
		return val, nil
	}

	// Initialiser les valeurs de départ
	a := big.NewInt(0) // F(n)
	b := big.NewInt(1) // F(n+1)
	c := new(big.Int)  // Valeur temporaire pour le calcul
	d := new(big.Int)  // Valeur temporaire pour le calcul

	// Utiliser l'algorithme de doublement pour calculer F(n)
	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		// c = a * (2 * b - a)
		c.Lsh(b, 1) // c = 2 * b
		c.Sub(c, a) // c = 2 * b - a
		c.Mul(c, a) // c = a * (2 * b - a)

		// d = a² + b²
		d.Mul(a, a)           // d = a²
		d.Add(d, b.Mul(b, b)) // d = a² + b²

		// Mise à jour des valeurs selon le bit actuel de n
		if ((n >> i) & 1) == 0 {
			a.Set(c)
			b.Set(d)
		} else {
			a.Set(d)
			b.Add(c, d)
		}
	}

	// Mettre en cache le résultat pour un accès futur plus rapide
	result := new(big.Int).Set(a)
	calc.addToCache(n, result)

	return result, nil
}

// Calcul d'un segment utilisant la méthode de doublement
// Calcule un segment de la séquence de Fibonacci à partir d'un index de départ et d'une taille donnée.
func (calc *DoublingFibCalculator) calculateSegment(start, size int) (*SegmentResult, error) {
	startTime := time.Now()
	defer func() {
		calc.metrics.TimeInSegments += time.Since(startTime)
	}()

	// Initialiser le résultat du segment
	result := &SegmentResult{
		Index:  start,
		Values: make([]*big.Int, size),
	}

	// Calculer les deux premières valeurs du segment
	var err error
	result.Values[0], err = calc.fibDoubling(start)
	if err != nil {
		return nil, err
	}

	if size > 1 {
		result.Values[1], err = calc.fibDoubling(start + 1)
		if err != nil {
			return nil, err
		}
	}

	// Calculer le reste du segment en utilisant la relation de récurrence de Fibonacci
	for i := 2; i < size; i++ {
		result.Values[i], err = calc.fibDoubling(start + i)
		if err != nil {
			return nil, err
		}

		// Vérification de cohérence pour chaque valeur calculée
		expected := new(big.Int).Add(result.Values[i-1], result.Values[i-2])
		if result.Values[i].Cmp(expected) != 0 {
			return nil, fmt.Errorf("erreur de cohérence à l'index %d", start+i)
		}
	}

	result.Complete = true
	return result, nil
}

// Pipeline de calcul optimisé
// Structure pour orchestrer les différentes étapes du calcul parallèle.
type calculationPipeline struct {
	segmentChan chan *SegmentResult    // Canal pour envoyer les segments calculés
	resultChan  chan *SegmentResult    // Canal pour les résultats finaux
	errorChan   chan error             // Canal pour les erreurs éventuelles
	workQueue   chan int               // Queue de travail pour les indices de segments à calculer
	wg          sync.WaitGroup         // Groupe d'attente pour la synchronisation des goroutines
	ctx         context.Context        // Contexte pour la gestion des annulations
	calculator  *DoublingFibCalculator // Calculateur de Fibonacci
	totalSize   int                    // Taille totale de la séquence à calculer
}

// Créer un nouveau pipeline de calcul
func newCalculationPipeline(ctx context.Context, calc *DoublingFibCalculator, size int) *calculationPipeline {
	return &calculationPipeline{
		segmentChan: make(chan *SegmentResult, calc.config.WorkerCount),
		resultChan:  make(chan *SegmentResult, calc.config.WorkerCount),
		errorChan:   make(chan error, 1),
		workQueue:   make(chan int, calc.config.WorkerCount),
		ctx:         ctx,
		calculator:  calc,
		totalSize:   size,
	}
}

// Calcul parallèle avec streaming des résultats
// Calcule les nombres de Fibonacci en parallèle, en envoyant les résultats au fur et à mesure.
func (calc *DoublingFibCalculator) CalculateParallelFibonacciStream(ctx context.Context, n int) (<-chan *SegmentResult, <-chan error) {
	if n < 0 {
		// Retourner une erreur si l'entrée est négative
		errChan := make(chan error, 1)
		errChan <- errors.New("n doit être positif")
		return nil, errChan
	}

	pipeline := newCalculationPipeline(ctx, calc, n)
	resultChan := make(chan *SegmentResult, calc.config.WorkerCount)
	errorChan := make(chan error, 1)

	// Démarrer les workers pour le calcul parallèle
	for i := 0; i < calc.config.WorkerCount; i++ {
		pipeline.wg.Add(1)
		go calc.segmentWorker(pipeline)
	}

	// Gérer la distribution du travail entre les workers
	go func() {
		defer close(pipeline.workQueue)
		remainingSize := n + 1
		currentStart := 0

		for remainingSize > 0 {
			// Calculer la taille adaptative du segment
			segmentSize := calc.calculateAdaptiveSegmentSize(remainingSize, currentStart)
			if segmentSize > remainingSize {
				segmentSize = remainingSize
			}

			select {
			case pipeline.workQueue <- currentStart:
				// Ajouter un segment aux métriques
				atomic.AddUint64(&calc.metrics.SegmentsCreated, 1)
				currentStart += segmentSize
				remainingSize -= segmentSize
			case <-ctx.Done():
				// Arrêter la distribution du travail si le contexte est annulé
				errorChan <- ctx.Err()
				return
			}
		}
	}()

	// Collecter et transmettre les résultats des segments
	go func() {
		defer close(resultChan)
		defer close(errorChan)

		results := make(map[int]*SegmentResult)
		nextIndexToSend := 0

		for result := range pipeline.resultChan {
			if result.Error != nil {
				errorChan <- result.Error
				return
			}

			results[result.Index] = result

			// Envoyer les résultats dans l'ordre
			for {
				if segment, ok := results[nextIndexToSend]; ok {
					select {
					case resultChan <- segment:
						delete(results, nextIndexToSend)
						nextIndexToSend += len(segment.Values)
					case <-ctx.Done():
						// Annuler l'envoi si le contexte est terminé
						errorChan <- ctx.Err()
						return
					}
				} else {
					break
				}
			}
		}
	}()

	return resultChan, errorChan
}

// Worker optimisé pour le calcul des segments
// Les workers calculent les segments de la séquence de Fibonacci en parallèle.
func (calc *DoublingFibCalculator) segmentWorker(pipeline *calculationPipeline) {
	defer pipeline.wg.Done()

	for start := range pipeline.workQueue {
		select {
		case <-pipeline.ctx.Done():
			// Arrêter le worker si le contexte est annulé
			return
		default:
			// Calculer la taille du segment et le résultat
			size := calc.calculateAdaptiveSegmentSize(pipeline.totalSize-start, start)
			result, err := calc.calculateSegment(start, size)
			if err != nil {
				pipeline.errorChan <- err
				return
			}
			pipeline.resultChan <- result
		}
	}
}

// Calcul adaptatif de la taille des segments
// Cette méthode ajuste la taille des segments en fonction des paramètres de configuration et de la progression actuelle.
func (calc *DoublingFibCalculator) calculateAdaptiveSegmentSize(remainingSize, currentStart int) int {
	if !calc.config.AdaptiveSegments {
		// Retourner la taille par défaut si l'adaptation est désactivée
		return calc.config.InitialSegmentSize
	}

	// Adapter la taille du segment en fonction de la progression
	baseSize := calc.config.InitialSegmentSize
	if currentStart > 0 {
		// Augmenter la taille des segments pour les calculs plus avancés
		baseSize = int(float64(baseSize) * 1.2)
	}

	// Réduire la taille si proche de la fin
	if remainingSize < baseSize*2 {
		baseSize = remainingSize / 2
	}

	// Respecter les limites minimales et maximales définies dans la configuration
	if baseSize < calc.config.MinSegmentSize {
		baseSize = calc.config.MinSegmentSize
	}
	if baseSize > calc.config.MaxSegmentSize {
		baseSize = calc.config.MaxSegmentSize
	}

	return baseSize
}

// Gestion optimisée du cache
// Méthodes pour obtenir et ajouter des valeurs au cache LRU.
func (calc *DoublingFibCalculator) getFromCache(n int) (*big.Int, bool) {
	calc.cacheMutex.RLock()
	defer calc.cacheMutex.RUnlock()

	if val, ok := calc.cache.Get(n); ok {
		// Incrémenter le nombre de hits du cache
		atomic.AddUint64(&calc.metrics.CacheHits, 1)
		return val.(*big.Int), true
	}
	// Incrémenter le nombre de ratés du cache
	atomic.AddUint64(&calc.metrics.CacheMisses, 1)
	return nil, false
}

// Ajouter une valeur au cache LRU
func (calc *DoublingFibCalculator) addToCache(n int, value *big.Int) {
	calc.cacheMutex.Lock()
	defer calc.cacheMutex.Unlock()
	calc.cache.Add(n, new(big.Int).Set(value))
}

// Exemple d'utilisation
// Utilise le calculateur pour calculer des valeurs de Fibonacci en parallèle.
func main() {
	// Créer la configuration par défaut et augmenter le nombre de workers
	config := DefaultConfig()
	config.AdaptiveSegments = true
	config.WorkerCount = runtime.NumCPU() * 2

	// Initialiser le calculateur de Fibonacci
	calculator, err := NewDoublingCalculator(config)
	if err != nil {
		panic(err)
	}

	// Créer un contexte avec une durée limite pour l'exécution
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	n := 1000 // Nombre de Fibonacci à calculer
	startTime := time.Now()

	// Démarrer le calcul parallèle
	resultChan, errorChan := calculator.CalculateParallelFibonacciStream(ctx, n)

	// Traiter les résultats en streaming
	var lastNumber *big.Int
	segmentCount := 0

	for result := range resultChan {
		segmentCount++
		if len(result.Values) > 0 {
			// Stocker le dernier nombre calculé
			lastNumber = result.Values[len(result.Values)-1]
		}

		select {
		case err := <-errorChan:
			if err != nil {
				panic(err)
			}
		default:
		}
	}

	duration := time.Since(startTime)

	// Afficher les métriques
	fmt.Printf("Calcul terminé en %s\n", duration)
	fmt.Printf("Dernier nombre: %s\n", lastNumber)
	fmt.Printf("Segments calculés: %d\n", segmentCount)
	fmt.Printf("Cache hits: %d, misses: %d\n", calculator.metrics.CacheHits, calculator.metrics.CacheMisses)
	fmt.Printf("Temps dans les calculs de doublement: %s\n", calculator.metrics.TimeInDoubling)
	fmt.Printf("Temps dans les calculs de segments: %s\n", calculator.metrics.TimeInSegments)
}
