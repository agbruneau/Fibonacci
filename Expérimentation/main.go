// =============================================================================
// Programme : Calcul et affichage de Fibonacci(n) en notation scientifique
//
// Description :
// Ce programme calcule le n‑ième nombre de Fibonacci en utilisant l'algorithme
// du doublement (doubling method), qui permet de réduire la complexité du calcul
// à O(log n). Pour optimiser les opérations sur de grands entiers (big.Int), le
// code parallélise les multiplications coûteuses à l'aide de goroutines et de canaux.
// Un contexte avec timeout est mis en place pour limiter la durée d'exécution, et
// des métriques de performance sont collectées. Le résultat est affiché en notation
// scientifique avec l'exposant rendu en notation exponentielle (ex: 1.23e45), facilitant
// ainsi la lecture de nombres très volumineux.
//
// Techniques employées :
// - Algorithme du doublement pour calculer Fibonacci(n) efficacement.
// - Parallélisation des multiplications (big.Int) par goroutines pour exploiter
//   la puissance des processeurs multicœurs.
// - Utilisation d'un contexte avec timeout pour la robustesse du calcul.
// - Formatage en notation scientifique avec notation exponentielle.
// - Pool de workers pour gérer efficacement les multiplications parallèles.
// - Système de cache pour éviter de recalculer les valeurs déjà connues.
// =============================================================================

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"
)

// Configuration centralise les paramètres configurables du programme.
type Configuration struct {
	M           int           // Calcul de Fibonacci(M)
	Timeout     time.Duration // Durée maximale d'exécution
	WorkerCount int           // Nombre de workers pour le pool
	CacheSize   int           // Taille maximale du cache
	CPUProfile  string        // Fichier pour le profilage CPU
	MemProfile  string        // Fichier pour le profilage mémoire
}

// DefaultConfig retourne une configuration par défaut.
func DefaultConfig() Configuration {
	return Configuration{
		M:           200000000,
		Timeout:     5 * time.Minute,
		WorkerCount: runtime.NumCPU(),
		CacheSize:   1000,
	}
}

// ParseFlags analyse les arguments de ligne de commande et met à jour la configuration.
func ParseFlags(config *Configuration) {
	flag.IntVar(&config.M, "n", config.M, "Calcul de Fibonacci(n)")
	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "Durée maximale d'exécution")
	flag.IntVar(&config.WorkerCount, "workers", config.WorkerCount, "Nombre de workers pour le pool")
	flag.IntVar(&config.CacheSize, "cache", config.CacheSize, "Taille maximale du cache")
	flag.StringVar(&config.CPUProfile, "cpuprofile", "", "Écrire le profil CPU dans un fichier")
	flag.StringVar(&config.MemProfile, "memprofile", "", "Écrire le profil mémoire dans un fichier")
	flag.Parse()
}

// Metrics conserve des informations de performance du calcul.
type Metrics struct {
	StartTime          time.Time     // Heure de début du calcul
	EndTime            time.Time     // Heure de fin du calcul
	TotalCalculations  int64         // Nombre de calculs réalisés
	CacheHits          int64         // Nombre d'accès au cache
	CacheMisses        int64         // Nombre d'échecs d'accès au cache
	MultiplicationTime time.Duration // Temps passé en multiplications
	mu                 sync.Mutex    // Mutex pour les opérations non atomiques
}

// NewMetrics initialise les métriques en enregistrant l'heure de début.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// AddCalculations incrémente de manière atomique le compteur de calculs.
func (m *Metrics) AddCalculations(n int64) {
	atomic.AddInt64(&m.TotalCalculations, n)
}

// AddCacheHit incrémente de manière atomique le compteur de hits du cache.
func (m *Metrics) AddCacheHit() {
	atomic.AddInt64(&m.CacheHits, 1)
}

// AddCacheMiss incrémente de manière atomique le compteur d'échecs du cache.
func (m *Metrics) AddCacheMiss() {
	atomic.AddInt64(&m.CacheMisses, 1)
}

// AddMultiplicationTime ajoute au temps total passé en multiplications.
func (m *Metrics) AddMultiplicationTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MultiplicationTime += duration
}

