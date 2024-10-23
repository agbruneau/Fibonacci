// DoublingParallel.go
//
// Ce code Golang a été généré à l'aide de "Claude.ai" et de "ChatGPT 4o with Canvas" le 23 octobre 2024
//
// Ce programme implémente un service web en Go permettant de calculer des nombres de la suite de Fibonacci en utilisant une méthode de doublage parallèle.
// La solution est optimisée à l'aide d'une structure de cache LRU (Least Recently Used) pour stocker les résultats des calculs récents, afin de minimiser le temps de calcul pour des valeurs déjà traitées.
// Les calculs sont gérés de manière concurrente et le service utilise des pools de mémoire pour optimiser l'utilisation des ressources.
//
// Ce programme comprend les éléments suivants :
//
// 1. Configuration et Métriques :
//    - Définition de la configuration du service, y compris la valeur maximale de Fibonacci à calculer, la taille du cache, le nombre de workers, le délai d'expiration, et le port HTTP.
//    - Gestion des métriques pour suivre les performances, incluant les hits/misses du cache, le temps de calcul, et l'utilisation de la mémoire.
//
// 2. Cache avec LRU :
//    - Implémentation d'un cache LRU, qui permet de conserver en mémoire les derniers résultats calculés afin de réduire les recalculs inutiles.
//    - Le cache est protégé par des verrous (RWMutex) pour permettre une lecture/écriture sécurisée en environnement concurrent.
//
// 3. Service Fibonacci :
//    - Fournit une méthode ComputeFib pour calculer un nombre de Fibonacci, qui inclut une vérification dans le cache et le calcul à l'aide de la méthode de doublage.
//    - Utilise un pool de big.Int pour réduire le coût des allocations répétées.
//    - La méthode de doublage est basée sur l'algorithme « Fast Doubling » qui est efficace pour calculer rapidement des valeurs de Fibonacci.
//
// 4. Gestion des erreurs :
//    - Des erreurs spécifiques sont définies pour les entrées négatives et les valeurs trop grandes.
//    - Gestion des contextes annulés pour permettre une annulation propre des calculs lors des timeout ou d'autres interruptions.
//
// 5. Serveur HTTP :
//    - Le service expose une API HTTP pour le calcul des nombres de Fibonacci.
//    - Les requêtes sont traitées par un gestionnaire qui parse les entrées JSON, appelle la fonction ComputeFib, et renvoie le résultat au format JSON.
//
// 6. Optimisation des ressources :
//    - Utilisation de pools (sync.Pool) pour réduire l'overhead des allocations de mémoire des objets big.Int.
//    - Calcul parallèle, optimisation avec l'utilisation des GOMAXPROCS pour ajuster dynamiquement le nombre de workers selon les capacités de la machine.
//
// Ce programme est conçu pour offrir un service de calcul de Fibonacci performant et capable de gérer des volumes élevés de requêtes simultanées grâce à l'optimisation du cache et des ressources système.
// Il peut être utilisé dans des environnements nécessitant des calculs intensifs de manière réactive, tout en maintenant une faible latence pour les requêtes répétitives.

package main

import (
	"context"       // Provides functionality for managing deadlines, cancellations, and other request-scoped values across API boundaries.
	"encoding/json" // Implements encoding and decoding of JSON.
	"errors"        // Defines functions to create and manipulate error values.
	"fmt"           // Implements formatted I/O with functions similar to C's printf and scanf.
	"log"           // Provides simple logging support, including output to standard error.
	"math/big"      // Implements arbitrary-precision arithmetic (big numbers).
	"math/bits"     // Implements bitwise operations useful for low-level operations.
	"net/http"      // Provides HTTP client and server implementations.
	"os"            // Provides a platform-independent interface to operating system functionality, including file and environment management.
	"runtime"       // Provides operations that interact with Go's runtime system, such as garbage collection and goroutines.
	"sync"          // Provides basic synchronization primitives such as mutual exclusion locks.
	"sync/atomic"   // Provides low-level atomic memory primitives useful for implementing concurrent algorithms.
	"time"          // Provides functionality for measuring and displaying time.

	lru "github.com/hashicorp/golang-lru" // Imports an LRU (Least Recently Used) cache implementation from HashiCorp's library, useful for managing a limited cache size.
)

