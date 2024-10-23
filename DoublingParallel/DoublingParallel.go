// Obtenir les métriques actuelles
// curl http://localhost:8080/metrics

// Calculer un nombre de Fibonacci
// curl -X POST -H "Content-Type: application/json" -d '{"n": 1000}' http://localhost:8080/compute

// Métriques détaillées pour le service Fibonacci
type FibMetrics struct {
	ComputationDuration atomic.Int64   // Durée totale des calculs en nanosecondes
	BitOperationsCount  atomic.Int64   // Nombre d'opérations bit à bit effectuées
	MemoryAllocations   atomic.Uint64  // Nombre d'allocations mémoire
	CacheEfficiency     atomic.Float64 // Ratio de hits cache (hits / total requests)

	// Statistiques des calculs
	TotalCalculations atomic.Int64   // Nombre total de calculs effectués
	AvgComputeTime    atomic.Float64 // Temps moyen de calcul en millisecondes
	PeakMemoryUsage   atomic.Uint64  // Utilisation maximale de la mémoire en bytes

	// Métriques de performance du cache
	CacheSize     atomic.Int64 // Taille actuelle du cache
	CacheHits     atomic.Int64 // Nombre de hits du cache
	CacheMisses   atomic.Int64 // Nombre de misses du cache
	EvictionCount atomic.Int64 // Nombre d'éléments évincés du cache

	// Métriques du pool
	PoolAcquisitions atomic.Int64 // Nombre total d'acquisitions depuis le pool
	PoolMisses       atomic.Int64 // Nombre de fois où le pool était vide
}

// Service Fibonacci modifié avec les nouvelles métriques
type FibService struct {
	config  Config
	metrics *FibMetrics
	cache   Cache
	logger  *log.Logger
}

// Constructeur mis à jour du service
func NewFibService(cfg Config) (*FibService, error) {
	lruCache, err := lru.New(cfg.MaxCacheSize)
	if err != nil {
		return nil, fmt.Errorf("initialisation du cache: %w", ErrCacheInitFailed)
	}

	cache := &LRUCache{
		cache: lruCache,
		onEvicted: func(key, value interface{}) {
			metrics.EvictionCount.Add(1)
		},
	}

	metrics := &FibMetrics{}

	logger := log.New(os.Stdout, "[FIB] ", log.LstdFlags|log.Lmicroseconds)

	return &FibService{
		config:  cfg,
		metrics: metrics,
		cache:   cache,
		logger:  logger,
	}, nil
}

// Méthode pour mettre à jour les métriques de calcul
func (s *FibService) updateComputationMetrics(startTime time.Time, n int) {
	duration := time.Since(startTime)
	s.metrics.ComputationDuration.Add(duration.Nanoseconds())
	s.metrics.TotalCalculations.Add(1)

	// Mise à jour du temps moyen de calcul
	total := float64(s.metrics.ComputationDuration.Load())
	count := float64(s.metrics.TotalCalculations.Load())
	s.metrics.AvgComputeTime.Store(total / count / 1e6) // Conversion en millisecondes

	// Mise à jour des métriques mémoire
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.metrics.MemoryAllocations.Add(m.Mallocs - m.Frees)

	// Mise à jour du pic mémoire si nécessaire
	for {
		current := s.metrics.PeakMemoryUsage.Load()
		if m.Alloc <= current {
			break
		}
		if s.metrics.PeakMemoryUsage.CompareAndSwap(current, m.Alloc) {
			break
		}
	}

	// Mise à jour de l'efficacité du cache
	hits := float64(s.metrics.CacheHits.Load())
	total = float64(hits + s.metrics.CacheMisses.Load())
	if total > 0 {
		s.metrics.CacheEfficiency.Store(hits / total)
	}
}

// Méthode pour mettre à jour les métriques du pool
func (s *FibService) updatePoolMetrics(acquired bool) {
	s.metrics.PoolAcquisitions.Add(1)
	if !acquired {
		s.metrics.PoolMisses.Add(1)
	}
}

// Méthode ComputeFib mise à jour avec les nouvelles métriques
func (s *FibService) ComputeFib(ctx context.Context, n int) (*big.Int, error) {
	start := time.Now()
	defer s.updateComputationMetrics(start, n)

	// Vérification des entrées
	if n < 0 {
		return nil, ErrNegativeInput
	}
	if n > s.config.MaxValue {
		return nil, ErrInputTooLarge
	}

	// Vérification dans le cache
	if result, ok := s.cache.Get(n); ok {
		s.metrics.CacheHits.Add(1)
		return result, nil
	}
	s.metrics.CacheMisses.Add(1)

	// Calcul de Fibonacci
	result, err := s.fibDoubling(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("calcul de Fibonacci: %w", err)
	}

	// Mise à jour du cache
	s.cache.Set(n, result)
	s.metrics.CacheSize.Store(int64(s.cache.(*LRUCache).cache.Len()))

	return result, nil
}

// Handler HTTP pour les métriques
func (s *FibService) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"computation": map[string]interface{}{
			"duration_ms":         float64(s.metrics.ComputationDuration.Load()) / 1e6,
			"avg_compute_time_ms": s.metrics.AvgComputeTime.Load(),
			"total_calculations":  s.metrics.TotalCalculations.Load(),
			"bit_operations":      s.metrics.BitOperationsCount.Load(),
		},
		"memory": map[string]interface{}{
			"allocations":      s.metrics.MemoryAllocations.Load(),
			"peak_usage_bytes": s.metrics.PeakMemoryUsage.Load(),
		},
		"cache": map[string]interface{}{
			"size":           s.metrics.CacheSize.Load(),
			"hits":           s.metrics.CacheHits.Load(),
			"misses":         s.metrics.CacheMisses.Load(),
			"efficiency":     s.metrics.CacheEfficiency.Load(),
			"eviction_count": s.metrics.EvictionCount.Load(),
		},
		"pool": map[string]interface{}{
			"acquisitions": s.metrics.PoolAcquisitions.Load(),
			"misses":       s.metrics.PoolMisses.Load(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// Mise à jour de la fonction main pour ajouter l'endpoint des métriques
func main() {
	// ... configuration existante ...

	// Ajout du handler des métriques
	http.HandleFunc("/metrics", service.handleMetrics)
	http.HandleFunc("/compute", service.handleCompute)

	// Démarrage du serveur avec logging des métriques périodiques
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			service.logger.Printf("Métriques actuelles: Cache efficacité=%.2f%%, Temps moyen de calcul=%.2fms, Mémoire utilisée=%d MB",
				service.metrics.CacheEfficiency.Load()*100,
				service.metrics.AvgComputeTime.Load(),
				service.metrics.PeakMemoryUsage.Load()/1024/1024)
		}
	}()

	log.Printf("Démarrage du serveur sur le port %s", cfg.HTTPPort)
	if err := http.ListenAndServe(cfg.HTTPPort, nil); err != nil {
		log.Fatalf("Erreur serveur HTTP: %v", err)
	}
}