// PrintMetrics affiche les métriques de performance.
func (m *Metrics) PrintMetrics(config Configuration) {
	duration := m.EndTime.Sub(m.StartTime)

	var avgTime time.Duration
	if m.TotalCalculations > 0 {
		avgTime = duration / time.Duration(m.TotalCalculations)
	}

	cacheRatio := float64(0)
	if m.CacheHits+m.CacheMisses > 0 {
		cacheRatio = float64(m.CacheHits) / float64(m.CacheHits+m.CacheMisses) * 100
	}

	fmt.Printf("\nConfiguration :\n")
	fmt.Printf("  Valeur de n             : %d\n", config.M)
	fmt.Printf("  Timeout                 : %v\n", config.Timeout)
	fmt.Printf("  Nombre de workers       : %d\n", config.WorkerCount)
	fmt.Printf("  Taille du cache         : %d\n", config.CacheSize)

	fmt.Printf("\nPerformance :\n")
	fmt.Printf("  Temps total d'exécution : %v\n", duration)
	fmt.Printf("  Temps en multiplications: %v (%.2f%%)\n", m.MultiplicationTime, float64(m.MultiplicationTime)/float64(duration)*100)
	fmt.Printf("  Nombre de calculs       : %d\n", m.TotalCalculations)
	fmt.Printf("  Hits du cache           : %d\n", m.CacheHits)
	fmt.Printf("  Ratio d'efficacité cache: %.2f%%\n", cacheRatio)
	fmt.Printf("  Temps moyen par calcul  : %v\n", avgTime)
}

// FibCache implémente un cache LRU simple pour les valeurs de Fibonacci.
type FibCache struct {
	cache    map[int]*big.Int
	maxSize  int
	metrics  *Metrics
	mu       sync.RWMutex
	lastUsed []int // Liste ordonnée des clés les plus récemment utilisées
}

// NewFibCache crée un nouveau cache pour les valeurs de Fibonacci.
func NewFibCache(maxSize int, metrics *Metrics) *FibCache {
	return &FibCache{
		cache:    make(map[int]*big.Int, maxSize),
		maxSize:  maxSize,
		metrics:  metrics,
		lastUsed: make([]int, 0, maxSize),
	}
}

// Get récupère une valeur du cache, s'il elle existe.
func (fc *FibCache) Get(n int) (*big.Int, bool) {
	fc.mu.RLock()
	val, exists := fc.cache[n]
	fc.mu.RUnlock()

	if exists {
		fc.updateUsage(n)
		if fc.metrics != nil {
			fc.metrics.AddCacheHit()
		}
		// Retourne une copie pour éviter les modifications
		return new(big.Int).Set(val), true
	}

	if fc.metrics != nil {
		fc.metrics.AddCacheMiss()
	}
	return nil, false
}

// Put ajoute une valeur au cache.
func (fc *FibCache) Put(n int, val *big.Int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Si le cache est plein, supprime l'élément le moins récemment utilisé
	if len(fc.cache) >= fc.maxSize && fc.maxSize > 0 {
		fc.evictLRU()
	}

	// Ajoute la nouvelle valeur et met à jour l'ordre d'utilisation
	fc.cache[n] = new(big.Int).Set(val)
	fc.updateUsageNoLock(n)
}

// updateUsage met à jour l'ordre d'utilisation des clés (thread-safe).
func (fc *FibCache) updateUsage(n int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.updateUsageNoLock(n)
}

// updateUsageNoLock met à jour l'ordre d'utilisation des clés (non thread-safe).
func (fc *FibCache) updateUsageNoLock(n int) {
	// Retire n de la liste s'il y est déjà
	for i, key := range fc.lastUsed {
		if key == n {
			fc.lastUsed = append(fc.lastUsed[:i], fc.lastUsed[i+1:]...)
			break
		}
	}

	// Ajoute n au début de la liste
	fc.lastUsed = append([]int{n}, fc.lastUsed...)
}

// evictLRU supprime l'élément le moins récemment utilisé du cache.
func (fc *FibCache) evictLRU() {
	if len(fc.lastUsed) > 0 {
		// Retire le dernier élément (le moins récemment utilisé)
		lruKey := fc.lastUsed[len(fc.lastUsed)-1]
		fc.lastUsed = fc.lastUsed[:len(fc.lastUsed)-1]
		delete(fc.cache, lruKey)
	}
}

// MultiplicationJob représente une tâche de multiplication à effectuer.
type MultiplicationJob struct {
	x, y       *big.Int
	resultChan chan<- *big.Int
	errChan    chan<- error
}

// WorkerPool gère un ensemble de goroutines pour traiter les multiplications en parallèle.
type WorkerPool struct {
	jobChan chan MultiplicationJob
	wg      sync.WaitGroup
	metrics *Metrics
}

// NewWorkerPool crée un nouveau pool de workers pour les multiplications.
func NewWorkerPool(workerCount int, metrics *Metrics) *WorkerPool {
	pool := &WorkerPool{
		jobChan: make(chan MultiplicationJob, workerCount*2),
		metrics: metrics,
	}

	// Démarrage des workers
	pool.wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go pool.worker()
	}

	return pool
}