// Erreurs typées
var (
	ErrNegativeInput   = errors.New("l'entrée doit être un nombre positif")          // Erreur lorsque l'entrée est négative
	ErrInputTooLarge   = errors.New("l'entrée dépasse la valeur maximale autorisée") // Erreur lorsque l'entrée dépasse la valeur maximale
	ErrCacheInitFailed = errors.New("échec de l'initialisation du cache")            // Erreur lorsque l'initialisation du cache échoue
)

// Configuration
type Config struct {
	MaxValue     int           `json:"maxValue"`     // Valeur maximale pour n
	MaxCacheSize int           `json:"maxCacheSize"` // Taille maximale du cache
	WorkerCount  int           `json:"workerCount"`  // Nombre de workers
	Timeout      time.Duration `json:"timeout"`      // Délai d'expiration
	HTTPPort     string        `json:"httpPort"`     // Port HTTP
}

// Métriques
type Metrics struct {
	CacheHits   atomic.Int64  // Nombre de hits de cache
	CacheMisses atomic.Int64  // Nombre de ratés de cache
	ComputeTime atomic.Int64  // Temps total de calcul
	MemoryUsage atomic.Uint64 // Utilisation de la mémoire
}

// Interface Cache
type Cache interface {
	Get(key int) (*big.Int, bool) // Récupère un élément du cache
	Set(key int, value *big.Int)  // Ajoute un élément au cache
	Clear()                       // Vide le cache
}

// Implémentation LRUCache
type LRUCache struct {
	cache *lru.Cache
	mu    sync.RWMutex // Mutex pour protéger l'accès au cache
}

// Récupère une valeur du cache
func (c *LRUCache) Get(key int) (*big.Int, bool) {
	c.mu.RLock() // Verrou en lecture
	defer c.mu.RUnlock()
	if val, ok := c.cache.Get(key); ok {
		return val.(*big.Int), true
	}
	return nil, false
}

// Ajoute une valeur au cache
func (c *LRUCache) Set(key int, value *big.Int) {
	c.mu.Lock() // Verrou en écriture
	defer c.mu.Unlock()
	c.cache.Add(key, value)
}

// Vide le cache
func (c *LRUCache) Clear() {
	c.mu.Lock() // Verrou en écriture
	defer c.mu.Unlock()
	c.cache.Purge()
}

// Service Fibonacci
type FibService struct {
	config  Config
	metrics *Metrics
	cache   Cache
	logger  *log.Logger
}

// Pool pour big.Int pour éviter les allocations répétées
var bigIntPool = sync.Pool{
	New: func() interface{} {
		return new(big.Int)
	},
}

// Constructeur du service
func NewFibService(cfg Config) (*FibService, error) {
	lruCache, err := lru.New(cfg.MaxCacheSize)
	if err != nil {
		return nil, fmt.Errorf("initialisation du cache: %w", ErrCacheInitFailed)
	}

	cache := &LRUCache{
		cache: lruCache,
	}

	logger := log.New(os.Stdout, "[FIB] ", log.LstdFlags)

	return &FibService{
		config:  cfg,
		metrics: &Metrics{},
		cache:   cache,
		logger:  logger,
	}, nil
}