// worker traite les multiplications dans une boucle infinie.
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for job := range wp.jobChan {
		startTime := time.Now()

		result, err := performMultiplication(job.x, job.y)

		if wp.metrics != nil {
			wp.metrics.AddMultiplicationTime(time.Since(startTime))
		}

		if err != nil {
			job.errChan <- err
		} else {
			job.resultChan <- result
		}
	}
}

// ScheduleMultiplication ajoute une tâche de multiplication au pool.
func (wp *WorkerPool) ScheduleMultiplication(x, y *big.Int, resultChan chan<- *big.Int, errChan chan<- error) {
	wp.jobChan <- MultiplicationJob{
		x:          x,
		y:          y,
		resultChan: resultChan,
		errChan:    errChan,
	}
}

// Shutdown arrête proprement le pool de workers.
func (wp *WorkerPool) Shutdown() {
	close(wp.jobChan)
	wp.wg.Wait()
}

// FibCalculator encapsule le calcul du n‑ième nombre de Fibonacci.
type FibCalculator struct {
	cache      *FibCache
	workerPool *WorkerPool
	metrics    *Metrics
}

// NewFibCalculator retourne une nouvelle instance de FibCalculator.
func NewFibCalculator(cache *FibCache, workerPool *WorkerPool, metrics *Metrics) *FibCalculator {
	return &FibCalculator{
		cache:      cache,
		workerPool: workerPool,
		metrics:    metrics,
	}
}

// Calculate retourne Fibonacci(n) pour n ≥ 0.
func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	// Vérification du contexte
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if n < 0 {
		return nil, fmt.Errorf("n doit être non négatif")
	}

	// Cas de base
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}

	// Vérification du cache
	if fc.cache != nil {
		if val, found := fc.cache.Get(n); found {
			return val, nil
		}
	}

	// Calcul avec l'algorithme du doublement
	result, err := fc.fibDoublingParallel(ctx, n)
	if err != nil {
		return nil, err
	}

	// Mise en cache du résultat
	if fc.cache != nil {
		fc.cache.Put(n, result)
	}

	return result, nil
}

// fibDoublingParallel calcule Fibonacci(n) en utilisant l'algorithme du doublement avec parallélisation.
func (fc *FibCalculator) fibDoublingParallel(ctx context.Context, n int) (*big.Int, error) {
	// Initialisation
	a := big.NewInt(0)
	b := big.NewInt(1)

	// Détermination du bit le plus significatif de n
	highest := determineHighestBit(n)

	// Création des big int utilisés dans la boucle
	twoB := new(big.Int)
	temp := new(big.Int)
	c := new(big.Int)
	d := new(big.Int)

	// Parcours des bits de n, du plus significatif au moins significatif
	for i := highest; i >= 0; i-- {
		// Vérification périodique du contexte
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calcul de deuxB = 2 * b (opération rapide via un décalage de bits)
		twoB.Lsh(b, 1)
		// Calcul de temp = 2*b - a
		temp.Sub(twoB, a)

		// Configuration des canaux pour les résultats
		cChan := make(chan *big.Int, 1)
		t1Chan := make(chan *big.Int, 1)
		t2Chan := make(chan *big.Int, 1)
		errChan := make(chan error, 3)

		// Lancement des multiplications parallèles via le pool de workers
		fc.workerPool.ScheduleMultiplication(a, temp, cChan, errChan)
		fc.workerPool.ScheduleMultiplication(a, a, t1Chan, errChan)
		fc.workerPool.ScheduleMultiplication(b, b, t2Chan, errChan)

		// Récupération des résultats et gestion des erreurs
		if err := fc.handleMultiplicationResults(ctx, cChan, t1Chan, t2Chan, errChan, c, d); err != nil {
			return nil, err
		}

		// Mise à jour des valeurs selon le bit courant
		updateFibonacciValues(n, i, a, b, c, d)
	}

	// À la fin de la boucle, a contient Fibonacci(n)
	return a, nil
}

// handleMultiplicationResults récupère les résultats des multiplications et gère les erreurs.
func (fc *FibCalculator) handleMultiplicationResults(
	ctx context.Context,
	cChan, t1Chan, t2Chan <-chan *big.Int,
	errChan <-chan error,
	c, d *big.Int,
) error {
	// Attente des résultats avec vérification du contexte
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			return err
		case c1 := <-cChan:
			c.Set(c1)
		case t1 := <-t1Chan:
			d.Set(t1)
		case t2 := <-t2Chan:
			// Ajout de t2 à d (qui contient déjà t1)
			d.Add(d, t2)
		}
	}

	return nil
}