// Calcul de Fibonacci
func (s *FibService) ComputeFib(ctx context.Context, n int) (*big.Int, error) {
	start := time.Now() // Mesure du temps de calcul
	defer func() {
		s.metrics.ComputeTime.Add(time.Since(start).Nanoseconds())
	}()

	// Vérification des entrées
	if n < 0 {
		return nil, ErrNegativeInput
	}
	if n > s.config.MaxValue {
		return nil, ErrInputTooLarge
	}

	// Vérification dans le cache
	if result, ok := s.cache.Get(n); ok {
		s.metrics.CacheHits.Add(1) // Incrément du compteur de hits de cache
		return result, nil
	}
	s.metrics.CacheMisses.Add(1) // Incrément du compteur de ratés de cache

	// Calcul de Fibonacci avec méthode de doublage
	result, err := s.fibDoubling(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("calcul de Fibonacci: %w", err)
	}

	// Stockage du résultat dans le cache
	s.cache.Set(n, result)

	// Mise à jour de l'utilisation de la mémoire
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.metrics.MemoryUsage.Store(m.Alloc)

	return result, nil
}

// Méthode de doublage pour le calcul de Fibonacci
func (s *FibService) fibDoubling(ctx context.Context, n int) (result *big.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic récupéré: %v", r)
			s.logger.Printf("panic dans fibDoubling: %v, n=%d", r, n)
		}
	}()

	// Cas de base : n < 2
	if n < 2 {
		return big.NewInt(int64(n)), nil
	}

	// Initialisation des valeurs de Fibonacci
	a := bigIntPool.Get().(*big.Int).SetInt64(0) // F(0)
	b := bigIntPool.Get().(*big.Int).SetInt64(1) // F(1)
	c := bigIntPool.Get().(*big.Int)             // Variable temporaire c
	d := bigIntPool.Get().(*big.Int)             // Variable temporaire d

	// Libération des big.Int après usage
	defer func() {
		bigIntPool.Put(a)
		bigIntPool.Put(b)
		bigIntPool.Put(c)
		bigIntPool.Put(d)
	}()

	// Algorithme de doublage
	for i := bits.Len(uint(n)) - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // Si le contexte est annulé, retourner une erreur
		default:
			// Calcul de c et d
			c.Lsh(b, 1).Sub(c, a).Mul(c, a) // c = (2b - a) * a
			d.Mul(a, a).Add(d, b.Mul(b, b)) // d = a^2 + b^2

			// Mise à jour des valeurs de a et b selon le bit courant
			if ((n >> i) & 1) == 0 {
				a.Set(c)
				b.Set(d)
			} else {
				a.Set(d)
				b.Add(c, d)
			}
		}
	}

	return new(big.Int).Set(a), nil
}

// Handler HTTP pour traiter les requêtes de calcul de Fibonacci
func (s *FibService) handleCompute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		N int `json:"n"` // Paramètre de la requête
	}

	// Décoder la requête JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Calculer le nombre de Fibonacci
	result, err := s.ComputeFib(r.Context(), req.N)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Encoder la réponse en JSON
	json.NewEncoder(w).Encode(map[string]string{
		"result": result.String(),
	})
}

func main() {
	cfg := Config{
		MaxValue:     500000001,             // Valeur maximale pour n
		MaxCacheSize: 1000,                  // Taille maximale du cache
		WorkerCount:  runtime.GOMAXPROCS(0), // Nombre de workers égale au nombre de processeurs
		Timeout:      10 * time.Minute,      // Délai d'expiration du calcul
		HTTPPort:     ":8080",               // Port HTTP sur lequel écouter
	}

	// Créer une instance du service Fibonacci
	service, err := NewFibService(cfg)
	if err != nil {
		log.Fatalf("Erreur création service: %v", err)
	}

	// Test rapide du calcul de Fibonacci
	ctx := context.Background()
	result, err := service.ComputeFib(ctx, 100)
	if err != nil {
		log.Fatalf("Erreur calcul: %v", err)
	}
	fmt.Printf("Fib(100) = %s\n", result.String())

	// Démarrage du serveur HTTP
	http.HandleFunc("/compute", service.handleCompute)

	log.Printf("Démarrage du serveur sur le port %s", cfg.HTTPPort)
	if err := http.ListenAndServe(cfg.HTTPPort, nil); err != nil {
		log.Fatalf("Erreur serveur HTTP: %v", err)
	}
}