// determineHighestBit détermine le bit le plus significatif de n.
func determineHighestBit(n int) int {
	highest := 0
	for i := 63; i >= 0; i-- {
		if n&(1<<uint(i)) != 0 {
			highest = i
			break
		}
	}
	return highest
}

// performMultiplication effectue la multiplication et retourne le résultat.
func performMultiplication(x, y *big.Int) (*big.Int, error) {
	if x == nil || y == nil {
		return nil, errors.New("cannot multiply nil big.Int")
	}
	return new(big.Int).Mul(x, y), nil
}

// updateFibonacciValues met à jour les valeurs de a et b selon la valeur du bit courant de n.
func updateFibonacciValues(n, i int, a, b, c, d *big.Int) {
	if n&(1<<uint(i)) != 0 {
		a.Set(d)
		b.Add(c, d)
	} else {
		a.Set(c)
		b.Set(d)
	}
}

// formatBigIntExp formate un grand entier en notation scientifique avec exposant.
func formatBigIntExp(n *big.Int, precision int) string {
	if n == nil {
		return "nil"
	}

	if n.Sign() == 0 {
		return "0"
	}

	s := n.String()
	isNegative := false
	if s[0] == '-' {
		isNegative = true
		s = s[1:]
	}

	// Cas simple si le nombre a un seul chiffre
	if len(s) <= 1 {
		if isNegative {
			return "-" + s
		}
		return s
	}

	// Détermination du nombre de chiffres significatifs
	var significand string
	if len(s) > precision {
		significand = s[:1] + "." + s[1:precision]
	} else {
		significand = s[:1] + "."
		if len(s) > 1 {
			significand += s[1:]
		}
		// Ajout de zéros pour atteindre la précision demandée
		for i := 0; i < precision-len(s); i++ {
			significand += "0"
		}
	}

	// Calcul de l'exposant
	exponent := len(s) - 1

	// Formatage de la sortie
	if isNegative {
		return fmt.Sprintf("-%se%d", significand, exponent)
	}
	return fmt.Sprintf("%se%d", significand, exponent)
}

// main constitue le point d'entrée du programme.
func main() {
	// Récupération de la configuration
	config := DefaultConfig()
	ParseFlags(&config)

	// Configuration du profilage CPU si demandé
	if config.CPUProfile != "" {
		f, err := os.Create(config.CPUProfile)
		if err != nil {
			log.Fatal("Erreur lors de la création du fichier de profil CPU:", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("Erreur lors du démarrage du profilage CPU:", err)
		}
		defer pprof.StopCPUProfile()
	}

	// Initialisation des métriques
	metrics := NewMetrics()

	// Création du contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Initialisation du cache
	cache := NewFibCache(config.CacheSize, metrics)

	// Initialisation du pool de workers
	workerPool := NewWorkerPool(config.WorkerCount, metrics)
	defer workerPool.Shutdown()

	// Création du calculateur de Fibonacci
	fc := NewFibCalculator(cache, workerPool, metrics)

	// Canaux pour récupérer le résultat ou une erreur
	resultChan := make(chan *big.Int, 1)
	errorChan := make(chan error, 1)

	// Lancement du calcul dans une goroutine
	go func() {
		fib, err := fc.Calculate(ctx, config.M)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- fib
	}()

	// Attente du résultat
	var fibResult *big.Int
	select {
	case <-ctx.Done():
		log.Fatalf("Délai d'exécution dépassé : %v", ctx.Err())
	case err := <-errorChan:
		log.Fatalf("Erreur lors du calcul de Fibonacci : %v", err)
	case fibResult = <-resultChan:
		// Le calcul s'est terminé correctement
	}

	// Enregistrement des métriques finales
	metrics.AddCalculations(1)
	metrics.EndTime = time.Now()

	// Affichage des métriques
	metrics.PrintMetrics(config)

	// Affichage du résultat
	fmt.Printf("\nRésultat :\n")
	fmt.Printf("  Fibonacci(%d) : %s\n", config.M, formatBigIntExp(fibResult, 6))

	// Profilage mémoire si demandé
	if config.MemProfile != "" {
		f, err := os.Create(config.MemProfile)
		if err != nil {
			log.Fatal("Erreur lors de la création du fichier de profil mémoire:", err)
		}
		defer f.Close()
		runtime.GC() // Forcer une collecte des déchets avant le profilage
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("Erreur lors de l'écriture du profil mémoire:", err)
		}
	}
}
